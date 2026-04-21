# LinkMus Monitor

Система мониторинга серверов и рабочих станций для гетерогенных корпоративных сетей. Агенты собирают метрики и отправляют их на центральный сервер, веб-интерфейс отображает состояние всей инфраструктуры в реальном времени.

---

## Возможности

- **Поддержка Linux и Windows** — агент компилируется под amd64/arm64 (Linux) и amd64 (Windows)
- **Дашборд** — карточки всех узлов с CPU, RAM, диском, сетью и статусом онлайн
- **Детальная страница узла** — графики, процессы, сервисы, диски, SNMP, FSRM-квоты
- **История за сутки** — график с 10-минутным усреднением, разрывы показывают периоды офлайна
- **Пробы сервисов** — SSH, RDP, SMB, HTTP, WinRM, DNS с временем отклика
- **Авторизация** — первый запуск создаёт учётную запись, далее вход по логину/паролю
- **Автообновление** — данные обновляются каждые 5 секунд без перезагрузки страницы

---

## Архитектура

```
  Агенты (mon-agent)          Мастер-сервер (mon-server)
  ┌─────────────────┐         ┌──────────────────────────┐
  │ Linux / Windows │──POST──▶│ Go + SQLite               │
  │ сбор метрик     │         │ /api/metrics  (агенты)    │
  │ каждые 10 сек   │         │ /api/nodes    (фронтенд)  │
  └─────────────────┘         │ /api/auth/*               │
                              └────────────┬─────────────┘
                                           │
                              ┌────────────▼─────────────┐
                              │ React + Vite + Recharts   │
                              │ Tailwind CSS, тёмная тема │
                              └──────────────────────────┘
```

**Push-модель:** агент сам инициирует соединение → сервер не требует доступа к агентам, работает через NAT.

---

## Установка сервера (Linux, amd64)

```bash
curl -sSL https://raw.githubusercontent.com/RomanMusalikin/linkmus-monitor/main/install.sh | sudo bash
# Выбрать пункт 1 — Сервер мониторинга
```

Скрипт:
- скачивает бинарник `mon-server` и фронтенд из последнего GitHub Release
- создаёт systemd-службу `mon-server` (порт 8080)
- настраивает Nginx как реверс-прокси (порт 80), если установлен
- при повторном запуске — обновляет до новой версии

После установки откройте `http://<IP-сервера>` — при первом входе система предложит создать учётную запись администратора.

---

## Установка агента

### Linux (amd64 / arm64)

```bash
curl -sSL https://raw.githubusercontent.com/RomanMusalikin/linkmus-monitor/main/install.sh | sudo bash
# Выбрать пункт 2 — Агент Linux
```

Скрипт спросит URL сервера и интервал отправки, создаст systemd-службу `mon-agent`.

### Windows (amd64)

Запустить PowerShell от имени администратора:

```powershell
powershell -ExecutionPolicy Bypass -Command "& { iwr https://raw.githubusercontent.com/RomanMusalikin/linkmus-monitor/main/install-agent.ps1 | iex }"
```

Скрипт скачает агент, запросит URL сервера, зарегистрирует службу `MonAgent` через NSSM с автозапуском.

---

## Метрики агента

| Категория | Метрики |
|-----------|---------|
| CPU | загрузка %, user/system/iowait/steal %, частота, модель, загрузка по ядрам, load avg (Linux) |
| RAM | использовано/всего ГБ, кэш, буферы, swap |
| Диски | процент, ГБ по каждому разделу, чтение/запись МБ/с, очередь I/O |
| Сеть | входящий/исходящий трафик Б/с по интерфейсу, TCP-соединения |
| Процессы | top-10 по CPU и по RAM (PID, имя, %, МБ, пользователь) |
| Система | hostname, OS, IP, uptime, boot time, кол-во пользователей, температура CPU |
| Windows | статус RDP/SMB/WinRM через WMI, FSRM-квоты (размер, использование, нарушения) |

---

## Пробы сервисов (server-side)

Сервер сам опрашивает каждый узел по известному IP:

| Сервис | Порт | Метод |
|--------|------|-------|
| SSH | 22 | TCP dial |
| RDP | 3389 | TCP dial |
| SMB | 445 | TCP dial |
| HTTP | 80 | TCP dial |
| WinRM | 5985 | TCP dial |
| DNS | 53 | TCP dial |

Для каждого сервиса показывается статус (OK / Down) и время отклика в мс.

Дополнительно — SNMP-поллер: собирает sysUpTime, sysName, CPU load, число интерфейсов (community string: public).

---

## API

| Метод | Путь | Описание | Авторизация |
|-------|------|----------|-------------|
| POST | `/api/metrics` | Агент отправляет метрики | Нет |
| GET | `/api/nodes` | Список всех узлов с последними метриками и историей | Bearer токен |
| GET | `/api/nodes?full=true` | То же + история за 24ч (10-мин бакеты) | Bearer токен |
| DELETE | `/api/nodes/{name}` | Удалить узел из базы | Bearer токен |
| GET | `/api/auth/setup` | Проверка: нужна ли первичная регистрация | Нет |
| POST | `/api/auth/register` | Создать первого пользователя | Нет (только если нет пользователей) |
| POST | `/api/auth/login` | Вход, возвращает токен | Нет |
| POST | `/api/auth/logout` | Выход, инвалидирует токен | Bearer токен |

---

## Структура проекта

```
linkmus-monitor/
├── cmd/
│   ├── agent/main.go            # точка входа агента
│   └── server/main.go           # точка входа сервера
├── internal/
│   ├── agent/
│   │   ├── agent.go             # цикл сбора и отправки
│   │   ├── config.go            # загрузка agent-config.yaml
│   │   └── sender.go            # HTTP POST на сервер
│   ├── collector/
│   │   ├── common.go            # общие структуры
│   │   ├── cpu_{linux,windows}.go
│   │   ├── memory_{linux,windows}.go
│   │   ├── disk_{linux,windows}.go
│   │   ├── network_{linux,windows}.go
│   │   ├── process_{linux,windows}.go
│   │   ├── services_{linux,windows}.go
│   │   ├── temperature_{linux,windows}.go
│   │   ├── connections_{linux,windows}.go
│   │   └── fsrm_{linux,windows}.go
│   └── server/
│       ├── server.go            # HTTP-роутер, middleware авторизации
│       ├── api.go               # типы NodeSummary, CpuPoint, NetPoint и т.д.
│       ├── storage.go           # SQLite: схема, сохранение, запросы истории
│       ├── auth.go              # сессии, bcrypt-пароли
│       ├── prober.go            # TCP-пробы сервисов
│       └── snmp_poller.go       # SNMP-опрос узлов
├── configs/
│   └── agent-config.yaml        # пример конфига агента
├── web/
│   └── src/
│       ├── pages/
│       │   ├── Dashboard.jsx    # карточки всех узлов
│       │   ├── NodeDetail.jsx   # детальная страница узла
│       │   └── LoginPage.jsx
│       ├── components/
│       │   ├── charts/          # CpuHistory, CpuGauge, NetworkLines, Sparkline
│       │   ├── cards/           # NodeCard
│       │   ├── common/          # ProgressBar
│       │   └── layout/          # Header, Sidebar
│       ├── hooks/
│       │   ├── useNodes.js      # polling каждые 5 сек
│       │   └── useAutoRefresh.js
│       └── lib/api.js           # fetchNodes, fetchNodes(full), deleteNode
├── install.sh                   # установщик сервера и агента (Linux)
└── install-agent.ps1            # установщик агента (Windows)
```

---

## Разработка

```bash
# Запуск сервера
go run ./cmd/server/

# Запуск фронтенда (Vite проксирует /api/* → localhost:8080)
cd web && npm install && npm run dev

# Сборка агентов
GOOS=linux  GOARCH=amd64 go build -o mon-agent-linux-amd64  ./cmd/agent/
GOOS=linux  GOARCH=arm64 go build -o mon-agent-linux-arm64  ./cmd/agent/
GOOS=windows GOARCH=amd64 go build -o mon-agent.exe         ./cmd/agent/

# Сборка сервера
GOOS=linux GOARCH=amd64 go build -o mon-server ./cmd/server/
cd web && npm run build  # → web/dist/
```

### Конфиг агента (`agent-config.yaml`)

```yaml
server:
  url: "http://10.10.10.10:8080/api/metrics"
  interval: 10s
```

---

## Зависимости

| Пакет | Назначение |
|-------|-----------|
| `github.com/shirou/gopsutil/v3` | CPU, RAM, диски, сеть, процессы, хост |
| `github.com/yusufpapurcu/wmi` | WMI-запросы на Windows |
| `github.com/gosnmp/gosnmp` | SNMP-поллер |
| `modernc.org/sqlite` | SQLite без CGo (pure Go) |
| `golang.org/x/crypto` | bcrypt для паролей |
| `gopkg.in/yaml.v3` | конфиг агента |

Фронтенд: React 18, Vite 8, Tailwind CSS 3, Recharts 2, Lucide React, React Router 6.

---

## Требования

- **Сервер:** Linux amd64, Go 1.21+ (для сборки), Nginx (опционально)
- **Агент Linux:** любой дистрибутив, amd64 или arm64, root
- **Агент Windows:** Windows 10/Server 2019+, PowerShell, права администратора, NSSM
