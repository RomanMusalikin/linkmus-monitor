# LinkMus Monitor

Система мониторинга гетерогенной сетевой инфраструктуры. Кросс-платформенные агенты (Linux / Windows) отправляют метрики на мастер-сервер (Go + SQLite), который отдаёт их веб-интерфейсу (React + Vite + Tailwind).

Курсовой проект по дисциплине «Телекоммуникационные технологии в отраслях транспортно-дорожного комплекса» — МАДИ, группа 2бИТС1.

---

## Содержание

1. [Архитектура и модель взаимодействия](#1-архитектура-и-модель-взаимодействия)
2. [Инфраструктура лабораторного стенда](#2-инфраструктура-лабораторного-стенда)
3. [Структура директорий](#3-структура-директорий)
4. [Бэкенд (Go)](#4-бэкенд-go)
   - [cmd/ — точки входа](#41-cmd--точки-входа)
   - [internal/agent — агент сбора метрик](#42-internalagent--агент-сбора-метрик)
   - [internal/collector — коллекторы метрик](#43-internalcollector--коллекторы-метрик)
   - [internal/server — мастер-сервер](#44-internalserver--мастер-сервер)
5. [База данных (SQLite)](#5-база-данных-sqlite)
6. [Фронтенд (React)](#6-фронтенд-react)
   - [Точка входа и роутинг](#61-точка-входа-и-роутинг)
   - [Слой работы с API](#62-слой-работы-с-api)
   - [Хуки (hooks)](#63-хуки-hooks)
   - [Страницы (pages)](#64-страницы-pages)
   - [Компоненты (components)](#65-компоненты-components)
7. [Как фронтенд взаимодействует с бэкендом](#7-как-фронтенд-взаимодействует-с-бэкендом)
8. [Полный путь данных: от агента до браузера](#8-полный-путь-данных-от-агента-до-браузера)
9. [API-контракт](#9-api-контракт)
10. [Конфигурация](#10-конфигурация)
11. [Сборка и запуск](#11-сборка-и-запуск)
12. [Деплой на боевые узлы](#12-деплой-на-боевые-узлы)
13. [Go-зависимости](#13-go-зависимости)
14. [Пороги алертов](#14-пороги-алертов)
15. [Правило добавления новой метрики](#15-правило-добавления-новой-метрики)

---

## 1. Архитектура и модель взаимодействия

Система построена по **push-модели**: агент сам инициирует отправку данных, сервер пассивно принимает.

```
┌──────────────────────────────────────────────────────────────┐
│                         БРАУЗЕР                              │
│   React-приложение (Vite dev-server или dist/)               │
│   GET /api/nodes  ──────────────────────────────────────┐    │
└─────────────────────────────────────────────────────────│────┘
                                                          │ JSON
┌──────────────────────────────────────────────────────────────┐
│                    МАСТЕР-СЕРВЕР  (Go)                       │
│   mon-server   :8080                                         │
│                                                              │
│   POST /api/metrics  ◄─── агент отправляет метрики           │
│   GET  /api/nodes    ───► отдаёт список узлов + историю      │
│                                                              │
│   SQLite  monitor.db  (хранит всю телеметрию)                │
└──────────────────────────────────────────────────────────────┘
         ▲                        ▲
         │  HTTP POST JSON        │  HTTP POST JSON
         │  каждые x сек          │  каждые x сек
┌────────┴──────────┐    ┌────────┴──────────┐
│  mon-agent        │    │  mon-agent        │
│  Windows          │    │  Linux            │
│  (WMI / gopsutil) │    │  (/proc, /sys)    │
└───────────────────┘    └───────────────────┘
```

**Nginx** на боевом сервере `srv-mon-01` служит обратным прокси: статика React раздаётся с диска, запросы `/api/*` проксируются на `127.0.0.1:8080`.

В режиме разработки роль прокси берёт на себя **Vite dev-server** (настроен в `vite.config.js`).

---

## 2. Инфраструктура лабораторного стенда

Стенд развёрнут на VMware Workstation с вложенной виртуализацией ESXi 8.0.2. Все виртуальные машины подключены к изолированному L2-сегменту `vSwitch-Lab`.

### Карта узлов

| Имя ВМ | ОС | IP | vCPU | RAM | Роль | Источник метрик |
|--------|----|----|------|-----|------|-----------------|
| gw-border-01 | MikroTik RouterOS 7.20.8 | 10.10.10.1 | 1 | 1 ГБ | Шлюз, NAT, DNS, Firewall | SNMP / RouterOS API |
| srv-mon-01 | Astra Linux CE «Орёл» 2.12 | 10.10.10.10 | 2 | 4 ГБ | **Мастер-сервер мониторинга** | Локальный агент |
| srv-corp-01 | Windows Server 2022 | 10.10.10.11 | 4 | 4 ГБ | Корпоративный сервер | WMI, SNMP, FSRM, Event Log |
| cl-astra-01 | Astra Linux CE «Орёл» 2.12 | 10.10.10.100 | 1 | 2 ГБ | АРМ (DEB) | /proc, /sys |
| cl-win-01 | Windows 10 Pro | 10.10.10.101 | 2 | 4 ГБ | АРМ (Windows) | WMI, RDP/SMB |
| cl-ubnt-01 | Ubuntu Desktop 24.04 | 10.10.10.102 | 1 | 2 ГБ | АРМ (DEB) | /proc, /sys |
| cl-redos-01 | РЕД ОС 8.2 | 10.10.10.103 | 1 | 2 ГБ | АРМ (RPM) | /proc, /sys |

### Сетевая топология

```
Internet
    │
    │ 192.168.5.0/24  (внешняя сеть ESXi)
    │
[gw-border-01]  10.10.10.1
MikroTik CHR
WAN: 192.168.5.144
LAN: 10.10.10.1
NAT: srcnat masquerade
DNAT: WAN:8080 → 10.10.10.10:80  ← публичный доступ к дашборду
    │
    │ 10.10.10.0/24  (vSwitch-Lab, изолированный L2)
    ├── srv-mon-01   10.10.10.10   ← мастер-сервер + Nginx
    ├── srv-corp-01  10.10.10.11
    ├── cl-astra-01  10.10.10.100
    ├── cl-win-01    10.10.10.101
    ├── cl-ubnt-01   10.10.10.102
    └── cl-redos-01  10.10.10.103
```

### Сетевые роли

- **Серверный пул:** `10.10.10.10 – 10.10.10.19`
- **Клиентский пул:** `10.10.10.100 – 10.10.10.150`
- **DNS:** MikroTik (10.10.10.1) — кэширующий резолвер для всей сети
- **Внешний доступ к дашборду:** `http://<WAN_IP>:8080` → DNAT → `10.10.10.10:80` → Nginx → React

---

## 3. Структура директорий

```
linkmus-monitor/
│
├── cmd/                        # Точки входа (исполняемые программы)
│   ├── agent/
│   │   └── main.go             # Запуск агента: вызывает agent.Run()
│   └── server/
│       └── main.go             # Запуск сервера: вызывает server.Run()
│
├── configs/
│   └── agent-config.yaml       # Адрес сервера и интервал отправки агента
│
├── internal/                   # Весь внутренний код (не экспортируется как библиотека)
│   ├── agent/
│   │   ├── agent.go            # Главный цикл: сбор → упаковка → отправка
│   │   ├── config.go           # Чтение agent-config.yaml
│   │   └── sender.go           # HTTP POST на сервер
│   │
│   ├── collector/              # Платформо-специфичный сбор метрик
│   │   ├── common.go           # Интерфейс Collector и базовая структура
│   │   ├── cpu_windows.go      # CPU: gopsutil + ручной расчёт дельты
│   │   ├── memory_windows.go   # RAM и Swap: gopsutil/mem
│   │   ├── disk_windows.go     # Диск: gopsutil/disk (C:\)
│   │   ├── network_windows.go  # Сеть: gopsutil/net, байт/сек по дельте
│   │   ├── process_windows.go  # Топ-10 процессов по CPU: gopsutil/process
│   │   └── services_windows.go # Статус служб RDP и SMB через WMI
│   │
│   └── server/
│       ├── server.go           # HTTP-хендлеры, запуск сервера
│       ├── api.go              # Структуры ответа, GET /api/nodes
│       └── storage.go          # SQLite: инит, миграции, запись, чтение
│
├── monitor.db                  # SQLite-файл базы данных (создаётся автоматически)
│
├── go.mod / go.sum             # Модуль Go: linkmus-monitor
│
└── web/                        # Фронтенд (React + Vite)
    ├── vite.config.js          # Прокси /api/* → localhost:8080
    ├── tailwind.config.js
    ├── package.json
    │
    ├── dist/                   # Собранная статика (npm run build)
    │
    └── src/
        ├── main.jsx            # Точка входа React, монтирует <App />
        ├── App.jsx             # Роутер, Layout (Header + Sidebar + <Routes>)
        │
        ├── lib/
        │   └── api.js          # fetchNodes(), fetchNodeDetail() — fetch-запросы
        │
        ├── hooks/
        │   ├── useAutoRefresh.js   # Базовый хук polling с setTimeout
        │   ├── useNodes.js         # Обёртка: useAutoRefresh(fetchNodes, 5000)
        │   └── useNodeDetail.js    # Обёртка: useAutoRefresh(fetchNodeDetail, 5000)
        │
        ├── pages/
        │   ├── Dashboard.jsx   # Главная: сводные карточки + сетка узлов
        │   └── NodeDetail.jsx  # Детальная страница узла
        │
        └── components/
            ├── cards/
            │   └── NodeCard.jsx        # Карточка узла в сетке Dashboard
            ├── charts/
            │   ├── CpuGauge.jsx        # Круговой индикатор загрузки CPU
            │   ├── CpuHistory.jsx      # AreaChart истории CPU (Recharts)
            │   ├── DiskBars.jsx        # Бары дисков (поддерживает % и GB)
            │   ├── NetworkLines.jsx    # График RX/TX (Recharts LineChart)
            │   ├── RamBar.jsx          # Горизонтальный бар памяти
            │   ├── RamPie.jsx          # Круговая диаграмма памяти
            │   └── Sparkline.jsx       # Мини-график для NodeCard
            ├── common/
            │   ├── MetricCard.jsx      # Универсальная карточка с заголовком
            │   └── ProgressBar.jsx     # Полоса прогресса с цветовой индикацией
            ├── layout/
            │   ├── Header.jsx          # Верхняя панель (кнопка меню, название)
            │   └── Sidebar.jsx         # Боковая навигация (Dashboard, узлы)
            ├── status/
            │   ├── ServiceStatus.jsx   # Индикаторы служб (SSH/RDP/SMB)
            │   └── FsrmQuota.jsx       # Блок FSRM-квот (Windows Server)
            └── tables/
                └── ProcessTable.jsx    # Таблица топ-процессов
```

---

## 4. Бэкенд (Go)

### 4.1 `cmd/` — точки входа

Два самостоятельных бинарника компилируются из одного модуля `linkmus-monitor`:

|         Файл          |             Бинарник          |                Что делает               |
|-----------------------|-------------------------------|-----------------------------------------|
|   `cmd/agent/main.go` | `mon-agent` / `mon-agent.exe` | Запускает агент на мониторируемом узле  |
|  `cmd/server/main.go` | `mon-server`                  | Запускает мастер-сервер на `srv-mon-01` |

Оба файла минимальны — по 5–10 строк. Вся логика вынесена в `internal/`.

---

### 4.2 `internal/agent` — агент сбора метрик

#### `config.go` — загрузка конфигурации

Читает `configs/agent-config.yaml` через `gopkg.in/yaml.v3`. Конфигурация содержит два поля:

```yaml
server:
  url: "http://127.0.0.1:8080/api/metrics"  # куда слать метрики
  interval: 3s                               # как часто
```

Структура `Config` автоматически парсит `interval` как `time.Duration` (строку `"3s"` → 3 секунды).

#### `agent.go` — главный цикл

Функция `Run()` создаёт `time.Ticker` на заданный интервал и в цикле вызывает `collectAndSend()`.

`collectAndSend()` выполняет следующие шаги:

1. **Системная информация** — `gopsutil/host`: hostname, OS, время аптайма
2. **IP-адрес** — `getOutboundIP()`: открывает UDP-соединение на `8.8.8.8:80` (не отправляет пакеты), из адреса соединения вычитывает локальный IP
3. **CPU** — `collector.CollectCPUBreakdown()`: возвращает user%, system%, total%
4. **Load average** — `gopsutil/load` (на Windows всегда 0, на Linux — реальные значения)
5. **RAM и Swap** — `gopsutil/mem`: `VirtualMemory()` и `SwapMemory()`
6. **Диск** — `collector.CollectDisk()`
7. **Службы** — `collector.CollectServices()`: статус RDP и SMB
8. **Сеть** — `collector.CollectNetwork(outboundIP)`: имя интерфейса, байт/сек
9. **Процессы** — `collector.CollectProcesses()`: топ-10 по CPU, сериализуется в JSON-строку

Все данные упаковываются в структуру `MetricPayload` и передаются в `sender.go`.

#### `sender.go` — HTTP-отправка

`SendToServer()` сериализует `MetricPayload` в JSON и делает `http.Post` на URL из конфига. При успехе (HTTP 200) печатает краткую сводку в лог.

---

### 4.3 `internal/collector` — коллекторы метрик

Каждый файл снабжён build-тегом `//go:build windows` (в будущем появятся `_linux.go` версии). Все функции **stateful** — хранят предыдущий снимок в глобальных переменных модуля для расчёта дельт.

#### `cpu_windows.go`

```
Первый вызов: запоминает cpu.Times() → возвращает 0
Следующие вызовы: δuser/δtotal×100, δsystem/δtotal×100, (1-δidle/δtotal)×100
```

Использует `gopsutil/v3/cpu`. Глобальная переменная `prevCPUTimes` хранит предыдущий снимок между вызовами.

#### `memory_windows.go`

Использует `gopsutil/v3/mem`. Возвращает используемую и общую RAM и Swap в байтах. Агент конвертирует в ГБ перед отправкой.

#### `disk_windows.go`

Проверяет использование диска `C:\` через `gopsutil/v3/disk`. Возвращает процент заполнения (0–100).

#### `network_windows.go`

1. Получает счётчики всех интерфейсов через `gopsutil/v3/net`
2. Определяет основной интерфейс: ищет тот, чей IP совпадает с `outboundIP`; если не нашёл — берёт с максимальным `BytesRecv`
3. Рассчитывает скорость: `(currBytes - prevBytes) / dtSeconds`
4. Хранит предыдущий снимок в `prevNetCounters` (map по имени интерфейса) и `prevNetTime`

#### `process_windows.go`

Получает список всех процессов через `gopsutil/v3/process`. Для каждого читает имя, `CPUPercent()`, RSS-память, имя пользователя. Сортирует по CPU desc, обрезает до топ-10.

> Первый вызов всегда вернёт `cpu=0` для всех процессов — gopsutil требует два снимка для расчёта дельты.

#### `services_windows.go`

Через WMI (`github.com/yusufpapurcu/wmi`) делает запрос к `Win32_Service`:
```sql
SELECT State FROM Win32_Service WHERE Name = 'TermService'   -- RDP
SELECT State FROM Win32_Service WHERE Name = 'LanmanServer'  -- SMB
```
Возвращает `true` если `State == "Running"`.

---

### 4.4 `internal/server` — мастер-сервер

#### `server.go` — HTTP-хендлеры

`Run()` инициализирует SQLite (`InitDB`), регистрирует два хендлера и запускает `http.ListenAndServe(":8080")`.

**`POST /api/metrics`** — `handleMetrics()`:
1. Проверяет метод (405 если не POST)
2. Десериализует тело запроса в `MetricPayload`
3. Вызывает `SaveMetric(dbConn, payload)`
4. Печатает строку лога с ключевыми метриками
5. Возвращает HTTP 200

**`GET /api/nodes`** — `HandleNodes()` (в `api.go`):
1. Устанавливает заголовки `Content-Type: application/json` и `Access-Control-Allow-Origin: *`
2. Вызывает `GetLatestNodes(dbConn)`
3. Сериализует результат в JSON

#### `api.go` — структуры ответа

Определяет все структуры, которые уходят на фронтенд:

| Структура | Назначение |
|-----------|-----------|
| `NodeSummary` | Полный набор данных по одному узлу |
| `CpuPoint` | `{value: int}` — одна точка истории CPU |
| `RamPoint` | `{value: int}` — одна точка истории RAM (в %) |
| `NetPoint` | `{recv: float64, sent: float64}` — одна точка истории сети |
| `ProcessInfo` | `{pid, name, cpu, ram, user}` — один процесс |

`NodeSummary` содержит:
- Базовые поля: `name`, `os`, `ip`, `online`, `uptime`
- CPU: `cpu` (int, %), `cpuUser`, `cpuSystem`, `loadAvg1/5/15`
- RAM: `ramUsed`, `ramTotal`, `ramCached`, `ramBuffers`, `swapUsed`, `swapTotal` (все в ГБ)
- Диск: `diskUsage` (%)
- Сеть: `netInterface`, `netRecvSec`, `netSentSec` (байт/сек)
- Службы: `rdpRunning`, `smbRunning` (bool)
- Истории: `cpuHistory[]`, `ramHistory[]`, `netHistory[]` (последние 20 точек)
- Процессы: `processes[]`

#### `storage.go` — работа с SQLite

**`InitDB(filepath)`** — открывает (или создаёт) файл `monitor.db`, создаёт таблицу `metrics` и вызывает `MigrateDB`.

**`MigrateDB(db)`** — добавляет новые колонки к существующей таблице через `ALTER TABLE ... ADD COLUMN`. Ошибки при повторном выполнении (колонка уже существует) игнорируются — это позволяет безопасно обновлять схему без потери данных.

**`SaveMetric(db, payload)`** — вставляет одну строку в таблицу `metrics`. Каждый вызов агента = одна строка в БД.

**`GetLatestNodes(db)`** — основная функция чтения:
1. `SELECT DISTINCT node_name FROM metrics` — получает список всех узлов, которые когда-либо отправляли данные
2. Для каждого узла — `SELECT ... ORDER BY timestamp DESC LIMIT 1` — последняя строка (актуальное состояние)
3. `queryCPUHistory` — последние 20 значений `cpu_usage` в хронологическом порядке (DESC LIMIT 20, затем разворот)
4. `queryRAMHistory` — последние 20 точек RAM в процентах: `used/total*100`
5. `queryNetHistory` — последние 20 пар `(recv, sent)`
6. Парсит `processes_json` из JSON-строки в `[]ProcessInfo`
7. Собирает `NodeSummary` и добавляет в результирующий слайс

---

## 5. База данных (SQLite)

Файл `monitor.db` в корне проекта. Используется `modernc.org/sqlite` — pure-Go реализация без CGo, что позволяет кросс-компилировать сервер под Linux без установленного `gcc`.

### Таблица `metrics`

```sql
CREATE TABLE IF NOT EXISTS metrics (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    node_name      TEXT,        -- hostname агента
    os             TEXT,        -- "windows Windows 10 Pro"
    ip             TEXT,        -- "10.10.10.101"
    uptime         TEXT,        -- "36 ч."
    timestamp      DATETIME,    -- RFC3339 от агента

    -- CPU
    cpu_usage      REAL,        -- суммарная загрузка CPU, %
    cpu_user       REAL,        -- user%, добавлено миграцией
    cpu_system     REAL,        -- system%, добавлено миграцией
    load_avg_1     REAL,        -- load average 1m (Linux)
    load_avg_5     REAL,
    load_avg_15    REAL,

    -- RAM
    ram_usage      REAL,        -- использовано, ГБ
    ram_total      REAL,        -- всего, ГБ
    ram_cached     REAL,        -- кэш, ГБ
    ram_buffers    REAL,        -- буферы, ГБ
    swap_used      REAL,        -- Swap используется, ГБ
    swap_total     REAL,        -- Swap всего, ГБ

    -- Диск
    disk_usage     REAL,        -- заполнение C:\ или /, %

    -- Службы Windows
    rdp_running    BOOLEAN,
    smb_running    BOOLEAN,

    -- Сеть
    net_interface  TEXT,        -- имя интерфейса
    net_bytes_recv REAL,        -- входящий трафик, байт/сек
    net_bytes_sent REAL,        -- исходящий трафик, байт/сек

    -- Процессы
    processes_json TEXT         -- JSON-массив топ-10 процессов
);
```

Данные накапливаются бесконечно (удаления нет). Запросы истории делают `ORDER BY timestamp DESC LIMIT 20`, что эффективно при наличии индекса. На большом объёме данных потребуется периодическая очистка или партиционирование.

---

## 6. Фронтенд (React)

Стек: **React 18**, **Vite**, **Tailwind CSS v3**, **Recharts** (графики), **React Router v6**, **Lucide React** (иконки).

### 6.1 Точка входа и роутинг

**`src/main.jsx`** — монтирует `<App />` в `#root`.

**`src/App.jsx`** — корневой компонент:
- Оборачивает всё в `<BrowserRouter>`
- Управляет состоянием сайдбара (`isSidebarOpen`)
- Рендерит `<Sidebar>` и `<Header>` вне маршрутов (они всегда видны)
- Определяет два маршрута:
  - `/` → `<Dashboard />`
  - `/node/:nodeId` → `<NodeDetail />` (`:nodeId` = hostname узла)
- На мобилках при открытом сайдбаре показывает полупрозрачный overlay, клик по которому закрывает меню

### 6.2 Слой работы с API

**`src/lib/api.js`** — единственный файл, который знает об URL-адресах:

```js
const API_BASE = '/api';

fetchNodes()       // GET /api/nodes → массив NodeSummary
fetchNodeDetail(nodeId) // GET /api/nodes/:nodeId → один NodeSummary (пока не реализован на бэке)
```

Все запросы идут на относительный путь `/api/...`. В dev-режиме Vite проксирует их на `http://localhost:8080`, в production Nginx проксирует на `127.0.0.1:8080`.

### 6.3 Хуки (hooks)

#### `useAutoRefresh.js` — базовый хук polling

```
fetchFn: функция, возвращающая Promise
intervalMs: интервал опроса (по умолчанию 5000 мс)
→ возвращает { data, loading, error }
```

Реализован через рекурсивный `setTimeout` (не `setInterval`), что гарантирует: следующий запрос начнётся только после завершения предыдущего. Это предотвращает накопление параллельных запросов при медленной сети.

При размонтировании компонента флаг `isMounted` предотвращает обновление стейта после уничтожения компонента.

#### `useNodes.js`

```js
export function useNodes() {
  return useAutoRefresh(fetchNodes, 5000);
}
```

Простая обёртка. Используется в `Dashboard` и `NodeDetail`.

#### `useNodeDetail.js`

Аналогичная обёртка для `fetchNodeDetail`. Пока фактически не используется — `NodeDetail` использует `useNodes()` и ищет нужный узел через `nodes.find(n => n.name === nodeId)`.

### 6.4 Страницы (pages)

#### `Dashboard.jsx`

Получает данные через `useNodes()`. Вычисляет агрегированные значения на стороне клиента:

- `online` / `offline` — фильтрация по `node.online`
- `avgCPU` — среднее `node.cpu` по онлайн-узлам
- `avgRAM` — среднее `(ramUsed/ramTotal)*100` по онлайн-узлам
- `avgDisk` — среднее `node.diskUsage` по онлайн-узлам

Рендерит:
1. **5 сводных карточек** (`StatCard`) — узлы онлайн/оффлайн, CPU/RAM/Disk средние
2. **Сетку карточек узлов** — `<NodeCard>` для каждого элемента массива

Цвета карточек динамически меняются: зелёный (<60%) → жёлтый (60–80%) → красный (>80%).

#### `NodeDetail.jsx`

Получает данные через `useNodes()`, находит нужный узел: `nodes.find(n => n.name === nodeId)`. Где `nodeId` — параметр маршрута из URL (`/node/HOSTNAME`).

Определяет тип ОС: `isWindows = node.os?.toLowerCase().includes('windows')`.

Рендерит секции (в сетке `xl:grid-cols-3`):
1. **Хлебные крошки + заголовок** — имя, статус online/offline, OS, IP, uptime
2. **4 быстрых показателя** — CPU%, RAM%, Disk%, скорость сети
3. **CPU** (2 колонки) — `CpuGauge` + детализация user/system/load avg + `CpuHistory`
4. **RAM** (1 колонка) — `ProgressBar` + детализация + Swap
5. **Сеть** (2 колонки) — текущие RX/TX скорости + `NetworkLines`
6. **Диски** (1 колонка) — `DiskBars`
7. **Топ процессов** (2 колонки) — таблица с мини-барами CPU
8. **Сервисы** (1 колонка) — `ServiceBadge` для RDP+SMB (Windows) или SSH (Linux) + системная информация

Load average показывается только для не-Windows: `{!isWindows && (...)}`.

### 6.5 Компоненты (components)

#### `cards/NodeCard.jsx`

Карточка узла в сетке Dashboard. Содержит:
- Цветной индикатор статуса, имя, OS, IP
- CPU%, RAM% с цветовой индикацией
- Disk%, uptime
- `<Sparkline>` — мини-график истории CPU
- Бейджи RDP/SMB (только Windows)
- Клик на карточку → `Link` на `/node/:name`

#### `charts/CpuGauge.jsx`

Круговой SVG-индикатор (0–100%). Цвет дуги меняется по порогам.

#### `charts/CpuHistory.jsx`

`AreaChart` из Recharts. Принимает массив `[{time, cpu}]`. Ось X — временные метки вида `-60с`, `-57с`... `0с`. Градиентная заливка под кривой.

#### `charts/NetworkLines.jsx`

`LineChart` из Recharts. Принимает `[{time, recv, sent}]`. Две линии: входящий (cyan) и исходящий (blue) трафик.

#### `charts/DiskBars.jsx`

Принимает массив дисков `[{mount, used, total, unit}]`. Отрисовывает горизонтальные бары. `unit: '%'` — значение уже в процентах; `unit: 'GB'` — рассчитывает `used/total*100`.

#### `charts/Sparkline.jsx`

Миниатюрный `LineChart` без осей и подписей. Используется внутри `NodeCard` для компактного отображения истории.

#### `common/ProgressBar.jsx`

Горизонтальная полоса. Цвет: зелёный (0–60), жёлтый (60–85), красный (>85).

#### `layout/Sidebar.jsx`

Левая навигационная панель. Содержит ссылку на Dashboard и динамический список узлов (получает через `useNodes()`). Каждый узел — ссылка на `/node/:name` с цветным индикатором статуса. На мобилках скрывается (абсолютное позиционирование + z-index).

#### `layout/Header.jsx`

Верхняя панель. Кнопка-гамбургер для переключения сайдбара, название системы, индикатор последнего обновления.

---

## 7. Как фронтенд взаимодействует с бэкендом

### В режиме разработки (Vite)

```
Браузер → http://localhost:5173/api/nodes
         ↓ (Vite proxy, vite.config.js)
Go-сервер → http://localhost:8080/api/nodes
```

Конфигурация прокси в `vite.config.js`:
```js
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    }
  }
}
```

Это позволяет фронтенду и бэкенду работать на разных портах без CORS-проблем.

### В production (Nginx)

```
Браузер → http://10.10.10.10/api/nodes
         ↓ (Nginx proxy_pass)
Go-сервер → http://127.0.0.1:8080/api/nodes

Браузер → http://10.10.10.10/
         ↓ (Nginx static files)
/opt/linkmus-monitor/web/dist/index.html
```

### Формат данных

Фронтенд получает **массив** `NodeSummary[]` от `GET /api/nodes`. Оба компонента (`Dashboard` и `NodeDetail`) используют один и тот же хук `useNodes()` и работают с одним запросом. `NodeDetail` просто фильтрует нужный узел из массива по имени.

Автоматическое обновление происходит каждые **5 секунд** (интервал в `useNodes.js`), агент отправляет данные каждые **3 секунды** (интервал в `agent-config.yaml`). Таким образом, данные на экране обновляются не позже чем через 8 секунд после изменения на узле.

---

## 8. Полный путь данных: от агента до браузера

```
[Узел (Windows/Linux)]
  │
  │  1. Каждые 3 сек: collectAndSend()
  │     - cpu_windows.go: δ(cpu_times) → user%, system%, total%
  │     - memory_windows.go: VirtualMemory() → used/total ГБ
  │     - disk_windows.go: Usage("C:\\") → %
  │     - network_windows.go: δ(IOCounters) / dt → байт/сек
  │     - process_windows.go: top-10 по CPU → JSON-строка
  │     - services_windows.go: WMI Win32_Service → rdp/smb bool
  │
  │  2. sender.go: JSON.Marshal(MetricPayload) → HTTP POST /api/metrics
  │
  ▼
[mon-server :8080]
  │
  │  3. handleMetrics(): json.Decode(r.Body) → MetricPayload
  │
  │  4. SaveMetric(): INSERT INTO metrics (...) VALUES (...)
  │     Каждый вызов агента = 1 новая строка в SQLite
  │
  │  5. GET /api/nodes (каждые 5 сек от браузера)
  │     GetLatestNodes():
  │     - SELECT DISTINCT node_name → список узлов
  │     - Для каждого:
  │       - SELECT ... LIMIT 1 ORDER BY timestamp DESC → последняя метрика
  │       - SELECT cpu_usage ... LIMIT 20 → 20 точек истории (разворот)
  │       - SELECT ram_usage, ram_total ... LIMIT 20 → история RAM %
  │       - SELECT net_bytes_recv, net_bytes_sent ... LIMIT 20 → история сети
  │       - json.Unmarshal(processes_json) → []ProcessInfo
  │     - Собирает NodeSummary для каждого узла
  │
  │  6. json.NewEncoder(w).Encode(nodes) → JSON-ответ
  │
  ▼
[Браузер]
  │
  │  7. useAutoRefresh → fetch('/api/nodes') → NodeSummary[]
  │
  │  8. Dashboard: nodes.map(n => <NodeCard node={n} />)
  │     NodeDetail: nodes.find(n => n.name === nodeId)
  │
  │  9. Recharts рендерит cpuHistory, ramHistory, netHistory
  │     CpuGauge рендерит текущую загрузку
  │     ProgressBar рендерит RAM, Disk
  │
  ▼
[Экран пользователя — живые данные]
```

---

## 9. API-контракт

### `POST /api/metrics`

Агент → Сервер. Тело запроса — JSON:

```json
{
  "node_name": "DESKTOP-ABC",
  "os": "windows Windows 10 Pro",
  "ip": "10.10.10.101",
  "uptime": "36 ч.",
  "timestamp": "2026-04-15T14:30:00Z",
  "cpu_usage": 42.5,
  "cpu_user": 30.1,
  "cpu_system": 12.4,
  "load_avg_1": 0,
  "load_avg_5": 0,
  "load_avg_15": 0,
  "ram_usage": 2.7,
  "ram_total": 4.0,
  "ram_cached": 0.5,
  "ram_buffers": 0.1,
  "swap_used": 0.2,
  "swap_total": 2.0,
  "disk_usage": 52.3,
  "rdp_running": true,
  "smb_running": true,
  "net_interface": "Ethernet",
  "net_bytes_recv": 12345.6,
  "net_bytes_sent": 6789.0,
  "processes_json": "[{\"pid\":4,\"name\":\"System\",\"cpu\":0.5,\"ram\":0.1,\"user\":\"SYSTEM\"}]"
}
```

Ответ: `200 OK` (пустое тело).

### `GET /api/nodes`

Сервер → Фронтенд. Ответ — JSON-массив:

```json
[
  {
    "name": "DESKTOP-ABC",
    "os": "windows Windows 10 Pro",
    "ip": "10.10.10.101",
    "online": true,
    "cpu": 42,
    "cpuUser": 30.1,
    "cpuSystem": 12.4,
    "loadAvg1": 0, "loadAvg5": 0, "loadAvg15": 0,
    "ramUsed": 2.7,
    "ramTotal": 4.0,
    "ramCached": 0.5,
    "ramBuffers": 0.1,
    "swapUsed": 0.2,
    "swapTotal": 2.0,
    "diskUsage": 52.3,
    "rdpRunning": true,
    "smbRunning": true,
    "uptime": "36 ч.",
    "ping": 1,
    "netInterface": "Ethernet",
    "netRecvSec": 12345.6,
    "netSentSec": 6789.0,
    "cpuHistory": [{"value": 40}, {"value": 41}, {"value": 42}],
    "ramHistory": [{"value": 65}, {"value": 66}, {"value": 67}],
    "netHistory": [{"recv": 11000, "sent": 6000}, {"recv": 12345, "sent": 6789}],
    "processes": [
      {"pid": 4, "name": "System", "cpu": 0.5, "ram": 0.1, "user": "SYSTEM"}
    ]
  }
]
```

Примечания:
- `cpu` — int (округлено от float64)
- `ramUsed`, `ramTotal` — float64, ГБ
- `diskUsage` — float64, % (не ГБ!)
- `netRecvSec`, `netSentSec` — байт/сек
- `cpuHistory` — от старых к новым, последние 20 точек
- `online` — всегда `true` (логика offline пока не реализована)
- `ping` — всегда `1` (заглушка)

---

## 10. Конфигурация

### `configs/agent-config.yaml`

```yaml
server:
  url: "http://127.0.0.1:8080/api/metrics"
  interval: 3s
```

На продакшн-узлах `url` меняется на `http://10.10.10.10:8080/api/metrics` (или через Nginx — `http://10.10.10.10/api/metrics`).

### Переменные окружения

Нет. Конфигурация только через YAML-файл.

### Путь к конфигу

Агент ищет конфиг по относительному пути `configs/agent-config.yaml` от рабочей директории. При запуске через systemd/NSSM нужно либо задать `WorkingDirectory`, либо передать путь аргументом.

---

## 11. Сборка и запуск

### Зависимости

- Go 1.21+
- Node.js 18+ / npm

### Агент

```bash
# Windows (из корня проекта)
GOOS=windows GOARCH=amd64 go build -o mon-agent.exe ./cmd/agent/

# Linux
GOOS=linux GOARCH=amd64 go build -o mon-agent-linux ./cmd/agent/
```

Скопировать на целевой узел вместе с `configs/agent-config.yaml`.

### Сервер

```bash
go build -o mon-server ./cmd/server/
./mon-server
# Слушает :8080, создаёт monitor.db в текущей директории
```

### Фронтенд (разработка)

```bash
cd web
npm install
npm run dev
# Vite dev-server: http://localhost:5173
# /api/* проксируется на http://localhost:8080
```

### Фронтенд (production)

```bash
cd web
npm run build
# Статика в web/dist/ — раздаётся через Nginx
```

### Быстрый старт для разработки

```bash
# Терминал 1: сервер
go run ./cmd/server/

# Терминал 2: агент (собирает метрики с текущей машины)
go run ./cmd/agent/

# Терминал 3: фронтенд
cd web && npm run dev
# Открыть http://localhost:5173
```

---

## 12. Деплой на боевые узлы

### Мастер-сервер (srv-mon-01, Astra Linux)

```bash
# 1. Собрать бинарник сервера под Linux
GOOS=linux GOARCH=amd64 go build -o mon-server ./cmd/server/

# 2. Скопировать на сервер
scp mon-server user@10.10.10.10:/opt/linkmus-monitor/

# 3. Собрать фронтенд и скопировать
cd web && npm run build
scp -r dist/ user@10.10.10.10:/opt/linkmus-monitor/web/

# 4. Запустить сервер (фоново или через systemd)
./mon-server
```

#### Systemd unit для сервера (`/etc/systemd/system/mon-server.service`)

```ini
[Unit]
Description=LinkMus Monitor Server
After=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/linkmus-monitor
ExecStart=/opt/linkmus-monitor/mon-server
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now mon-server
systemctl status mon-server
```

#### Nginx (`/etc/nginx/sites-available/linkmus`)

```nginx
server {
    listen 80;
    server_name _;

    # Статика React (собранный dist/)
    root /opt/linkmus-monitor/web/dist;
    index index.html;

    # API проксируется на Go-сервер
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $host;
    }

    # SPA: все маршруты отдают index.html
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

```bash
ln -s /etc/nginx/sites-available/linkmus /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

---

### Агент на Linux-узлах (Astra / Ubuntu / РЕД ОС)

```bash
# 1. Скопировать бинарник и конфиг
scp mon-agent-linux user@10.10.10.100:/opt/mon-agent/mon-agent
scp configs/agent-config.yaml user@10.10.10.100:/opt/mon-agent/configs/

# 2. Поправить URL сервера в конфиге
# url: "http://10.10.10.10/api/metrics"
```

#### Systemd unit для агента (`/etc/systemd/system/mon-agent.service`)

```ini
[Unit]
Description=LinkMus Monitor Agent
After=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/mon-agent
ExecStart=/opt/mon-agent/mon-agent
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable --now mon-agent
journalctl -u mon-agent -f   # следить за логом агента
```

---

### Агент на Windows-узлах (cl-win-01, srv-corp-01)

```powershell
# 1. Скопировать mon-agent.exe и configs/ в C:\mon-agent\

# 2. Установить как службу через NSSM (Non-Sucking Service Manager)
nssm install MonAgent "C:\mon-agent\mon-agent.exe"
nssm set MonAgent AppDirectory "C:\mon-agent"
nssm set MonAgent Start SERVICE_AUTO_START
nssm start MonAgent

# Проверить статус
nssm status MonAgent

# Посмотреть лог (NSSM пишет stdout в файл)
nssm set MonAgent AppStdout "C:\mon-agent\logs\agent.log"
nssm set MonAgent AppStderr "C:\mon-agent\logs\agent-err.log"
```

Поправить `C:\mon-agent\configs\agent-config.yaml`:
```yaml
server:
  url: "http://10.10.10.10/api/metrics"
  interval: 3s
```

---

## 13. Go-зависимости

Файл `go.mod` объявляет единый модуль `linkmus-monitor`. Все зависимости — pure-Go или с минимальными требованиями к системе.

| Пакет | Версия | Назначение | Используется в |
|-------|--------|-----------|----------------|
| `github.com/shirou/gopsutil/v3` | v3.x | CPU, RAM, диск, сеть, процессы, хост-инфо | `internal/collector/*`, `internal/agent/agent.go` |
| `github.com/yusufpapurcu/wmi` | v1.x | WMI-запросы к Win32_Service на Windows | `internal/collector/services_windows.go` |
| `github.com/go-ole/go-ole` | v1.x | COM-инициализация (транзитивная зависимость wmi) | — |
| `gopkg.in/yaml.v3` | v3.x | Парсинг `agent-config.yaml` | `internal/agent/config.go` |
| `modernc.org/sqlite` | v1.x | Pure-Go SQLite без CGo | `internal/server/storage.go` |

#### Почему `modernc.org/sqlite`, а не `mattn/go-sqlite3`?

`mattn/go-sqlite3` требует CGo и установленного `gcc` для компиляции. Это делает кросс-компиляцию под Linux с Windows крайне неудобной. `modernc.org/sqlite` — транспилированная в Go версия оригинального SQLite C-кода, работает без CGo, компилируется под любую платформу стандартным `go build`.

---

## 14. Пороги алертов

Система готовится к реализации алертинга. Запланированные пороги:

| Метрика | Warning | Critical | Примечание |
|---------|---------|----------|-----------|
| `cpu_percent` | > 80% в течение 5 мин | > 95% в течение 3 мин | Скользящее окно |
| `mem_usage_percent` | > 85% | > 95% | |
| `disk_usage_percent` | > 80% | > 90% | Для каждого раздела |
| `swap_usage_percent` | > 50% | > 80% | |
| `node_offline` | — | `last_seen > 60 сек` | Узел не отвечает |
| `service_down` | — | `reachable = false` дважды подряд | RDP, SMB, SSH |
| `fsrm_quota` | > 80% использования | > 95% или жёсткий лимит | Только srv-corp-01 |
| `disk_iowait` | > 30% | > 60% | Только Linux |
| `load_avg_1m` | > `cpu_count × 2` | > `cpu_count × 4` | Только Linux |

Цветовая индикация уже реализована в UI:
- **Зелёный** (`text-emerald-400`) — норма (< 60%)
- **Жёлтый** (`text-amber-400`) — внимание (60–85%)
- **Красный** (`text-red-400`) — критично (> 85%)

---

## 15. Правило добавления новой метрики

При добавлении любой новой метрики необходимо последовательно обновить **все** перечисленные места. Пропуск любого шага приведёт к тому, что данные будут собираться, но не отображаться (или наоборот).

```
Шаг 1. Агент — структура данных
        internal/agent/agent.go       → добавить поле в MetricPayload
        internal/server/server.go     → добавить то же поле в MetricPayload (зеркальная структура)

Шаг 2. Агент — сбор
        internal/collector/*_windows.go  → реализовать сбор для Windows
        internal/collector/*_linux.go    → реализовать сбор для Linux (когда появится)
        internal/agent/agent.go          → вызвать коллектор, заполнить поле в payload

Шаг 3. Сервер — хранение
        internal/server/storage.go    → добавить колонку в CREATE TABLE или MigrateDB()
        internal/server/storage.go    → добавить поле в INSERT (SaveMetric)
        internal/server/storage.go    → добавить поле в SELECT (GetLatestNodes)

Шаг 4. Сервер — API
        internal/server/api.go        → добавить поле в NodeSummary
        internal/server/storage.go    → заполнить поле при сборке NodeSummary

Шаг 5. Фронтенд — отображение
        web/src/pages/NodeDetail.jsx  → отобразить новое поле (секция / компонент)
        web/src/pages/Dashboard.jsx   → если метрика нужна в сводке Dashboard
        web/src/components/cards/NodeCard.jsx → если нужна на карточке узла
```

**Пример:** добавление `cpu_temperature_celsius`

1. `MetricPayload.CPUTemp float64 json:"cpu_temp"` — в обоих файлах
2. `collector/temp_linux.go` — читать `/sys/class/thermal/thermal_zone0/temp`
3. `ALTER TABLE metrics ADD COLUMN cpu_temp REAL DEFAULT 0` — в MigrateDB
4. `NodeSummary.CPUTemp float64 json:"cpuTemp"` — в api.go
5. Отображение в `NodeDetail.jsx` в блоке CPU: `<InfoRow label="Температура" value={...} />`
