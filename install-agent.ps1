# LinkMus Monitor — установщик агента для Windows
# Запуск: powershell -ExecutionPolicy Bypass -File install-agent.ps1
#
# Режимы работы:
#   ЛОКАЛЬНЫЙ — если mon-agent.exe лежит рядом со скриптом, сеть не нужна.
#               Просто скачай ZIP со страницы релизов и распакуй.
#   ОНЛАЙН    — скачивает последний релиз с GitHub автоматически.

#Requires -RunAsAdministrator

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
chcp 65001 | Out-Null

$REPO         = "RomanMusalikin/linkmus-monitor"
$INSTALL_DIR  = "C:\mon-agent"
$CONFIG_FILE  = "$INSTALL_DIR\agent-config.yaml"
$VERSION_FILE = "$INSTALL_DIR\.version"
$SERVICE_NAME = "MonAgent"
$NSSM_URL     = "https://nssm.cc/release/nssm-2.24.zip"

# Директория, где лежит сам скрипт
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step { Write-Host "[....] $args" -ForegroundColor Cyan }
function Write-Ok   { Write-Host "[ OK ] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Fail { param([string]$msg) Write-Host "[ERR]  $msg" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "╔══════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  LinkMus Monitor — Установщик агента    ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ── Определяем режим: локальный или онлайн ────────────────────────────────────
$localExe = Join-Path $SCRIPT_DIR "mon-agent.exe"
$useLocal = Test-Path $localExe

if ($useLocal) {
    Write-Ok "Найден mon-agent.exe рядом со скриптом — работаем без сети."
    $latestTag = "local"
    if (Test-Path $VERSION_FILE) {
        $latestTag = (Get-Content $VERSION_FILE -Raw).Trim()
    }
} else {
    Write-Step "mon-agent.exe не найден рядом — пробуем скачать с GitHub..."

    $latestTag   = $null
    $downloadUrl = $null
    try {
        $release    = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
        $latestTag  = $release.tag_name
        $asset      = $release.assets | Where-Object { $_.name -like "*windows*amd64*" } | Select-Object -First 1
        if (-not $asset) { Write-Fail "Не найден артефакт windows/amd64 в релизе $latestTag" }
        $downloadUrl = $asset.browser_download_url
        Write-Ok "Доступна версия: $latestTag"
    } catch {
        Write-Host ""
        Write-Host "  Не удалось связаться с GitHub: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "  Установка без сети:" -ForegroundColor Yellow
        Write-Host "  1. Зайдите в браузере на https://github.com/$REPO/releases" -ForegroundColor Yellow
        Write-Host "  2. Скачайте mon-agent-windows-amd64.zip" -ForegroundColor Yellow
        Write-Host "  3. Распакуйте ZIP — там будет mon-agent.exe и install-agent.ps1" -ForegroundColor Yellow
        Write-Host "  4. Запустите install-agent.ps1 из той же папки" -ForegroundColor Yellow
        Write-Host ""
        exit 1
    }
}

# ── Проверяем установленную версию ───────────────────────────────────────────
$currentVersion = ""
if (Test-Path $VERSION_FILE) {
    $currentVersion = (Get-Content $VERSION_FILE -Raw).Trim()
}

if (-not $useLocal -and $currentVersion -eq $latestTag) {
    Write-Warn "Уже установлена актуальная версия $latestTag"
    $force = Read-Host "  Переустановить? [y/N]"
    if ($force -notmatch '^[Yy]$') { Write-Host "Отменено."; exit 0 }
} elseif ($currentVersion -and $currentVersion -ne "local") {
    Write-Host "  Установлена: $currentVersion  →  $latestTag" -ForegroundColor Yellow
} else {
    Write-Host "  Первая установка" -ForegroundColor Cyan
}

# ── Получаем exe ──────────────────────────────────────────────────────────────
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

$svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq 'Running') {
    Write-Step "Останавливаем службу для обновления..."
    Stop-Service -Name $SERVICE_NAME -Force
    Start-Sleep -Seconds 2
}

if ($useLocal) {
    Write-Step "Копируем mon-agent.exe..."
    Copy-Item $localExe "$INSTALL_DIR\mon-agent.exe" -Force
} else {
    Write-Step "Загружаем агент с GitHub..."
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null
    $zipPath = "$tmpDir\mon-agent.zip"

    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Fail "Ошибка загрузки: $_"
    }

    Write-Step "Распаковка..."
    Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

    $exeFile = Get-ChildItem -Path $tmpDir -Filter "mon-agent.exe" -Recurse | Select-Object -First 1
    if (-not $exeFile) { Write-Fail "mon-agent.exe не найден в архиве" }
    Copy-Item $exeFile.FullName "$INSTALL_DIR\mon-agent.exe" -Force
}

if (-not (Test-Path "$INSTALL_DIR\mon-agent.exe")) {
    Write-Fail "Не удалось скопировать mon-agent.exe в $INSTALL_DIR"
}
Write-Ok "Агент: $INSTALL_DIR\mon-agent.exe"

# ── Конфиг — только при первой установке ─────────────────────────────────────
if (-not (Test-Path $CONFIG_FILE)) {
    Write-Host ""
    Write-Host "  Настройка:" -ForegroundColor White
    $serverUrl = Read-Host "  URL сервера [http://10.10.10.10:8080]"
    if ([string]::IsNullOrWhiteSpace($serverUrl)) { $serverUrl = "http://10.10.10.10:8080" }
    $interval = Read-Host "  Интервал отправки [5s]"
    if ([string]::IsNullOrWhiteSpace($interval)) { $interval = "5s" }

    @"
server:
  url: "$serverUrl/api/metrics"
  interval: $interval
"@ | Set-Content -Path $CONFIG_FILE -Encoding UTF8
    Write-Ok "Конфиг создан: $CONFIG_FILE"
} else {
    Write-Ok "Конфиг сохранён без изменений"
}

# ── NSSM ──────────────────────────────────────────────────────────────────────
$nssmExe = $null

# 1. В системе
$nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
if ($nssmCmd) { $nssmExe = $nssmCmd.Source }

# 2. Уже в папке установки
if (-not $nssmExe -and (Test-Path "$INSTALL_DIR\nssm.exe")) {
    $nssmExe = "$INSTALL_DIR\nssm.exe"
}

# 3. Рядом со скриптом (из ZIP-архива)
if (-not $nssmExe) {
    $localNssm = Join-Path $SCRIPT_DIR "nssm.exe"
    if (Test-Path $localNssm) {
        Copy-Item $localNssm "$INSTALL_DIR\nssm.exe" -Force
        $nssmExe = "$INSTALL_DIR\nssm.exe"
        Write-Ok "NSSM взят из архива"
    }
}

# 4. Скачиваем (если нет ни в одном месте)
if (-not $nssmExe) {
    Write-Step "NSSM не найден, загружаем с nssm.cc..."
    if (-not $tmpDir) {
        $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
        New-Item -ItemType Directory -Path $tmpDir | Out-Null
    }
    $nssmZip = "$tmpDir\nssm.zip"
    try {
        Invoke-WebRequest -Uri $NSSM_URL -OutFile $nssmZip -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Fail "Не удалось загрузить NSSM. Скачайте nssm.exe с https://nssm.cc и положите рядом со скриптом, затем запустите снова."
    }
    Expand-Archive -Path $nssmZip -DestinationPath $tmpDir -Force
    $arch = if ([Environment]::Is64BitOperatingSystem) { "win64" } else { "win32" }
    $nssmFound = Get-ChildItem "$tmpDir\nssm-*\$arch\nssm.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $nssmFound) { Write-Fail "nssm.exe не найден в архиве (arch=$arch)" }
    Copy-Item $nssmFound.FullName "$INSTALL_DIR\nssm.exe" -Force
    $nssmExe = "$INSTALL_DIR\nssm.exe"
    Write-Ok "NSSM установлен: $nssmExe"
}

# ── Служба ────────────────────────────────────────────────────────────────────
Write-Step "Настройка службы $SERVICE_NAME..."

$existing = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if (-not $existing) {
    & $nssmExe install $SERVICE_NAME "$INSTALL_DIR\mon-agent.exe"
    & $nssmExe set     $SERVICE_NAME AppDirectory $INSTALL_DIR
    & $nssmExe set     $SERVICE_NAME AppStdout    "$INSTALL_DIR\mon-agent.log"
    & $nssmExe set     $SERVICE_NAME AppStderr    "$INSTALL_DIR\mon-agent-err.log"
    & $nssmExe set     $SERVICE_NAME Start        SERVICE_AUTO_START
    Write-Ok "Служба зарегистрирована"
}

$svcNow = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svcNow -and $svcNow.Status -eq 'Running') {
    & $nssmExe restart $SERVICE_NAME 2>&1 | Out-Null
} else {
    & $nssmExe start $SERVICE_NAME 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
    }
}
Write-Ok "Служба $SERVICE_NAME запущена"

# ── Версия и очистка ──────────────────────────────────────────────────────────
$latestTag | Set-Content -Path $VERSION_FILE -Encoding UTF8
if ($tmpDir) { Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue }

Write-Host ""
Write-Host "  Готово! Агент установлен и работает." -ForegroundColor Green
Write-Host "  Логи:       $INSTALL_DIR\mon-agent.log" -ForegroundColor Gray
Write-Host "  Управление: nssm start/stop/restart $SERVICE_NAME" -ForegroundColor Gray
Write-Host ""
