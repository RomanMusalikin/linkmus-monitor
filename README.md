# LinkMus Monitor

Система мониторинга серверов и рабочих станций для гетерогенных корпоративных сетей. Агенты собирают метрики и отправляют их на центральный сервер, веб-интерфейс отображает состояние всей инфраструктуры в реальном времени.

---

## Возможности

- **Поддержка Linux и Windows** — агент компилируется под amd64/arm64 (Linux) и amd64 (Windows)
- **Дашборд** — карточки всех узлов с CPU, RAM, диском, сетью и статусом сервисов (SSH/RDP/SMB)
- **Детальная страница узла** — графики, процессы, диски, сетевые интерфейсы, TCP, SNMP, FSRM-квоты
- **История за сутки** — график с 10-минутным усреднением, разрывы показывают периоды офлайна
- **Пробы сервисов** — SSH, RDP, SMB, HTTP, WinRM, DNS с временем отклика (server-side TCP)
- **Авторизация** — первый запуск создаёт учётную запись, далее вход по логину/паролю
- **Автообновление** — данные обновляются каждые 10 секунд без перезагрузки страницы

---

## Архитектура

```
  Агенты (mon-agent)          Мастер-сервер (mon-server)
  ┌─────────────────┐         ┌──────────────────────────┐
  │ Linux / Windows │──POST──▶│ Go + SQLite (WAL)         │
  │ сбор метрик     │         │ /api/metrics  (агенты)    │
  │ каждые N сек    │         │ /api/nodes    (фронтенд)  │
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
# Выбрать пункт 1 — Установить / обновить сервер
```

Скрипт скачает бинарник и фронтенд из последнего GitHub Release, создаст systemd-службу `mon-server` (порт 8080) и настроит Nginx (если установлен). При повторном запуске — обновит до новой версии.

После установки откройте `http://<IP-сервера>` — при первом входе система предложит создать учётную запись администратора.

---

## Управление службами

После установки доступна команда `mon`:

```bash
mon server start|stop|restart|status|logs
mon agent  start|stop|restart|status|logs
mon server update   # проверить и установить обновление сервера
mon agent  update   # проверить и установить обновление агента
mon help
```

---

## Установка агента

### Linux (amd64 / arm64)

```bash
curl -sSL https://raw.githubusercontent.com/RomanMusalikin/linkmus-monitor/main/install.sh | sudo bash
# Выбрать пункт 2 — Установить / обновить агент Linux
```

Скрипт спросит URL сервера и интервал отправки, создаст systemd-службу `mon-agent`. При повторном запуске — обновит бинарник и предложит изменить настройки.

### Windows (amd64)

1. Скачайте `mon-agent-windows-amd64.zip` со страницы релизов через браузер
2. Распакуйте ZIP — внутри будет `mon-agent.exe` и `install-agent.ps1`
3. Запустите PowerShell от имени Администратора в папке с файлами:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; .\install-agent.ps1
```

Скрипт:
- Обнаружит `mon-agent.exe` рядом с собой (офлайн-режим, сеть не нужна)
- Спросит URL сервера и интервал отправки (можно ввести `5` или `5s`)
- Создаст конфиг `C:\mon-agent\agent-config.yaml`
- Зарегистрирует службу `MonAgent` через встроенный `New-Service` с автозапуском и перезапуском при сбое

Логи агента: `C:\mon-agent\mon-agent.log` (ротация при достижении 5 МБ).

После установки `install-agent.ps1` команда `mon` доступна из любого терминала автоматически.

Если команда `mon` не найдена — создайте шим вручную в PowerShell от имени Администратора:

```powershell
Set-Content "C:\Windows\System32\mon.cmd" -Encoding ASCII -Value "@echo off`npowershell -NoProfile -ExecutionPolicy Bypass -File `"C:\mon-agent\mon.ps1`" %*"
```

```powershell
mon agent start|stop|restart|status|enable|disable|logs|update
mon help
```

---

## Метрики агента

| Категория | Метрики |
|-----------|---------|
| CPU | загрузка %, user/system/iowait/steal %, частота, модель, загрузка по ядрам, load avg (Linux) |
| RAM | использовано/всего ГБ, кэш, буферы, swap |
| Диски | процент, ГБ по каждому разделу, чтение/запись МБ/с, очередь I/O |
| Сеть | входящий/исходящий трафик Б/с по интерфейсу и всем интерфейсам, TCP-соединения |
| Процессы | top-10 по CPU и по RAM (PID, имя, %, МБ, пользователь) |
| Система | hostname, OS, IP, uptime, boot time, кол-во пользователей, температура CPU |
| Windows | статус RDP/SMB через WMI, FSRM-квоты (размер, использование, нарушения) |

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

Для каждого сервиса показывается статус (зелёный / красный) и время отклика в мс.

Дополнительно — SNMP-поллер: собирает sysUpTime, sysName, CPU load, число интерфейсов (community: public).

---

## API

| Метод | Путь | Описание | Авторизация |
|-------|------|----------|-------------|
| POST | `/api/metrics` | Агент отправляет метрики | Нет |
| GET | `/api/nodes` | Список узлов из кэша (обновляется каждые 10с) | Bearer токен |
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
│   ├── agent/main.go              # точка входа агента
│   └── server/main.go             # точка входа сервера
├── internal/
│   ├── agent/
│   │   ├── agent.go               # цикл сбора и отправки, ротация лога
│   │   ├── config.go              # загрузка agent-config.yaml (путь рядом с exe)
│   │   └── sender.go              # HTTP POST с таймаутом 10с
│   ├── collector/
│   │   ├── common.go              # общие структуры
│   │   ├── cpu_linux.go / cpu_windows.go
│   │   ├── disk_linux.go          # фильтрует snap/squashfs/tmpfs
│   │   ├── disk_windows.go
│   │   ├── memory_linux.go / memory_windows.go
│   │   ├── network_linux.go / network_windows.go
│   │   ├── process_linux.go / process_windows.go
│   │   ├── services_linux.go / services_windows.go
│   │   ├── temperature_linux.go / temperature_windows.go
│   │   ├── connections_linux.go / connections_windows.go
│   │   └── fsrm_linux.go / fsrm_windows.go
│   └── server/
│       ├── server.go              # HTTP-роутер, CORS, middleware авторизации
│       ├── api.go                 # типы NodeSummary, кэш узлов (10с), HandleNodes
│       ├── storage.go             # SQLite WAL, JOIN-запрос, история, очистка 25ч
│       ├── auth.go                # сессии, bcrypt-пароли
│       ├── prober.go              # TCP-пробы сервисов
│       └── snmp_poller.go         # SNMP-опрос узлов
├── web/
│   ├── index.html                 # title: LinkMus Monitor
│   ├── public/favicon.svg         # иконка сервера
│   └── src/
│       ├── App.jsx                # роутер + NodesContext (единый polling)
│       ├── context/NodesContext.js
│       ├── pages/
│       │   ├── Dashboard.jsx
│       │   ├── NodeDetail.jsx
│       │   └── LoginPage.jsx
│       ├── components/
│       │   ├── cards/NodeCard.jsx  # SSH/RDP/SMB всегда видны, красный если недоступен
│       │   ├── charts/
│       │   └── layout/
│       ├── hooks/
│       │   ├── useNodes.js         # polling каждые 10 сек
│       │   ├── useAutoRefresh.js
│       │   └── useAuth.js
│       └── lib/api.js
├── .github/workflows/release.yml  # CI: сборка linux/windows/arm64 + релиз
├── install.sh                     # установщик сервера и агента Linux
├── install-agent.ps1              # установщик агента Windows (офлайн + онлайн режим)
├── go.mod
└── go.sum
```

---

## Разработка

```bash
# Сервер
go run ./cmd/server/

# Фронтенд (Vite проксирует /api/* → localhost:8080)
cd web && npm install && npm run dev

# Сборка агентов
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/mon-agent-linux        ./cmd/agent/
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -o dist/mon-agent-linux-arm64  ./cmd/agent/
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/mon-agent.exe          ./cmd/agent/

# Сборка сервера
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/mon-server ./cmd/server/
cd web && npm run build  # → web/dist/
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

Фронтенд: React 18, Vite, Tailwind CSS 3, Recharts 2, Lucide React, React Router 6.

---

## Требования

- **Сервер:** Linux amd64, Go 1.21+ (для сборки из исходников)
- **Агент Linux:** любой дистрибутив, amd64 или arm64, права root
- **Агент Windows:** Windows 10 / Server 2019+, PowerShell, права Администратора
