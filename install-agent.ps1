# LinkMus Monitor — установщик / обновлятор агента для Windows
# Запуск: powershell -ExecutionPolicy Bypass -File install-agent.ps1

#Requires -RunAsAdministrator

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
chcp 65001 | Out-Null

$REPO        = "RomanMusalikin/linkmus-monitor"
$INSTALL_DIR = "C:\mon-agent"
$CONFIG_FILE = "$INSTALL_DIR\agent-config.yaml"
$VERSION_FILE= "$INSTALL_DIR\.version"
$SERVICE_NAME= "MonAgent"
$NSSM_URL    = "https://nssm.cc/release/nssm-2.24.zip"

function Write-Step { Write-Host "[....] $args" -ForegroundColor Cyan }
function Write-Ok   { Write-Host "[ OK ] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Fail { Write-Host "[ERR]  $args" -ForegroundColor Red; exit 1 }

# ── Заголовок ─────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "╔══════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  LinkMus Monitor — Установщик агента    ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ── Последний релиз с GitHub ──────────────────────────────────────────────────
Write-Step "Получение последней версии с GitHub..."
try {
    $release    = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest"
    $latestTag  = $release.tag_name
    $asset      = $release.assets | Where-Object { $_.name -like "*windows*amd64*" } | Select-Object -First 1
    if (-not $asset) { Write-Fail "Не найден артефакт windows/amd64 в релизе $latestTag" }
    $downloadUrl = $asset.browser_download_url
    Write-Ok "Доступна версия: $latestTag"
} catch {
    Write-Fail "Не удалось получить релиз с GitHub: $_"
}

# ── Проверяем текущую версию ──────────────────────────────────────────────────
$currentVersion = ""
if (Test-Path $VERSION_FILE) {
    $currentVersion = (Get-Content $VERSION_FILE -Raw).Trim()
}

if ($currentVersion -eq $latestTag) {
    Write-Warn "Уже установлена актуальная версия $latestTag"
    $force = Read-Host "  Всё равно переустановить? [y/N]"
    if ($force -notmatch '^[Yy]$') { Write-Host "Отменено."; exit 0 }
} elseif ($currentVersion) {
    Write-Host "  Установлена: $currentVersion  →  доступна: $latestTag" -ForegroundColor Yellow
} else {
    Write-Host "  Первая установка версии $latestTag" -ForegroundColor Cyan
}

# ── Загрузка и распаковка ─────────────────────────────────────────────────────
Write-Step "Загрузка агента..."
$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmpDir | Out-Null
$zipPath = "$tmpDir\mon-agent.zip"

try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
} catch {
    Write-Fail "Ошибка загрузки: $_"
}

Write-Step "Распаковка..."
Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

# Останавливаем службу перед заменой бинарника
$svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq 'Running') {
    Write-Step "Останавливаем службу для обновления..."
    Stop-Service -Name $SERVICE_NAME -Force
    Start-Sleep -Seconds 2
}

Copy-Item "$tmpDir\mon-agent.exe" "$INSTALL_DIR\mon-agent.exe" -Force
Write-Ok "Бинарник обновлён: $INSTALL_DIR\mon-agent.exe"

# ── Конфиг — только при первой установке ─────────────────────────────────────
if (-not (Test-Path $CONFIG_FILE)) {
    Write-Host ""
    Write-Host "Настройка агента:" -ForegroundColor White
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
$nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
$nssmExe = if ($nssmCmd) { $nssmCmd.Source } else { $null }
if (-not $nssmExe) {
    $nssmLocal = "$INSTALL_DIR\nssm.exe"
    if (Test-Path $nssmLocal) {
        $nssmExe = $nssmLocal
    } else {
        Write-Step "NSSM не найден, загружаем..."
        $nssmZip = "$tmpDir\nssm.zip"
        Invoke-WebRequest -Uri $NSSM_URL -OutFile $nssmZip -UseBasicParsing
        Expand-Archive -Path $nssmZip -DestinationPath $tmpDir -Force
        $nssmExe = (Get-ChildItem "$tmpDir\nssm-*\win64\nssm.exe" | Select-Object -First 1).FullName
        Copy-Item $nssmExe $nssmLocal -Force
        $nssmExe = $nssmLocal
        Write-Ok "NSSM: $nssmExe"
    }
}

# ── Служба: установка или перезапуск ─────────────────────────────────────────
Write-Step "Настройка службы $SERVICE_NAME..."

$existing = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if (-not $existing) {
    # Первая установка — регистрируем службу
    & $nssmExe install    $SERVICE_NAME "$INSTALL_DIR\mon-agent.exe"
    & $nssmExe set        $SERVICE_NAME AppDirectory  $INSTALL_DIR
    & $nssmExe set        $SERVICE_NAME AppStdout     "$INSTALL_DIR\mon-agent.log"
    & $nssmExe set        $SERVICE_NAME AppStderr     "$INSTALL_DIR\mon-agent-err.log"
    & $nssmExe set        $SERVICE_NAME Start         SERVICE_AUTO_START
    Write-Ok "Служба зарегистрирована"
}

& $nssmExe start $SERVICE_NAME 2>$null
if ($LASTEXITCODE -ne 0) {
    Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
}
Write-Ok "Служба $SERVICE_NAME запущена"

# ── Сохраняем версию ──────────────────────────────────────────────────────────
$latestTag | Set-Content -Path $VERSION_FILE -Encoding UTF8

# ── Очистка ───────────────────────────────────────────────────────────────────
Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Агент $latestTag установлен!" -ForegroundColor Green
Write-Host "  Логи:       $INSTALL_DIR\mon-agent.log"
Write-Host "  Управление: nssm start/stop/restart $SERVICE_NAME"
Write-Host ""
