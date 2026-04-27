# LinkMus Monitor - CLI for Windows
# Usage: mon <agent> <start|stop|restart|status|enable|disable|logs|update|help>
# Installed to C:\mon-agent\mon.ps1, called via mon.cmd shim in System32

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$REPO         = "RomanMusalikin/linkmus-monitor"
$INSTALL_DIR  = "C:\mon-agent"
$LOG_FILE     = "$INSTALL_DIR\mon-agent.log"
$VERSION_FILE = "$INSTALL_DIR\.version"
$SERVICE_NAME = "MonAgent"

function Write-Ok   { Write-Host "[ OK ] $args" -ForegroundColor Green }
function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Cyan }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Err  { Write-Host "[ERR]  $args" -ForegroundColor Red }

function Show-Usage {
    Write-Host ""
    Write-Host "LinkMus Monitor" -ForegroundColor Cyan
    Write-Host "Usage: " -NoNewline; Write-Host "mon agent <command>" -ForegroundColor White
    Write-Host ""
    Write-Host "  start    " -NoNewline -ForegroundColor White; Write-Host "Start the agent service"
    Write-Host "  stop     " -NoNewline -ForegroundColor White; Write-Host "Stop the agent service"
    Write-Host "  restart  " -NoNewline -ForegroundColor White; Write-Host "Restart the agent service"
    Write-Host "  status   " -NoNewline -ForegroundColor White; Write-Host "Show service status and recent log"
    Write-Host "  enable   " -NoNewline -ForegroundColor White; Write-Host "Enable autostart on boot"
    Write-Host "  disable  " -NoNewline -ForegroundColor White; Write-Host "Disable autostart on boot"
    Write-Host "  logs     " -NoNewline -ForegroundColor White; Write-Host "Follow live log output"
    Write-Host "  update   " -NoNewline -ForegroundColor White; Write-Host "Check for updates and install if available"
    Write-Host ""
}

function Do-Update {
    if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Err "Run as Administrator: right-click terminal -> Run as Administrator"
        exit 1
    }

    Write-Info "Checking for updates..."
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest" -ErrorAction Stop
    } catch {
        Write-Err "Cannot reach GitHub: $_"
        exit 1
    }

    $latestTag = $release.tag_name
    $current   = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "" }

    if ($current -eq $latestTag) {
        Write-Warn "Already up to date: $latestTag"
        $ans = Read-Host "  Reinstall anyway? [y/N]"
        if ($ans -notmatch '^[Yy]$') { Write-Host "Cancelled."; return }
    } else {
        $from = if ($current) { $current } else { "not installed" }
        Write-Host "  Update available: " -NoNewline
        Write-Host "$from -> $latestTag" -ForegroundColor Yellow
        $ans = Read-Host "  Install? [y/N]"
        if ($ans -notmatch '^[Yy]$') { Write-Host "Cancelled."; return }
    }

    $asset = $release.assets | Where-Object { $_.name -like "*windows*amd64*" } | Select-Object -First 1
    if (-not $asset) { Write-Err "No windows/amd64 artifact found in release $latestTag"; exit 1 }

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
    New-Item -ItemType Directory -Path $tmpDir | Out-Null

    Write-Info "Downloading $latestTag..."
    try {
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$tmpDir\mon-agent.zip" -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-Err "Download failed: $_"
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
        exit 1
    }

    Expand-Archive -Path "$tmpDir\mon-agent.zip" -DestinationPath $tmpDir -Force
    $exeFile = Get-ChildItem -Path $tmpDir -Filter "mon-agent.exe" -Recurse | Select-Object -First 1
    if (-not $exeFile) { Write-Err "mon-agent.exe not found in archive"; exit 1 }

    $svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
    if ($svc -and $svc.Status -eq 'Running') {
        Stop-Service -Name $SERVICE_NAME -Force
        Start-Sleep -Seconds 2
    }

    Copy-Item $exeFile.FullName "$INSTALL_DIR\mon-agent.exe" -Force
    $latestTag | Set-Content -Path $VERSION_FILE -Encoding UTF8
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue

    Start-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue
    Write-Ok "Agent updated to $latestTag"
}

# -- Entry point --

if ($args.Count -lt 2) { Show-Usage; exit 1 }

$target  = $args[0].ToLower()
$command = $args[1].ToLower()

if ($target -eq "help" -or $target -eq "--help" -or $target -eq "-h") { Show-Usage; exit 0 }
if ($target -ne "agent") { Write-Err "Unknown target: $target"; Show-Usage; exit 1 }

$svc = Get-Service -Name $SERVICE_NAME -ErrorAction SilentlyContinue

switch ($command) {
    "start" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        Start-Service -Name $SERVICE_NAME
        Write-Ok "$SERVICE_NAME started"
    }
    "stop" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        Stop-Service -Name $SERVICE_NAME -Force
        Write-Ok "$SERVICE_NAME stopped"
    }
    "restart" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        Restart-Service -Name $SERVICE_NAME -Force
        Write-Ok "$SERVICE_NAME restarted"
    }
    "status" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        $s = Get-Service -Name $SERVICE_NAME
        $color = if ($s.Status -eq 'Running') { 'Green' } else { 'Red' }
        Write-Host ""
        Write-Host "  Service : " -NoNewline; Write-Host $SERVICE_NAME -ForegroundColor White
        Write-Host "  Status  : " -NoNewline; Write-Host $s.Status -ForegroundColor $color
        Write-Host "  Startup : " -NoNewline; Write-Host $s.StartType
        $ver = if (Test-Path $VERSION_FILE) { (Get-Content $VERSION_FILE -Raw).Trim() } else { "unknown" }
        Write-Host "  Version : " -NoNewline; Write-Host $ver
        Write-Host "  Log     : $LOG_FILE"
        Write-Host ""
        if (Test-Path $LOG_FILE) {
            Write-Host "--- Last 20 log lines ---" -ForegroundColor DarkGray
            Get-Content $LOG_FILE -Tail 20 -Encoding UTF8
            Write-Host "-------------------------" -ForegroundColor DarkGray
        }
        Write-Host ""
    }
    "enable" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        Set-Service -Name $SERVICE_NAME -StartupType Automatic
        Write-Ok "Autostart enabled"
    }
    "disable" {
        if (-not $svc) { Write-Err "Service $SERVICE_NAME not found"; exit 1 }
        Set-Service -Name $SERVICE_NAME -StartupType Disabled
        Write-Ok "Autostart disabled"
    }
    "logs" {
        if (-not (Test-Path $LOG_FILE)) { Write-Err "Log file not found: $LOG_FILE"; exit 1 }
        Write-Info "Following $LOG_FILE (Ctrl+C to stop)..."
        Get-Content $LOG_FILE -Wait -Tail 50 -Encoding UTF8
    }
    "update" {
        Do-Update
    }
    "help" {
        Show-Usage
    }
    default {
        Write-Err "Unknown command: $command"
        Show-Usage
        exit 1
    }
}
