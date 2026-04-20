# LinkMus Monitor — CLAUDE.md

## Обзор проекта

**LinkMus Monitor** — кроссплатформенная система мониторинга и визуализации состояния сетевой инфраструктуры для гетерогенной корпоративной среды. Курсовая работа по дисциплине «Телекоммуникационные технологии в отраслях транспортно-дорожного комплекса» (МАДИ, группа 2бИТС1).

---

## Архитектура

### Модель взаимодействия

**Push-модель:** агенты (`mon-agent`) на конечных узлах самостоятельно собирают метрики и периодически отправляют JSON-телеметрию на мастер-сервер (`mon-server`) через HTTP POST.

```
┌─────────────────────────────────────────────────────────────────┐
│                    VMware ESXi 8.0.2                            │
│              (Вложенная виртуализация, VMware Workstation)       │
│                                                                 │
│  ┌──────────────────┐        vSwitch-Lab (изолированный L2)     │
│  │  gw-border-01    │───────────────────────────────────────┐   │
│  │  MikroTik CHR    │  10.10.10.0/24                        │   │
│  │  10.10.10.1      │                                       │   │
│  │  WAN: ether1     │                                       │   │
│  │  LAN: ether2     │                                       │   │
│  └──────────────────┘                                       │   │
│           │                                                 │   │
│  ┌────────┴─────────────────────────────────────────────┐   │   │
│  │                                                      │   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   │   │
│  │  │ srv-mon-01  │  │ srv-corp-01 │  │ cl-astra-01 │  │   │   │
│  │  │ Astra Linux │  │ Win Server  │  │ Astra Linux │  │   │   │
│  │  │ 10.10.10.10 │  │ 10.10.10.11 │  │ 10.10.10.100│  │   │   │
│  │  │ MASTER      │  │ WMI/SNMP/   │  │ Agent       │  │   │   │
│  │  │ SERVER      │  │ FSRM/Agent  │  │             │  │   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │   │   │
│  │                                                      │   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   │   │
│  │  │ cl-win-01   │  │ cl-ubnt-01  │  │ cl-redos-01 │  │   │   │
│  │  │ Windows 10  │  │ Ubuntu 24.04│  │ РЕД ОС 8.2 │  │   │   │
│  │  │ 10.10.10.101│  │ 10.10.10.102│  │ 10.10.10.103│  │   │   │
│  │  │ Agent       │  │ Agent       │  │ Agent       │  │   │   │
│  │  │ RDP/SMB     │  │             │  │             │  │   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │   │   │
│  └──────────────────────────────────────────────────────┘   │   │
└─────────────────────────────────────────────────────────────────┘
```

### Технологический стек

| Компонент | Технология |
|-----------|-----------|
| Агент (mon-agent) | Go, кросс-компиляция (linux/amd64, windows/amd64) |
| Сервер (mon-server) | Go, net/http, SQLite (`modernc.org/sqlite` — pure Go, без CGo) |
| Фронтенд | React (Vite), Tailwind CSS, Recharts, тёмная тема |
| Прокси | Nginx на srv-mon-01 (порт 80) |
| Внешний доступ | Destination NAT: WAN:8080 → 10.10.10.10:80 |
| Сборка метрик (Linux) | /proc, /sys, системные вызовы |
| Сборка метрик (Windows) | WMI (Win32_* классы), WinRM, SNMP, Event Log |

---

## Узлы инфраструктуры

### Полная карта узлов

| Имя ВМ | ОС | IP | vCPU | RAM | vHDD | Роль | Источники метрик |
|--------|----|----|------|-----|------|------|-----------------|
| gw-border-01 | MikroTik RouterOS 7.20.8 | 10.10.10.1 | 1 | 1 ГБ | 128 МБ | Пограничный шлюз, NAT, DNS, Firewall | SNMP (опционально), API RouterOS |
| srv-mon-01 | Astra Linux CE «Орёл» 2.12 | 10.10.10.10 | 2 | 4 ГБ | 40 ГБ | Мастер-сервер мониторинга | Локальный сбор (self-monitoring) |
| srv-corp-01 | Windows Server 2022 | 10.10.10.11 | 4 | 4 ГБ | 65 ГБ | Корпоративный сервер | WMI, WinRM, SNMP, FSRM, Event Log |
| cl-astra-01 | Astra Linux CE «Орёл» 2.12 | 10.10.10.100 | 1 | 2 ГБ | 20 ГБ | АРМ (DEB) | /proc, /sys, SSH |
| cl-win-01 | Windows 10 Pro | 10.10.10.101 | 2 | 4 ГБ | 55 ГБ | АРМ (Windows) | WMI, RDP/SMB probe |
| cl-ubnt-01 | Ubuntu Desktop 24.04 | 10.10.10.102 | 1 | 2 ГБ | 20 ГБ | АРМ (DEB) | /proc, /sys, SSH |
| cl-redos-01 | РЕД ОС 8.2 | 10.10.10.103 | 1 | 2 ГБ | 25 ГБ | АРМ (RPM) | /proc, /sys, SSH |

### Сетевая конфигурация

- **Внешняя сеть:** 192.168.5.0/24 (ESXi: .135, MikroTik WAN: .144)
- **Внутренняя сеть:** 10.10.10.0/24 (vSwitch-Lab, изолированный L2)
- **DNS:** MikroTik (10.10.10.1) — кэширующий DNS для всех узлов
- **NAT:** srcnat masquerade на ether1 (выход в интернет)
- **DNAT:** WAN:8080 → 10.10.10.10:80 (доступ к веб-интерфейсу)
- **Серверный пул:** 10.10.10.10–10.10.10.19
- **Клиентский пул:** 10.10.10.100–10.10.10.150

---

## Метрики для сбора

### Цель: максимально информативный дашборд

Агент должен собирать **расширенный** набор метрик, выходящий далеко за рамки базовых CPU/RAM/Disk. Это позволит строить насыщенные визуализации, выявлять аномалии и демонстрировать глубину системы.

---

### 1. Системные метрики (все узлы)

#### 1.1. CPU

| Метрика | Linux-источник | Windows-источник | Тип |
|---------|---------------|-----------------|-----|
| cpu_usage_percent | /proc/stat (user+system+nice+irq+softirq) | Win32_Processor.LoadPercentage | gauge |
| cpu_usage_per_core[] | /proc/stat (cpuN строки) | Win32_PerfFormattedData_Counters (per instance) | gauge[] |
| cpu_user_percent | /proc/stat (user) | — | gauge |
| cpu_system_percent | /proc/stat (system) | — | gauge |
| cpu_iowait_percent | /proc/stat (iowait) | — | gauge |
| cpu_steal_percent | /proc/stat (steal) — важно для VM | — | gauge |
| cpu_count | /proc/cpuinfo | Win32_Processor.NumberOfLogicalProcessors | info |
| cpu_model | /proc/cpuinfo (model name) | Win32_Processor.Name | info |
| cpu_freq_mhz | /proc/cpuinfo или /sys/devices/system/cpu/cpu0/cpufreq | Win32_Processor.CurrentClockSpeed | gauge |
| load_avg_1m | /proc/loadavg (поле 1) | — | gauge |
| load_avg_5m | /proc/loadavg (поле 2) | — | gauge |
| load_avg_15m | /proc/loadavg (поле 3) | — | gauge |
| process_count | /proc/loadavg (поле 4, running/total) | Win32_OperatingSystem.NumberOfProcesses | gauge |
| context_switches | /proc/stat (ctxt) | — | counter |
| interrupts | /proc/stat (intr, первое число) | — | counter |

#### 1.2. Память (RAM)

| Метрика | Linux-источник | Windows-источник | Тип |
|---------|---------------|-----------------|-----|
| mem_total_bytes | /proc/meminfo (MemTotal) | Win32_OperatingSystem.TotalVisibleMemorySize | info |
| mem_used_bytes | MemTotal - MemAvailable | Total - FreePhysicalMemory | gauge |
| mem_available_bytes | /proc/meminfo (MemAvailable) | Win32_OperatingSystem.FreePhysicalMemory | gauge |
| mem_usage_percent | (used / total) * 100 | вычислить | gauge |
| mem_buffers_bytes | /proc/meminfo (Buffers) | — | gauge |
| mem_cached_bytes | /proc/meminfo (Cached) | — | gauge |
| mem_swap_total_bytes | /proc/meminfo (SwapTotal) | Win32_PageFileUsage.AllocatedBaseSize | info |
| mem_swap_used_bytes | SwapTotal - SwapFree | Win32_PageFileUsage.CurrentUsage | gauge |
| mem_swap_usage_percent | (swap_used / swap_total) * 100 | вычислить | gauge |

#### 1.3. Диски

| Метрика | Linux-источник | Windows-источник | Тип |
|---------|---------------|-----------------|-----|
| disk_total_bytes | syscall Statfs | Win32_LogicalDisk.Size | info |
| disk_used_bytes | Statfs | Size - FreeSpace | gauge |
| disk_free_bytes | Statfs (Bavail * Bsize) | Win32_LogicalDisk.FreeSpace | gauge |
| disk_usage_percent | вычислить | вычислить | gauge |
| disk_mount_point | парсить /proc/mounts | Win32_LogicalDisk.DeviceID (C:, D:) | info |
| disk_fs_type | /proc/mounts | Win32_LogicalDisk.FileSystem | info |
| disk_inodes_total | Statfs.Files | — | info |
| disk_inodes_used | Files - Ffree | — | gauge |
| disk_read_bytes_sec | /proc/diskstats (сектора * 512) | Win32_PerfFormattedData_PerfDisk | gauge |
| disk_write_bytes_sec | /proc/diskstats | Win32_PerfFormattedData_PerfDisk | gauge |
| disk_read_iops | /proc/diskstats (поле 1) | Win32_PerfFormattedData_PerfDisk | gauge |
| disk_write_iops | /proc/diskstats (поле 5) | Win32_PerfFormattedData_PerfDisk | gauge |
| disk_io_time_ms | /proc/diskstats (поле 10) | — | counter |
| disk_queue_length | /proc/diskstats | Win32_PerfFormattedData_PerfDisk.AvgDiskQueueLength | gauge |

> **Текущее состояние:** агент собирает только `disk_usage_percent` для `C:\` (Windows) или `/` (Linux) через `gopsutil/disk`. Остальные дисковые метрики — в планах расширения.

#### 1.4. Сеть

| Метрика | Linux-источник | Windows-источник | Тип |
|---------|---------------|-----------------|-----|
| net_interface_name | /proc/net/dev | Win32_NetworkAdapter.NetConnectionID | info |
| net_ip_address | ip addr | Win32_NetworkAdapterConfiguration.IPAddress | info |
| net_mac_address | /sys/class/net/*/address | Win32_NetworkAdapter.MACAddress | info |
| net_bytes_recv | /proc/net/dev (bytes recv) | Win32_PerfRawData_Tcpip.BytesReceivedPersec | counter |
| net_bytes_sent | /proc/net/dev (bytes sent) | Win32_PerfRawData_Tcpip.BytesSentPersec | counter |
| net_packets_recv | /proc/net/dev | Win32_PerfRawData_Tcpip.PacketsReceivedPersec | counter |
| net_packets_sent | /proc/net/dev | Win32_PerfRawData_Tcpip.PacketsSentPersec | counter |
| net_errors_in / out | /proc/net/dev (errs) | — | counter |
| net_drops_in / out | /proc/net/dev (drop) | — | counter |
| net_bandwidth_mbps | /sys/class/net/*/speed | Win32_NetworkAdapter.Speed | info |
| net_link_status | /sys/class/net/*/operstate | Win32_NetworkAdapter.NetConnectionStatus | info |
| net_tcp_connections | /proc/net/tcp (count) | netstat / WMI | gauge |
| net_tcp_established | /proc/net/tcp (state=01) | — | gauge |
| net_tcp_time_wait | /proc/net/tcp (state=06) | — | gauge |

#### 1.5. Uptime и системная информация

| Метрика | Linux-источник | Windows-источник | Тип |
|---------|---------------|-----------------|-----|
| uptime_seconds | /proc/uptime | Win32_OperatingSystem.LastBootUpTime | gauge |
| hostname | os.Hostname() | os.Hostname() | info |
| os_name | /etc/os-release (PRETTY_NAME) | Win32_OperatingSystem.Caption | info |
| os_version | /etc/os-release (VERSION_ID) | Win32_OperatingSystem.Version | info |
| kernel_version | /proc/version | Win32_OperatingSystem.BuildNumber | info |
| architecture | runtime.GOARCH | runtime.GOARCH | info |
| agent_version | встроенная константа | встроенная константа | info |
| logged_users | /var/run/utmp | Win32_LogonSession (count) | gauge |

> **Текущее состояние:** агент собирает hostname, os, ip, uptime через `gopsutil/host`. Формат uptime — строка `"N ч."`.

#### 1.6. Процессы (Top-N)

| Метрика | Linux-источник | Windows-источник |
|---------|---------------|-----------------|
| top_cpu_processes[] | /proc/[pid]/stat, top-10 | Win32_PerfFormattedData_PerfProc, top-10 |
| top_mem_processes[] | /proc/[pid]/status (VmRSS), top-10 | Win32_Process.WorkingSetSize, top-10 |
| process_total | count /proc/[pid] | Win32_OperatingSystem.NumberOfProcesses |
| zombies_count | /proc/[pid]/status (State=Z) | — |

Формат каждого процесса в массиве:
```json
{
  "pid": 1234,
  "name": "nginx",
  "cpu_percent": 12.5,
  "mem_bytes": 52428800,
  "user": "www-data"
}
```

#### 1.7. Температура и датчики (если доступно)

| Метрика | Linux-источник | Windows-источник |
|---------|---------------|-----------------|
| cpu_temperature_celsius | /sys/class/thermal/thermal_zone*/temp | WMI MSAcpi_ThermalZoneTemperature |

> В виртуальных машинах датчики часто недоступны. Агент должен gracefully пропускать, если файлы/классы отсутствуют.

---

### 2. Метрики доступности сервисов (Service Probes)

| Проба | Цель | Метод | Метрики |
|-------|------|-------|---------|
| ICMP Ping | Все узлы | net.Dial("ip4:icmp") | alive (bool), rtt_ms, packet_loss_percent |
| SSH (TCP 22) | Linux-узлы | net.DialTimeout("tcp", ":22") | reachable (bool), response_time_ms |
| RDP (TCP 3389) | cl-win-01, srv-corp-01 | net.DialTimeout("tcp", ":3389") | reachable (bool), response_time_ms |
| SMB (TCP 445) | cl-win-01, srv-corp-01 | net.DialTimeout("tcp", ":445") | reachable (bool), response_time_ms |
| HTTP (TCP 80) | srv-mon-01 (self) | http.Get | reachable (bool), response_time_ms, status_code |
| SNMP (UDP 161) | srv-corp-01 | gosnmp GET sysUpTime | reachable (bool), response_time_ms |
| WinRM (TCP 5985) | srv-corp-01 | net.DialTimeout("tcp", ":5985") | reachable (bool), response_time_ms |
| DNS | gw-border-01 | net.Resolver с custom dialer на 10.10.10.1 | reachable (bool), response_time_ms |

> **Текущее состояние:** агент на Windows опрашивает RDP (TermService) и SMB (LanmanServer) через WMI Win32_Service и возвращает `rdp_running` / `smb_running` (bool). TCP-пробы и ping — в планах.

---

### 3. Специфические метрики корпоративного сервера (srv-corp-01)

#### 3.1. SNMP-метрики

| OID | Метрика |
|-----|---------|
| .1.3.6.1.2.1.1.3.0 | sysUpTime |
| .1.3.6.1.2.1.1.5.0 | sysName |
| .1.3.6.1.2.1.25.2.3.1.* | hrStorageTable (диски) |
| .1.3.6.1.2.1.25.3.3.1.2 | hrProcessorLoad |
| .1.3.6.1.2.1.2.2.1.* | ifTable (сетевые интерфейсы) |

Community String: Read-Only, доверенный узел 10.10.10.10.

#### 3.2. FSRM-события (из Windows Event Log)

| Event Source | Event ID | Значение |
|-------------|----------|----------|
| SRMSVC | 8215 | Превышение мягкой квоты (Warning) |
| SRMSVC | 8216 | Превышение жёсткой квоты (Error) |
| SRMSVC | 12325 | Блокировка файла по фильтру |

Метрики для дашборда:
```json
{
  "fsrm_quota_path": "C:\\CorpShare",
  "fsrm_quota_limit_bytes": 1073741824,
  "fsrm_quota_used_bytes": 734003200,
  "fsrm_quota_usage_percent": 68.4,
  "fsrm_violations_24h": 3,
  "fsrm_last_violation_time": "2026-04-15T14:32:00Z",
  "fsrm_last_violation_type": "quota_exceeded"
}
```

#### 3.3. Windows-специфичные WMI-метрики

| WMI-класс | Метрика |
|-----------|---------|
| Win32_Service | Список служб: Name, State, StartMode |
| Win32_Share | Сетевые шары: Name, Path, Status |
| Win32_LogicalDisk | Для всех томов (C:, D:, ...) |
| Win32_NTLogEvent | Критические события за последний час (EventType=1) |

---

### 4. Метрики MikroTik (gw-border-01) — опционально

| Метрика | Источник |
|---------|----------|
| mikrotik_cpu_load | /system/resource (cpu-load) |
| mikrotik_memory_used / total | /system/resource |
| mikrotik_uptime | /system/resource |
| mikrotik_interface_rx/tx_bytes | /interface/monitor-traffic |
| mikrotik_firewall_connections | /ip/firewall/connection |
| mikrotik_routes_count | /ip/route/print count-only |

---

## API-контракт (mon-server)

### Текущие эндпоинты (реализованы)

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/metrics` | Агент отправляет метрики |
| GET | `/api/nodes` | Фронтенд получает все узлы + историю CPU (последние 20 точек) |

### Планируемые эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/nodes/{id}` | Детальная информация об узле |
| GET | `/api/v1/nodes/{id}/metrics` | Исторические метрики (для графиков) |
| GET | `/api/v1/nodes/{id}/processes` | Top-N процессов |
| GET | `/api/v1/nodes/{id}/services` | Статус сервисов |
| GET | `/api/v1/overview` | Агрегированная сводка |
| GET | `/api/v1/alerts` | Активные алерты |
| GET | `/api/v1/events` | Лента событий |

### Текущий формат ответа `/api/nodes` (структура `NodeSummary`)

```json
{
  "name": "HOSTNAME",
  "os": "windows Windows Server 2022",
  "ip": "10.10.10.11",
  "online": true,
  "cpu": 42,
  "ramUsed": 2,
  "ramTotal": 4,
  "diskUsage": 52.3,
  "rdpRunning": true,
  "smbRunning": true,
  "uptime": "36 ч.",
  "ping": 1,
  "cpuHistory": [{"value": 40}, {"value": 45}]
}
```

> `diskUsage` — процент (0–100), не ГБ. `ramUsed` / `ramTotal` — в ГБ (float64, приводится к int при записи в NodeSummary). `cpuHistory` — последние 20 точек, порядок от старых к новым.

### Текущий формат payload от агента (`MetricPayload`)

```json
{
  "node_name": "HOSTNAME",
  "os": "windows Windows Server 2022",
  "ip": "10.10.10.11",
  "uptime": "36 ч.",
  "timestamp": "2026-04-15T14:30:00Z",
  "cpu_usage": 42.5,
  "ram_usage": 2.7,
  "ram_total": 4.0,
  "disk_usage": 52.3,
  "rdp_running": true,
  "smb_running": true
}
```

---

## Структура проекта (текущая + целевая)

```
linkmus-monitor/
├── CLAUDE.md
├── go.mod / go.sum                  # Единый Go-модуль: linkmus-monitor
├── configs/
│   └── agent-config.yaml            # server.url, server.interval (сейчас 3s)
├── cmd/
│   └── agent/
│       └── main.go                  # Точка входа агента
├── internal/
│   ├── agent/
│   │   ├── agent.go                 # Цикл сбора и отправки (collectAndSend)
│   │   ├── config.go                # LoadConfig (yaml -> struct)
│   │   └── sender.go                # SendToServer (HTTP POST)
│   ├── collector/
│   │   ├── common.go                # Интерфейс Collector, структура SystemMetrics
│   │   ├── cpu_windows.go           # CollectCPU() — gopsutil/cpu
│   │   ├── memory_windows.go        # (есть в проекте)
│   │   ├── disk_windows.go          # CollectDisk() — gopsutil/disk, только C:\
│   │   └── services_windows.go      # CollectServices() — WMI Win32_Service (RDP, SMB)
│   └── server/
│       ├── server.go                # Run(), handleMetrics(), MetricPayload struct
│       ├── api.go                   # HandleNodes(), NodeSummary struct, CpuPoint struct
│       └── storage.go               # InitDB(), SaveMetric(), GetLatestNodes()
└── web/
    ├── package.json
    ├── vite.config.js               # Proxy /api/* -> localhost:8080
    └── src/
        ├── App.jsx                  # Router: / -> Dashboard, /node/:nodeId -> NodeDetail
        ├── main.jsx
        ├── lib/
        │   └── api.js               # fetchNodes(), fetchNodeDetail()
        ├── hooks/
        │   ├── useNodes.js          # useAutoRefresh(fetchNodes, 5000)
        │   ├── useNodeDetail.js     # useAutoRefresh(fetchNodeDetail, 5000)
        │   └── useAutoRefresh.js    # Хук polling
        ├── pages/
        │   ├── Dashboard.jsx        # Карточки всех узлов
        │   └── NodeDetail.jsx       # Детальная страница (использует useNodes + find по имени)
        └── components/
            ├── common/
            │   ├── MetricCard.jsx
            │   └── ProgressBar.jsx
            ├── charts/
            │   ├── CpuGauge.jsx
            │   ├── CpuHistory.jsx   # Recharts AreaChart
            │   ├── RamBar.jsx
            │   ├── RamPie.jsx
            │   ├── DiskBars.jsx     # Поддерживает unit: '%' и unit: 'GB'
            │   ├── NetworkLines.jsx
            │   └── Sparkline.jsx
            ├── cards/
            │   └── NodeCard.jsx
            ├── tables/
            │   └── ProcessTable.jsx
            ├── status/
            │   ├── ServiceStatus.jsx
            │   └── FsrmQuota.jsx
            └── layout/
                ├── Header.jsx
                └── Sidebar.jsx
```

---

## SQLite-схема (текущая + целевая)

### Текущая таблица `metrics`

```sql
CREATE TABLE IF NOT EXISTS metrics (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node_name   TEXT,
    os          TEXT,
    ip          TEXT,
    uptime      TEXT,
    timestamp   DATETIME,
    cpu_usage   REAL,
    ram_usage   REAL,
    ram_total   REAL,
    disk_usage  REAL,
    rdp_running BOOLEAN,
    smb_running BOOLEAN
);
```

### Целевая расширенная схема

```sql
-- Узлы (справочник)
CREATE TABLE nodes (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    ip_address TEXT,
    os_name TEXT,
    os_version TEXT,
    kernel_version TEXT,
    cpu_model TEXT,
    cpu_count INTEGER,
    mem_total_bytes INTEGER,
    agent_version TEXT,
    status TEXT DEFAULT 'unknown',
    first_seen DATETIME,
    last_seen DATETIME
);

-- Метрики (time-series)
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    cpu_percent REAL, cpu_user REAL, cpu_system REAL,
    cpu_iowait REAL, cpu_steal REAL,
    load_1m REAL, load_5m REAL, load_15m REAL,
    mem_used_bytes INTEGER, mem_available_bytes INTEGER,
    mem_usage_percent REAL, mem_swap_used_bytes INTEGER,
    process_count INTEGER, tcp_connections INTEGER, logged_users INTEGER
);
CREATE INDEX idx_metrics_node_time ON metrics(node_id, timestamp);

-- Дисковые метрики (per-mount)
CREATE TABLE disk_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    mount_point TEXT NOT NULL,
    total_bytes INTEGER, used_bytes INTEGER, usage_percent REAL,
    read_bytes_sec REAL, write_bytes_sec REAL,
    read_iops REAL, write_iops REAL, queue_length REAL
);

-- Сетевые метрики (per-interface)
CREATE TABLE net_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    interface_name TEXT NOT NULL,
    bytes_recv INTEGER, bytes_sent INTEGER,
    errors_in INTEGER, errors_out INTEGER,
    drops_in INTEGER, drops_out INTEGER
);

-- Сервисные пробы
CREATE TABLE service_probes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    service TEXT NOT NULL,
    port INTEGER,
    reachable BOOLEAN,
    response_time_ms REAL
);

-- FSRM-события
CREATE TABLE fsrm_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL,
    event_id INTEGER,
    path TEXT,
    details TEXT
);

-- Алерты
CREATE TABLE alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    created_at DATETIME NOT NULL,
    resolved_at DATETIME,
    severity TEXT NOT NULL,
    metric TEXT NOT NULL,
    threshold REAL, current_value REAL,
    message TEXT, active BOOLEAN DEFAULT 1
);

-- Снапшоты процессов
CREATE TABLE process_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL REFERENCES nodes(id),
    timestamp DATETIME NOT NULL,
    data_json TEXT NOT NULL
);
```

---

## Пороги алертов

| Метрика | Warning | Critical |
|---------|---------|----------|
| cpu_percent | > 80% (5 мин) | > 95% (3 мин) |
| mem_usage_percent | > 85% | > 95% |
| disk_usage_percent | > 80% | > 90% |
| swap_usage_percent | > 50% | > 80% |
| node_offline | — | last_seen > 60 сек |
| service_down | — | reachable = false (2 проверки подряд) |
| fsrm_quota | > 80% usage | > 95% или hard limit |
| disk_iowait | > 30% | > 60% |
| load_avg_1m | > cpu_count * 2 | > cpu_count * 4 |

---

## Фронтенд: целевой вид интерфейса

### Приоритет: современный, приятный, максимально информативный UI

Ориентир — Grafana, Datadog, Netdata. Принципы:
- Живые графики важнее статичных чисел
- Цветовая индикация: 🟢 норма / 🟡 внимание / 🔴 критично
- Карточки узлов на Dashboard дают мгновенный обзор без перехода на детали
- Страница NodeDetail — исчерпывающая картина по узлу
- Гетерогенность: Linux-узлы не показывают RDP/SMB, Windows — не показывает load avg / iowait

### Dashboard (главная страница)

```
┌─── Summary Cards ─────────────────────────────────────────┐
│ 🖥 Узлы: 6/7  │ 🔔 Алерты: 2  │ ⚡ Сервисы: 11/12       │
│ CPU avg: 18%  │ RAM avg: 52%  │ Диск avg: 45%            │
└───────────────────────────────────────────────────────────┘

┌─── Topology Map (SVG) ────────────────────────────────────┐
│  [MikroTik]──┬──[srv-mon-01]──[srv-corp-01]               │
│              ├──[cl-astra-01]                              │
│              ├──[cl-win-01]                                │
│              ├──[cl-ubnt-01]                               │
│              └──[cl-redos-01]                              │
│  Цвета: 🟢 online  🔴 offline  🟡 warning                 │
└───────────────────────────────────────────────────────────┘

┌─── Node Cards Grid ───────────────────────────────────────┐
│ Карточка: hostname, OS, IP, CPU%, RAM%, Disk%,            │
│ uptime, sparkline CPU, статусы сервисов (SSH/RDP/SMB)     │
└───────────────────────────────────────────────────────────┘

┌─── Recent Alerts ─────────────────────────────────────────┐
│ 🔴 cl-win-01 — SMB недоступен (14:25)                     │
│ 🟡 srv-corp-01 — Disk C: > 80% (14:20)                    │
└───────────────────────────────────────────────────────────┘

┌─── Event Feed ────────────────────────────────────────────┐
│ 14:32 srv-corp-01 FSRM: Quota warning on CorpShare        │
│ 14:30 cl-redos-01 Agent reconnected                        │
└───────────────────────────────────────────────────────────┘
```

### NodeDetail (детальная страница)

Секции (показываются по наличию данных от агента):
1. **System Info** — hostname, OS, IP, uptime, cpu_model, kernel
2. **CPU** — AreaChart (user/system/iowait/steal), текущая загрузка, load avg
3. **RAM** — gauge + AreaChart, buffers/cached, swap
4. **Disks** — per-mount bars, IO (R/W MB/s, IOPS, queue)
5. **Network** — per-interface (RX/TX chart, errors, drops, TCP connections)
6. **Services** — индикаторы (SSH/RDP/SMB/HTTP/SNMP) с временем отклика
7. **Top Processes** — таблица (PID, Name, CPU%, RAM, User)
8. **FSRM** — только для srv-corp-01 (quota bar, нарушения за 24ч)

---

## Правило добавления новой метрики

При добавлении любой новой метрики нужно последовательно обновить:

1. `MetricPayload` в `internal/agent/agent.go` и `internal/server/server.go`
2. Коллектор: `internal/collector/*_linux.go` и/или `*_windows.go`
3. `CREATE TABLE` и `INSERT` в `internal/server/storage.go`
4. `NodeSummary` в `internal/server/api.go`
5. `GetLatestNodes()` в `internal/server/storage.go`
6. Отображение в `web/src/pages/NodeDetail.jsx` и/или `Dashboard.jsx`

---

## Go-зависимости (go.mod)

```
github.com/shirou/gopsutil/v3      # CPU, RAM, Disk, Host (текущая зависимость)
github.com/yusufpapurcu/wmi        # WMI на Windows (текущая зависимость)
gopkg.in/yaml.v3                   # Конфиг агента (текущая зависимость)
modernc.org/sqlite                 # Pure Go SQLite, без CGo (текущая зависимость)
github.com/go-ole/go-ole           # COM для WMI (транзитивная)
github.com/gosnmp/gosnmp           # SNMP-поллер (планируется)
github.com/rs/cors                 # CORS для dev-режима (планируется)
```

---

## Сборка и деплой

### Кросс-компиляция агента

```bash
# Linux (amd64) — для Astra, Ubuntu, РЕД ОС
GOOS=linux GOARCH=amd64 go build -o mon-agent-linux ./cmd/agent/

# Windows (amd64)
GOOS=windows GOARCH=amd64 go build -o mon-agent.exe ./cmd/agent/
```

### Запуск для разработки

```bash
# Сервер
go run ./cmd/agent/   # или go run ./cmd/server/ когда будет выделен

# Фронтенд
cd web && npm install && npm run dev
# Vite проксирует /api/* -> http://localhost:8080
```

### Systemd unit (Linux-агент)

```ini
[Unit]
Description=LinkMus Monitor Agent
After=network-online.target

[Service]
Type=simple
ExecStart=/opt/mon-agent/mon-agent -config /opt/mon-agent/config.yaml
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
```

### Windows (через NSSM)

```powershell
nssm install MonAgent "C:\mon-agent\mon-agent.exe" "-config C:\mon-agent\config.yaml"
nssm set MonAgent Start SERVICE_AUTO_START
nssm start MonAgent
```

### Nginx (srv-mon-01)

```nginx
server {
    listen 80;
    server_name _;

    root /opt/linkmus-monitor/web/dist;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```
