# LinkMus Monitor - Agent Installer for Windows
# Usage: powershell -ExecutionPolicy Bypass -File install-agent.ps1
#
# Modes:
#   LOCAL  - mon-agent.exe is next to this script, no internet needed.
#            Download ZIP from releases page via browser and extract.
#   ONLINE - downloads latest release from GitHub automatically.

#Requires -RunAsAdministrator

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$REPO         = "RomanMusalikin/linkmus-monitor"
$INSTALL_DIR  = "C:\mon-agent"
$CONFIG_FILE  = "$INSTALL_DIR\agent-config.yaml"
$VERSION_FILE = "$INSTALL_DIR\.version"
$SERVICE_NAME = "MonAgent"
$SCRIPT_DIR   = Split-Path -Parent $MyInvocation.MyCommand.Path

function Write-Step { Write-Host "[....] $args" -ForegroundColor Cyan }
function Write-Ok   { Write-Host "[ OK ] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Fail {
    param([string]$msg)
    Write-Host "[ERR]  $msg" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  LinkMus Monitor - Agent Installer            " -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# -- Mode: local or online ----------------------------------------------------
$localExe = Join-Path $SCRIPT_DIR "mon-agent.exe"
$useLocal = Test-Path $localExe

if ($useLocal) {
    Write-Ok "Found mon-agent.exe next to script - working offline."
    $latestTag = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "local" }
} else {
    Write-Step "mon-agent.exe not found locally - trying GitHub..."
    try {
        $release     = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
        $latestTag   = $release.tag_name
        $asset       = $release.assets | Where-Object { $_.name -like "*windows*amd64*" } | Select-Object -First 1
        if (-not $asset) { Write-Fail "No windows/amd64 artifact found in release $latestTag" }
        $downloadUrl = $asset.browser_download_url
        Write-Ok "Version: $latestTag"
    } catch {
        Write-Host ""
        Write-Host "  Cannot reach GitHub: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "  Offline install steps:" -ForegroundColor Yellow
        Write-Host "  1. Open in browser: https://github.com/$REPO/releases" -ForegroundColor Yellow
        Write-Host "  2. Download mon-agent-windows-amd64.zip" -ForegroundColor Yellow
        Write-Host "  3. Extract ZIP - it contains mon-agent.exe and install-agent.ps1" -ForegroundColor Yellow
        Write-Host "  4. Run install-agent.ps1 from that folder as Administrator" -ForegroundColor Yellow
        Write-Host ""
        exit 1
    }
}

# -- Check installed version --------------------------------------------------
$currentVersion = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "" }

if (-not $useLocal -and $currentVersion -eq $latestTag) {
    Write-Warn "Already installed: $latestTag"
    $force = Read-Host "  Reinstall? [y/N]"
    if ($force -notmatch '^[Yy]$') { Write-Host "Cancelled."; exit 0 }
} elseif ($currentVersion) {
    Write-Host "  Installed: $currentVersion  ->  $latestTag" -ForegroundColor Yellow
} else {
    Write-Host "  First install" -ForegroundColor Cyan
}

# -- Stop service if running --------------------------------------------------
$svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq 'Running') {
    Write-Step "Stopping service for update..."
    Stop-Service -Name $SERVICE_NAME -Force
    Start-Sleep -Seconds 2
}

# -- Copy / download exe ------------------------------------------------------
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

if ($useLocal) {
    Write-Step "Copying mon-agent.exe..."
    Copy-Item $localExe "$INSTALL_DIR\mon-agent.exe" -Force
} else {
    Write-Step "Downloading agent from GitHub..."
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile "$tmpDir\mon-agent.zip" -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Fail "Download error: $_"
    }
    Expand-Archive -Path "$tmpDir\mon-agent.zip" -DestinationPath $tmpDir -Force
    $exeFile = Get-ChildItem -Path $tmpDir -Filter "mon-agent.exe" -Recurse | Select-Object -First 1
    if (-not $exeFile) { Write-Fail "mon-agent.exe not found in archive" }
    Copy-Item $exeFile.FullName "$INSTALL_DIR\mon-agent.exe" -Force
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}

if (-not (Test-Path "$INSTALL_DIR\mon-agent.exe")) {
    Write-Fail "Failed to copy mon-agent.exe to $INSTALL_DIR"
}
Write-Ok "Agent: $INSTALL_DIR\mon-agent.exe"

# -- Config -------------------------------------------------------------------
Write-Host ""
if (Test-Path $CONFIG_FILE) {
    Write-Host "  Current config:" -ForegroundColor White
    Get-Content $CONFIG_FILE | ForEach-Object { Write-Host "    $_" -ForegroundColor Gray }
    Write-Host ""
    $change = Read-Host "  Change settings? [y/N]"
    $doConfig = $change -match '^[Yy]$'
} else {
    Write-Host "  Configuration:" -ForegroundColor White
    $doConfig = $true
}

if ($doConfig) {
    $serverUrl = Read-Host "  Server URL [http://10.10.10.10:8080]"
    if ([string]::IsNullOrWhiteSpace($serverUrl)) { $serverUrl = "http://10.10.10.10:8080" }
    $intervalRaw = Read-Host "  Send interval in seconds [5]"
    if ([string]::IsNullOrWhiteSpace($intervalRaw)) { $intervalRaw = "5" }
    $interval = $intervalRaw.Trim().TrimEnd('s').Trim() + "s"

    @"
server:
  url: "$serverUrl/api/metrics"
  interval: $interval
"@ | Set-Content -Path $CONFIG_FILE -Encoding UTF8
    Write-Ok "Config: $CONFIG_FILE"
} else {
    Write-Ok "Config unchanged"
}

# -- Service via built-in New-Service (no NSSM) -------------------------------
Write-Step "Configuring service $SERVICE_NAME..."

$existing = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($existing) {
    sc.exe delete $SERVICE_NAME | Out-Null
    Start-Sleep -Seconds 1
}

$result = New-Service `
    -Name        $SERVICE_NAME `
    -BinaryPathName "$INSTALL_DIR\mon-agent.exe" `
    -DisplayName "LinkMus Monitor Agent" `
    -Description "LinkMus monitoring agent - sends metrics to server" `
    -StartupType Automatic `
    -ErrorAction Stop

sc.exe failure $SERVICE_NAME reset= 86400 actions= restart/5000 | Out-Null

Write-Ok "Service registered"

Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
$svcFinal = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
if ($svcFinal -and $svcFinal.Status -eq 'Running') {
    Write-Ok "Service $SERVICE_NAME is running"
} else {
    Write-Warn "Service created but not started. Check log: $INSTALL_DIR\mon-agent.log"
}

# -- Version ------------------------------------------------------------------
$latestTag | Set-Content -Path $VERSION_FILE -Encoding UTF8

# -- Install mon CLI ----------------------------------------------------------
Write-Step "Installing mon CLI..."

$monPs1Src = Join-Path $SCRIPT_DIR "mon.ps1"
if (Test-Path $monPs1Src) {
    Copy-Item $monPs1Src "$INSTALL_DIR\mon.ps1" -Force
} else {
    # Download mon.ps1 from GitHub if not next to script
    try {
        Invoke-WebRequest -Uri "https://raw.githubusercontent.com/$REPO/main/mon.ps1" `
            -OutFile "$INSTALL_DIR\mon.ps1" -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Warn "Could not download mon.ps1: $_"
    }
}

# Create mon.cmd shim in System32 so 'mon' works from any terminal
$shimPath = "C:\Windows\System32\mon.cmd"
@"
@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "$INSTALL_DIR\mon.ps1" %*
"@ | Set-Content -Path $shimPath -Encoding ASCII
Write-Ok "CLI: mon agent start|stop|restart|status|enable|disable|logs|update"

Write-Host ""
Write-Host "  Done! Agent installed." -ForegroundColor Green
Write-Host "  Log:    $INSTALL_DIR\mon-agent.log" -ForegroundColor Gray
Write-Host "  Usage:  mon agent start|stop|restart|status|logs|update" -ForegroundColor Gray
Write-Host ""
