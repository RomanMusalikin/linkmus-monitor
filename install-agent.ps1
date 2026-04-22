# LinkMus Monitor — установщик агента для Windows
# Запуск: powershell -ExecutionPolicy Bypass -File install-agent.ps1
#
# Режимы:
#   ЛОКАЛЬНЫЙ — mon-agent.exe лежит рядом со скриптом, сеть не нужна.
#               Скачай ZIP со страницы релизов через браузер и распакуй.
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
$SCRIPT_DIR   = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step { Write-Host "[....] $args" -ForegroundColor Cyan }
function Write-Ok   { Write-Host "[ OK ] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Fail { param([string]$msg) Write-Host "[ERR]  $msg" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "╔══════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  LinkMus Monitor — Установщик агента    ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ── Режим: локальный или онлайн ───────────────────────────────────────────────
$localExe = Join-Path $SCRIPT_DIR "mon-agent.exe"
$useLocal = Test-Path $localExe

if ($useLocal) {
    Write-Ok "Найден mon-agent.exe рядом со скриптом — работаем без сети."
    $latestTag = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "local" }
} else {
    Write-Step "mon-agent.exe не найден рядом — пробуем скачать с GitHub..."
    try {
        $release     = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
        $latestTag   = $release.tag_name
        $asset       = $release.assets | Where-Object { $_.name -like "*windows*amd64*" } | Select-Object -First 1
        if (-not $asset) { Write-Fail "Не найден артефакт windows/amd64 в релизе $latestTag" }
        $downloadUrl = $asset.browser_download_url
        Write-Ok "Версия: $latestTag"
    } catch {
        Write-Host ""
        Write-Host "  Не удалось связаться с GitHub: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "  Установка без сети:" -ForegroundColor Yellow
        Write-Host "  1. Откройте в браузере: https://github.com/$REPO/releases" -ForegroundColor Yellow
        Write-Host "  2. Скачайте mon-agent-windows-amd64.zip" -ForegroundColor Yellow
        Write-Host "  3. Распакуйте ZIP — там будет mon-agent.exe и install-agent.ps1" -ForegroundColor Yellow
        Write-Host "  4. Запустите install-agent.ps1 из той же папки от имени Администратора" -ForegroundColor Yellow
        Write-Host ""
        exit 1
    }
}

# ── Проверяем установленную версию ───────────────────────────────────────────
$currentVersion = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "" }

if (-not $useLocal -and $currentVersion -eq $latestTag) {
    Write-Warn "Уже установлена актуальная версия $latestTag"
    $force = Read-Host "  Переустановить? [y/N]"
    if ($force -notmatch '^[Yy]$') { Write-Host "Отменено."; exit 0 }
} elseif ($currentVersion) {
    Write-Host "  Установлена: $currentVersion  →  $latestTag" -ForegroundColor Yellow
} else {
    Write-Host "  Первая установка" -ForegroundColor Cyan
}

# ── Останавливаем службу если работает ───────────────────────────────────────
$svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq 'Running') {
    Write-Step "Останавливаем службу для обновления..."
    Stop-Service -Name $SERVICE_NAME -Force
    Start-Sleep -Seconds 2
}

# ── Копируем / скачиваем exe ──────────────────────────────────────────────────
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

if ($useLocal) {
    Write-Step "Копируем mon-agent.exe..."
    Copy-Item $localExe "$INSTALL_DIR\mon-agent.exe" -Force
} else {
    Write-Step "Загружаем агент с GitHub..."
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile "$tmpDir\mon-agent.zip" -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Fail "Ошибка загрузки: $_"
    }
    Expand-Archive -Path "$tmpDir\mon-agent.zip" -DestinationPath $tmpDir -Force
    $exeFile = Get-ChildItem -Path $tmpDir -Filter "mon-agent.exe" -Recurse | Select-Object -First 1
    if (-not $exeFile) { Write-Fail "mon-agent.exe не найден в архиве" }
    Copy-Item $exeFile.FullName "$INSTALL_DIR\mon-agent.exe" -Force
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
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
    Write-Ok "Конфиг: $CONFIG_FILE"
} else {
    Write-Ok "Конфиг сохранён без изменений"
}

# ── Служба через встроенный sc.exe / New-Service (без NSSM) ──────────────────
Write-Step "Настройка службы $SERVICE_NAME..."

$existing = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($existing) {
    # Удаляем старую службу чтобы обновить параметры
    sc.exe delete $SERVICE_NAME | Out-Null
    Start-Sleep -Seconds 1
}

# Регистрируем службу
$result = New-Service `
    -Name        $SERVICE_NAME `
    -BinaryPathName "$INSTALL_DIR\mon-agent.exe" `
    -DisplayName "LinkMus Monitor Agent" `
    -Description "Агент мониторинга LinkMus — отправляет метрики на сервер" `
    -StartupType Automatic `
    -ErrorAction Stop

# Настраиваем восстановление после сбоя: перезапуск через 5 секунд
sc.exe failure $SERVICE_NAME reset= 86400 actions= restart/5000 | Out-Null

Write-Ok "Служба зарегистрирована"

# Запускаем
Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
$svcFinal = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svcFinal -and $svcFinal.Status -eq 'Running') {
    Write-Ok "Служба $SERVICE_NAME запущена"
} else {
    Write-Warn "Служба создана, но не запустилась. Проверьте лог: $INSTALL_DIR\mon-agent.log"
}

# ── Версия ────────────────────────────────────────────────────────────────────
$latestTag | Set-Content -Path $VERSION_FILE -Encoding UTF8

Write-Host ""
Write-Host "  Готово! Агент установлен." -ForegroundColor Green
Write-Host "  Логи:       $INSTALL_DIR\mon-agent.log" -ForegroundColor Gray
Write-Host "  Управление: Start-Service / Stop-Service / Restart-Service $SERVICE_NAME" -ForegroundColor Gray
Write-Host ""
