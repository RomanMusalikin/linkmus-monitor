# 🖥️ LinkMus Monitor

> Система мониторинга гетерогенной сетевой инфраструктуры в реальном времени.  
> Курсовая работа по дисциплине **«Телекоммуникационные технологии в отраслях транспортно-дорожного комплекса»** — МАДИ, группа 2бИТС1.

---

## 📋 Содержание

1. [О проекте](#-о-проекте)
2. [Архитектура](#-архитектура)
3. [Технологический стек](#-технологический-стек)
4. [Инфраструктура стенда](#-инфраструктура-стенда)
5. [Собираемые метрики](#-собираемые-метрики)
6. [Структура проекта](#-структура-проекта)
7. [Быстрый старт](#-быстрый-старт)
8. [Сборка и деплой](#-сборка-и-деплой)
9. [API-контракт](#-api-контракт)
10. [Схема базы данных](#-схема-базы-данных)
11. [Добавление новой метрики](#-добавление-новой-метрики)
12. [Пороги алертов](#-пороги-алертов)
13. [Авторство](#-авторство)

---

## 🎯 О проекте

**LinkMus Monitor** — кроссплатформенная система мониторинга состояния узлов корпоративной сети. Разработана для гетерогенной среды, где одновременно работают машины на Windows, Astra Linux, Ubuntu и РЕД ОС.

**Ключевые возможности:**
- Push-модель: агенты сами отправляют метрики на сервер, не требуя входящих подключений
- Кросс-платформенность: один бинарник агента для Linux (amd64), другой для Windows (amd64)
- Богатый набор метрик: CPU по ядрам, RAM+Swap, все диски, disk I/O, все сетевые интерфейсы, топ процессов по CPU и RAM, статус служб
- Автоматическое определение офлайн-узлов: если агент не присылал данные более 30 секунд — узел помечается недоступным
- Живые графики с абсолютным временем на осях
- Адаптивный интерфейс: тёмная тема, плавная боковая панель, мобильная поддержка

---

## 🏗️ Архитектура

### Модель взаимодействия

```
┌────────────────────────────────────────────────────────────┐
│                        БРАУЗЕР                             │
│  React-приложение (Vite dev или dist/)                     │
│  GET /api/nodes каждые 5 сек  ────────────────────────┐    │
└───────────────────────────────────────────────────────│────┘
                                                        │ JSON
┌───────────────────────────────────────────────────────│────┐
│                  МАСТЕР-СЕРВЕР (Go)                   │    │
│  mon-server  :8080                                    │    │
│                                                       │    │
│  POST /api/metrics ◄── агенты шлют метрики каждые 3с  │    │
│  GET  /api/nodes   ──────────────────────────────────►│    │
│                                                            │
│  SQLite  monitor.db  (накопительная история метрик)        │
└────────────────────────────────────────────────────────────┘
      ▲                ▲                ▲              ▲
      │ HTTP POST      │ HTTP POST      │ HTTP POST    │ HTTP POST
      │ JSON  3с       │ JSON  3с       │ JSON  3с     │ JSON  3с
┌─────┴──────┐  ┌──────┴─────┐  ┌──────┴─────┐  ┌────┴───────┐
│ mon-agent  │  │ mon-agent  │  │ mon-agent  │  │ mon-agent  │
│ Windows    │  │ Astra Linux│  │ Ubuntu     │  │ РЕД ОС    │
│ WMI+gops  │  │ /proc+gops │  │ /proc+gops │  │ /proc+gops│
└────────────┘  └────────────┘  └────────────┘  └────────────┘
```

### Nginx как обратный прокси (production)

```
Internet ──► WAN:8080 ──► DNAT ──► 10.10.10.10:80 ──► Nginx
                                         │
                                    ┌────┴────────────────────┐
                                    │  /api/*  → :8080 (Go)  │
                                    │  /*      → web/dist/   │
                                    └─────────────────────────┘
```

В **dev-режиме** роль прокси берёт Vite (`vite.config.js`): все запросы на `/api/*` автоматически проксируются на `http://localhost:8080`.

---

## 🛠️ Технологический стек

| Компонент | Технология | Описание |
|-----------|-----------|----------|
| **Агент** | Go 1.21+, `gopsutil/v3`, `wmi` | Кросс-компиляция: linux/amd64, windows/amd64 |
| **Сервер** | Go 1.21+, `net/http` | REST API, без фреймворков |
| **База данных** | SQLite (`modernc.org/sqlite`) | Pure-Go, без CGo — удобная кросс-компиляция |
| **Фронтенд** | React 18, Vite, Tailwind CSS v3 | Тёмная тема, адаптивная вёрстка |
| **Графики** | Recharts | AreaChart, линейные графики с градиентом |
| **Иконки** | Lucide React | Консистентный набор SVG-иконок |
| **Роутинг** | React Router v6 | SPA-роутинг: `/` и `/node/:nodeId` |
| **Метрики Linux** | `/proc`, `/sys`, `gopsutil/v3` | CPU, RAM, диск, сеть, процессы |
| **Метрики Windows** | WMI (`Win32_*`), `gopsutil/v3` | CPU, RAM, диск, сеть, службы |
| **Прокси** | Nginx | Обратный прокси на `srv-mon-01` |

---

## 🌐 Инфраструктура стенда

Стенд развёрнут на **VMware Workstation** с вложенной виртуализацией **ESXi 8.0.2**. Все VM подключены к изолированному L2-сегменту `vSwitch-Lab (10.10.10.0/24)`.

### Карта узлов

| Имя ВМ | ОС | IP-адрес | vCPU | RAM | Роль | Метрики |
|--------|----|----|------|-----|------|---------|
| `gw-border-01` | MikroTik RouterOS 7.20.8 | 10.10.10.1 | 1 | 1 ГБ | Шлюз, NAT, DNS, Firewall | SNMP / RouterOS API |
| `srv-mon-01` | Astra Linux CE «Орёл» 2.12 | 10.10.10.10 | 2 | 4 ГБ | **Мастер-сервер + Nginx** | Локальный агент |
| `srv-corp-01` | Windows Server 2022 | 10.10.10.11 | 4 | 4 ГБ | Корпоративный сервер | WMI, SNMP, FSRM |
| `cl-astra-01` | Astra Linux CE «Орёл» 2.12 | 10.10.10.100 | 1 | 2 ГБ | АРМ (DEB) | /proc, /sys |
| `cl-win-01` | Windows 10 Pro | 10.10.10.101 | 2 | 4 ГБ | АРМ (Windows) | WMI, RDP, SMB |
| `cl-ubnt-01` | Ubuntu Desktop 24.04 | 10.10.10.102 | 1 | 2 ГБ | АРМ (DEB) | /proc, /sys |
| `cl-redos-01` | РЕД ОС 8.2 | 10.10.10.103 | 1 | 2 ГБ | АРМ (RPM) | /proc, /sys |

### Топология сети

```
Internet
    │
    │ 192.168.5.0/24  (внешняя сеть ESXi, WAN MikroTik: 192.168.5.144)
    │
[gw-border-01]  MikroTik RouterOS
  ├─ NAT: srcnat masquerade (выход в интернет)
  ├─ DNAT: WAN:8080 → 10.10.10.10:80  (публичный доступ к дашборду)
  └─ DNS: кэширующий резолвер для 10.10.10.0/24
    │
    │ 10.10.10.0/24  (vSwitch-Lab, изолированный L2)
    ├── srv-mon-01   10.10.10.10   ← мастер-сервер + Nginx
    ├── srv-corp-01  10.10.10.11   ← WMI, SNMP, FSRM
    ├── cl-astra-01  10.10.10.100
    ├── cl-win-01    10.10.10.101
    ├── cl-ubnt-01   10.10.10.102
    └── cl-redos-01  10.10.10.103
```

---

## 📊 Собираемые метрики

Агент реализован для обеих платформ с build-тегами (`//go:build windows`, `//go:build linux`). Все дельта-метрики (CPU, сеть, disk I/O) вычисляются между двумя последовательными снимками — первый вызов всегда возвращает 0.

### CPU

| Метрика | Источник Linux | Источник Windows | Описание |
|---------|---------------|-----------------|----------|
| `cpu_usage` | `/proc/stat` δ | `cpu.Times()` δ | Суммарная загрузка, % |
| `cpu_user` | `/proc/stat` δ | `cpu.Times()` δ | User-пространство, % |
| `cpu_system` | `/proc/stat` δ | `cpu.Times()` δ | Kernel-пространство, % |
| `cpu_cores_json` | `/proc/stat` (cpuN) | `cpu.Times(true)` | Загрузка каждого ядра, `[]float64` |
| `cpu_model` | `/proc/cpuinfo` | `cpu.Info()` | Название модели процессора |
| `cpu_freq_mhz` | `cpu.Info()` | `cpu.Info()` | Текущая частота, МГц |
| `load_avg_1/5/15` | `/proc/loadavg` | — (0 на Windows) | Load average |

### Память (RAM)

| Метрика | Источник | Описание |
|---------|----------|----------|
| `ram_usage` | `mem.VirtualMemory()` | Используется, ГБ |
| `ram_total` | `mem.VirtualMemory()` | Всего, ГБ |
| `ram_cached` | `mem.VirtualMemory()` | Кэш страниц, ГБ (Linux) |
| `ram_buffers` | `mem.VirtualMemory()` | Буферы ядра, ГБ (Linux) |
| `swap_used` | `mem.SwapMemory()` | Swap используется, ГБ |
| `swap_total` | `mem.SwapMemory()` | Swap всего, ГБ |

### Диски

| Метрика | Источник | Описание |
|---------|----------|----------|
| `disk_usage` | `disk.Usage("/")` или `("C:\\")` | Заполнение корневого раздела, % |
| `disks_json` | `disk.Partitions()` + `disk.Usage()` | Все разделы: mount, fstype, ГБ, % |
| `disk_read_sec` | `disk.IOCounters()` δ | Суммарное чтение, байт/сек |
| `disk_write_sec` | `disk.IOCounters()` δ | Суммарная запись, байт/сек |

> Linux-коллектор фильтрует виртуальные ФС: `tmpfs`, `devtmpfs`, `sysfs`, `proc`, `cgroup` и монтирования в `/sys`, `/proc`, `/dev`.

### Сеть

| Метрика | Источник | Описание |
|---------|----------|----------|
| `net_interface` | `net.Interfaces()` | Имя основного интерфейса |
| `net_bytes_recv` | `net.IOCounters()` δ | Входящий трафик, байт/сек |
| `net_bytes_sent` | `net.IOCounters()` δ | Исходящий трафик, байт/сек |
| `all_ifaces_json` | `net.IOCounters(true)` δ | Все интерфейсы: имя, recv, sent |

> Основной интерфейс определяется по совпадению IP: открывается UDP-соединение на `8.8.8.8:80` (без реальной отправки), из адреса вычитывается outbound IP, затем ищется интерфейс с этим IP. Если не найден — выбирается интерфейс с максимальным `BytesRecv`.

### Процессы

| Метрика | Источник | Описание |
|---------|----------|----------|
| `processes_json` | `process.Processes()` | Топ-10 по CPU: PID, Name, CPU%, RAM, User |
| `top_mem_json` | `process.Processes()` | Топ-10 по RAM: PID, Name, CPU%, RAM, User |
| `process_count` | `process.Processes()` | Общее количество процессов |

### Службы и система

| Метрика | Источник | Описание |
|---------|----------|----------|
| `rdp_running` | WMI `Win32_Service` (Windows) | Статус службы `TermService` |
| `smb_running` | WMI `Win32_Service` (Windows) | Статус службы `LanmanServer` |
| `uptime` | `host.Info()` | Время работы: «N ч.» |
| `boot_time` | `host.Info()` | Дата последней загрузки |
| `logged_users` | `host.Users()` | Количество залогиненных пользователей |

---

## 📁 Структура проекта

```
linkmus-monitor/
│
├── cmd/
│   ├── agent/
│   │   └── main.go                  # Точка входа агента
│   └── server/
│       └── main.go                  # Точка входа сервера
│
├── configs/
│   └── agent-config.yaml            # URL сервера и интервал отправки
│
├── internal/
│   ├── agent/
│   │   ├── agent.go                 # collectAndSend(): сбор всех метрик + MetricPayload
│   │   ├── config.go                # LoadConfig() → yaml → struct
│   │   └── sender.go                # SendToServer() → HTTP POST JSON
│   │
│   ├── collector/
│   │   ├── common.go                # Общие типы: DiskInfo, NetIfaceInfo
│   │   ├── cpu_windows.go           # CollectCPUBreakdown(), CollectCPUPerCore(), CollectCPUInfo()
│   │   ├── cpu_linux.go             # То же для Linux
│   │   ├── disk_windows.go          # CollectDisk(), CollectAllDisks(), CollectDiskIO()
│   │   ├── disk_linux.go            # То же для Linux (с фильтрацией виртуальных ФС)
│   │   ├── network_windows.go       # CollectNetwork(), CollectAllInterfaces()
│   │   ├── network_linux.go         # То же для Linux
│   │   ├── process_windows.go       # CollectProcesses(), CollectTopMemProcesses(), CollectProcessCount()
│   │   ├── process_linux.go         # То же для Linux
│   │   ├── memory_windows.go        # CollectMemory() (legacy)
│   │   ├── memory_linux.go          # То же для Linux
│   │   ├── services_windows.go      # CollectServices() → WMI Win32_Service (RDP, SMB)
│   │   └── services_linux.go        # Заглушка: return false, false
│   │
│   └── server/
│       ├── server.go                # Run(), handleMetrics(), MetricPayload struct
│       ├── api.go                   # HandleNodes(), NodeSummary, CpuPoint, RamPoint, NetPoint
│       └── storage.go               # InitDB(), MigrateDB(), SaveMetric(), GetLatestNodes()
│
├── monitor.db                       # SQLite-файл (создаётся автоматически)
├── go.mod / go.sum
│
└── web/
    ├── vite.config.js               # Proxy /api/* → localhost:8080
    ├── tailwind.config.js
    ├── package.json
    ├── dist/                        # Production-сборка (npm run build)
    └── src/
        ├── App.jsx                  # Router + Layout (Sidebar + Header)
        ├── main.jsx
        ├── lib/
        │   └── api.js               # fetchNodes() — единственная точка работы с API
        ├── hooks/
        │   ├── useAutoRefresh.js    # Polling через рекурсивный setTimeout
        │   ├── useNodes.js          # useAutoRefresh(fetchNodes, 5000)
        │   └── useNodeDetail.js     # useAutoRefresh(fetchNodeDetail, 5000)
        ├── pages/
        │   ├── Dashboard.jsx        # Сводные карточки + сетка NodeCard
        │   └── NodeDetail.jsx       # Детальная страница: CPU/RAM/Disk/Net/Процессы/Службы
        └── components/
            ├── layout/
            │   ├── Header.jsx       # Время, счётчик онлайн/всего
            │   └── Sidebar.jsx      # Навигация + список узлов + кнопка ☰
            ├── cards/
            │   └── NodeCard.jsx     # Карточка узла: метрики, sparkline, бейджи служб
            ├── charts/
            │   ├── CpuGauge.jsx     # Круговой SVG-индикатор CPU
            │   ├── CpuHistory.jsx   # AreaChart истории CPU (Recharts)
            │   ├── NetworkLines.jsx # AreaChart RX/TX (Recharts)
            │   ├── DiskBars.jsx     # Горизонтальные бары дисков
            │   ├── RamBar.jsx       # Бар памяти
            │   ├── RamPie.jsx       # Круговая диаграмма памяти
            │   └── Sparkline.jsx    # Мини-график для NodeCard
            ├── common/
            │   ├── MetricCard.jsx   # Универсальная карточка с заголовком
            │   └── ProgressBar.jsx  # Полоса прогресса с цветовой индикацией
            ├── status/
            │   ├── ServiceStatus.jsx
            │   └── FsrmQuota.jsx
            └── tables/
                └── ProcessTable.jsx
```

---

## 🚀 Быстрый старт

### Требования

- Go 1.21+
- Node.js 18+, npm

### 1. Клонировать репозиторий

```bash
git clone <url> linkmus-monitor
cd linkmus-monitor
```

### 2. Запустить сервер

```bash
go run ./cmd/server/
# ✅ Сервер запущен на :8080
# ✅ monitor.db создан автоматически
```

### 3. Запустить агент (на той же или другой машине)

```bash
# Убедитесь что в configs/agent-config.yaml указан правильный URL
go run ./cmd/agent/
# 💾 [HOSTNAME] CPU:5.2% | RAM:3.1/7.8GB | Disk:45.2% ...
```

### 4. Запустить фронтенд

```bash
cd web
npm install
npm run dev
# Открыть http://localhost:5173
```

### Полный цикл (три терминала)

```bash
# Терминал 1
go run ./cmd/server/

# Терминал 2
go run ./cmd/agent/

# Терминал 3
cd web && npm run dev
```

### Конфигурация агента (`configs/agent-config.yaml`)

```yaml
server:
  url: "http://127.0.0.1:8080/api/metrics"  # адрес мастер-сервера
  interval: 3s                               # интервал отправки
```

---

## 📦 Сборка и деплой

### Кросс-компиляция агента

```bash
# Linux (для Astra, Ubuntu, РЕД ОС)
GOOS=linux GOARCH=amd64 go build -o mon-agent-linux ./cmd/agent/

# Windows
GOOS=windows GOARCH=amd64 go build -o mon-agent.exe ./cmd/agent/
```

### Сборка сервера

```bash
GOOS=linux GOARCH=amd64 go build -o mon-server ./cmd/server/
```

### Сборка фронтенда

```bash
cd web && npm run build
# Статика в web/dist/
```

---

### Деплой мастер-сервера на `srv-mon-01` (Astra Linux)

```bash
# Скопировать бинарник и статику
scp mon-server user@10.10.10.10:/opt/linkmus-monitor/
scp -r web/dist user@10.10.10.10:/opt/linkmus-monitor/web/

# Systemd unit
sudo tee /etc/systemd/system/mon-server.service << 'EOF'
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
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now mon-server
```

### Nginx (`/etc/nginx/sites-available/linkmus`)

```nginx
server {
    listen 80;
    server_name _;

    root /opt/linkmus-monitor/web/dist;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header Host $host;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/linkmus /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

### Деплой агента на Linux-узлах

```bash
# Скопировать бинарник и конфиг
scp mon-agent-linux user@10.10.10.100:/opt/mon-agent/mon-agent
scp configs/agent-config.yaml user@10.10.10.100:/opt/mon-agent/configs/
chmod +x /opt/mon-agent/mon-agent

# Поправить URL в конфиге на целевом узле:
# url: "http://10.10.10.10/api/metrics"

# Systemd unit
sudo tee /etc/systemd/system/mon-agent.service << 'EOF'
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
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now mon-agent
sudo journalctl -u mon-agent -f
```

---

### Деплой агента на Windows-узлах (через NSSM)

```powershell
# Скопировать mon-agent.exe и configs/ в C:\mon-agent\
# Поправить url в configs\agent-config.yaml

# Установить как системную службу
nssm install MonAgent "C:\mon-agent\mon-agent.exe"
nssm set MonAgent AppDirectory "C:\mon-agent"
nssm set MonAgent AppStdout "C:\mon-agent\logs\agent.log"
nssm set MonAgent AppStderr "C:\mon-agent\logs\agent-err.log"
nssm set MonAgent Start SERVICE_AUTO_START
nssm start MonAgent

# Проверить
nssm status MonAgent
```

---

## 🔌 API-контракт

### `POST /api/metrics` — агент → сервер

Тело запроса (JSON):

```json
{
  "node_name": "srv-corp-01",
  "os": "windows Windows Server 2022",
  "ip": "10.10.10.11",
  "uptime": "72 ч.",
  "boot_time": "13.04.2026 10:00",
  "timestamp": "2026-04-16T14:30:00Z",
  "logged_users": 2,

  "cpu_usage": 42.5,
  "cpu_user": 30.1,
  "cpu_system": 12.4,
  "cpu_model": "Intel(R) Xeon(R) CPU E5-2670",
  "cpu_freq_mhz": 2600.0,
  "cpu_cores_json": "[15.2, 42.5, 8.1, 60.3]",
  "load_avg_1": 0, "load_avg_5": 0, "load_avg_15": 0,

  "ram_usage": 2.7,
  "ram_total": 4.0,
  "ram_cached": 0.5,
  "ram_buffers": 0.1,
  "swap_used": 0.2,
  "swap_total": 2.0,

  "disk_usage": 52.3,
  "disk_read_sec": 1048576.0,
  "disk_write_sec": 524288.0,
  "disks_json": "[{\"mount\":\"C:\\\\\",\"fstype\":\"NTFS\",\"totalGB\":50,\"usedGB\":26.2,\"usedPercent\":52.3}]",

  "rdp_running": true,
  "smb_running": true,

  "net_interface": "Ethernet",
  "net_bytes_recv": 125000.0,
  "net_bytes_sent": 48000.0,
  "all_ifaces_json": "[{\"name\":\"Ethernet\",\"bytesRecvSec\":125000,\"bytesSentSec\":48000}]",

  "process_count": 128,
  "processes_json": "[{\"pid\":4,\"name\":\"System\",\"cpu\":0.5,\"ram\":0.1,\"user\":\"SYSTEM\"}]",
  "top_mem_json": "[{\"pid\":1234,\"name\":\"sqlservr.exe\",\"cpu\":2.1,\"ram\":512.4,\"user\":\"MSSQLSERVER\"}]"
}
```

Ответ: `200 OK`

---

### `GET /api/nodes` — сервер → фронтенд

Ответ — JSON-массив `NodeSummary[]`:

```json
[
  {
    "name": "srv-corp-01",
    "os": "windows Windows Server 2022",
    "ip": "10.10.10.11",
    "online": true,
    "lastSeen": "16.04 14:30:05",
    "uptime": "72 ч.",
    "bootTime": "13.04.2026 10:00",
    "ping": 1,

    "cpu": 42,
    "cpuUser": 30.1,
    "cpuSystem": 12.4,
    "cpuModel": "Intel(R) Xeon(R) CPU E5-2670",
    "cpuFreqMHz": 2600.0,
    "cpuCores": [15.2, 42.5, 8.1, 60.3],
    "loadAvg1": 0, "loadAvg5": 0, "loadAvg15": 0,

    "ramUsed": 2.7,
    "ramTotal": 4.0,
    "ramCached": 0.5,
    "ramBuffers": 0.1,
    "swapUsed": 0.2,
    "swapTotal": 2.0,

    "diskUsage": 52.3,
    "diskReadSec": 1048576.0,
    "diskWriteSec": 524288.0,
    "disks": [
      {"mount": "C:\\", "fstype": "NTFS", "totalGB": 50, "usedGB": 26.2, "usedPercent": 52.3}
    ],

    "rdpRunning": true,
    "smbRunning": true,

    "netInterface": "Ethernet",
    "netRecvSec": 125000.0,
    "netSentSec": 48000.0,
    "allIfaces": [
      {"name": "Ethernet", "bytesRecvSec": 125000, "bytesSentSec": 48000}
    ],

    "processCount": 128,
    "loggedUsers": 2,
    "processes": [
      {"pid": 4, "name": "System", "cpu": 0.5, "ram": 0.1, "user": "SYSTEM"}
    ],
    "topMemProcesses": [
      {"pid": 1234, "name": "sqlservr.exe", "cpu": 2.1, "ram": 512.4, "user": "MSSQLSERVER"}
    ],

    "cpuHistory": [
      {"value": 40, "time": "14:29:45"},
      {"value": 42, "time": "14:29:48"},
      {"value": 42, "time": "14:29:51"}
    ],
    "ramHistory": [
      {"value": 67, "time": "14:29:45"}
    ],
    "netHistory": [
      {"recv": 120000, "sent": 45000, "time": "14:29:45"}
    ]
  }
]
```

**Важные примечания:**
- `online: true` — если последняя метрика поступила менее 30 секунд назад
- `lastSeen` — абсолютное время последней метрики (формат `ДД.ММ ЧЧ:ММ:СС`)
- `cpu` — int (округлено), `ramUsed`/`ramTotal` — float64 (ГБ), `diskUsage` — float64 (%)
- `cpuHistory`, `ramHistory`, `netHistory` — последние 20 точек в хронологическом порядке (от старых к новым), с абсолютным временем на оси X

---

## 🗄️ Схема базы данных

Единственная таблица `metrics` — накопительный лог всех телеметрических данных.

```sql
CREATE TABLE metrics (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Идентификация узла
    node_name       TEXT,          -- hostname
    os              TEXT,          -- "windows Windows Server 2022"
    ip              TEXT,          -- "10.10.10.11"
    uptime          TEXT,          -- "72 ч."
    boot_time       TEXT,          -- "13.04.2026 10:00"
    timestamp       DATETIME,      -- RFC3339, от агента
    logged_users    INTEGER,

    -- CPU
    cpu_usage       REAL,          -- суммарная загрузка, %
    cpu_user        REAL,          -- user-пространство, %
    cpu_system      REAL,          -- system (kernel), %
    cpu_model       TEXT,          -- "Intel(R) Xeon(R)..."
    cpu_freq_mhz    REAL,          -- текущая частота
    cpu_cores_json  TEXT,          -- JSON []float64 — по ядрам

    -- Load average
    load_avg_1      REAL,
    load_avg_5      REAL,
    load_avg_15     REAL,

    -- RAM
    ram_usage       REAL,          -- используется, ГБ
    ram_total       REAL,          -- всего, ГБ
    ram_cached      REAL,          -- кэш, ГБ
    ram_buffers     REAL,          -- буферы, ГБ
    swap_used       REAL,          -- swap используется, ГБ
    swap_total      REAL,          -- swap всего, ГБ

    -- Диск
    disk_usage      REAL,          -- корневой раздел, %
    disk_read_sec   REAL,          -- чтение, байт/сек
    disk_write_sec  REAL,          -- запись, байт/сек
    disks_json      TEXT,          -- JSON []DiskInfo — все разделы

    -- Службы Windows
    rdp_running     BOOLEAN,
    smb_running     BOOLEAN,

    -- Сеть
    net_interface   TEXT,          -- имя основного интерфейса
    net_bytes_recv  REAL,          -- входящий, байт/сек
    net_bytes_sent  REAL,          -- исходящий, байт/сек
    all_ifaces_json TEXT,          -- JSON []NetIfaceInfo — все интерфейсы

    -- Процессы
    process_count   INTEGER,       -- общее количество
    processes_json  TEXT,          -- JSON топ-10 по CPU
    top_mem_json    TEXT           -- JSON топ-10 по RAM
);
```

**Миграция** реализована через `MigrateDB()` — функция выполняет `ALTER TABLE ... ADD COLUMN` для каждой новой колонки. Ошибки игнорируются (если колонка уже существует). Это позволяет безопасно обновлять схему на работающей системе без пересоздания таблицы и потери данных.

**Хранение истории:** каждый вызов агента = одна строка. Запросы истории для графиков: `ORDER BY timestamp DESC LIMIT 20` с последующим разворотом массива (хронологический порядок). Очистка старых данных не реализована — для учебного стенда объём данных незначителен.

---

## ➕ Добавление новой метрики

При добавлении любой новой метрики нужно последовательно обновить **5 слоёв** системы:

```
1. КОЛЛЕКТОР (сбор)
   internal/collector/*_windows.go  — добавить функцию или расширить существующую
   internal/collector/*_linux.go    — аналогично для Linux

2. АГЕНТ (упаковка)
   internal/agent/agent.go          — добавить поле в MetricPayload, вызвать коллектор

3. СЕРВЕР (приём + хранение)
   internal/server/server.go        — добавить то же поле в MetricPayload (зеркально)
   internal/server/storage.go       — MigrateDB(): ALTER TABLE ADD COLUMN
   internal/server/storage.go       — SaveMetric(): добавить в INSERT
   internal/server/storage.go       — GetLatestNodes(): добавить в SELECT + Scan

4. API (отдача на фронт)
   internal/server/api.go           — добавить поле в NodeSummary
   internal/server/storage.go       — заполнить поле при сборке NodeSummary

5. ФРОНТЕНД (отображение)
   web/src/pages/NodeDetail.jsx     — добавить отображение в нужную секцию
   web/src/pages/Dashboard.jsx      — если метрика нужна в сводке
   web/src/components/cards/NodeCard.jsx — если нужна на карточке
```

**Пример — добавление температуры CPU:**

```go
// 1. collector/temp_linux.go
func CollectCPUTemp() float64 {
    data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
    if err != nil { return 0 }
    t, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
    return t / 1000
}

// 2. agent.go — MetricPayload + collectAndSend()
CPUTemp float64 `json:"cpu_temp"`
// payload.CPUTemp = collector.CollectCPUTemp()

// 3. storage.go — MigrateDB
`ALTER TABLE metrics ADD COLUMN cpu_temp REAL DEFAULT 0`

// 4. api.go — NodeSummary
CPUTemp float64 `json:"cpuTemp"`

// 5. NodeDetail.jsx
<InfoRow label="Температура CPU" value={node.cpuTemp > 0 ? `${node.cpuTemp.toFixed(1)} °C` : 'N/A'} />
```

---

## 🚨 Пороги алертов

Цветовая индикация реализована в UI. Алертинг (уведомления) — в планах.

| Метрика | 🟡 Внимание | 🔴 Критично |
|---------|------------|------------|
| CPU | > 60% | > 85% |
| RAM | > 60% | > 85% |
| Disk (раздел) | > 60% | > 85% |
| Swap | > 50% | > 80% |
| Узел недоступен | — | last_seen > 30 сек |
| Служба (RDP/SMB) | — | `running = false` |
| Load avg (Linux) | > cpu_count × 2 | > cpu_count × 4 |
| Disk I/O (ожидание) | > 30% | > 60% |

Цветовая схема во всём интерфейсе:
- `text-emerald-400` — норма (< 60%)
- `text-amber-400` — внимание (60–85%)
- `text-red-400` — критично (> 85%)

---

## 👤 Авторство

| Параметр | Значение |
|----------|---------|
| **Учебное заведение** | МАДИ (Московский автомобильно-дорожный государственный технический университет) |
| **Дисциплина** | Телекоммуникационные технологии в отраслях транспортно-дорожного комплекса |
| **Группа** | 2бИТС1 |
| **Тип работы** | Курсовая работа |
| **Год** | 2026 |

---

*LinkMus Monitor — учебный проект. Не предназначен для использования в production без доработки безопасности (аутентификация, HTTPS, ротация логов БД).*
