#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# LinkMus Monitor — установщик / обновлятор
# Использование: curl -sSL https://raw.githubusercontent.com/RomanMusalikin/linkmus-monitor/main/install.sh | sudo bash
# ─────────────────────────────────────────────────────────────────────────────

REPO="RomanMusalikin/linkmus-monitor"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
VERSION_FILE="/opt/linkmus-monitor/.version"
AGENT_VERSION_FILE="/opt/mon-agent/.version"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info()    { echo -e "${CYAN}[INFO]${RESET} $*"; }
success() { echo -e "${GREEN}[OK]${RESET}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET} $*"; }
error()   { echo -e "${RED}[ERR]${RESET}  $*" >&2; exit 1; }

require() { command -v "$1" &>/dev/null || error "Требуется '$1', но он не установлен"; }

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *) error "Неподдерживаемая архитектура: $(uname -m)" ;;
  esac
}

# Возвращает tag_name и asset URL последнего релиза
fetch_release_info() {
  local name="$1"
  require curl
  local json; json=$(curl -fsSL "$GITHUB_API")
  LATEST_TAG=$(echo "$json" | grep -o '"tag_name": *"[^"]*"' | head -1 | grep -o '"[^"]*"$' | tr -d '"')
  ASSET_URL=$(echo "$json" \
    | grep -o "\"browser_download_url\": *\"[^\"]*${name}[^\"]*\"" \
    | head -1 | grep -o 'https://[^"]*')
  [ -n "$LATEST_TAG" ] || error "Не удалось получить версию из GitHub"
  [ -n "$ASSET_URL"  ] || error "Не найден артефакт '${name}' в релизе ${LATEST_TAG}"
}

download() {
  local url="$1" dest="$2"
  info "Загрузка $(basename "$dest") ..."
  curl -fsSL --progress-bar -o "$dest" "$url" || error "Не удалось загрузить $url"
}

# ── Проверяем: установлен ли уже компонент, и нужно ли обновление ─────────────
check_version() {
  local version_file="$1"
  local current=""
  [ -f "$version_file" ] && current=$(cat "$version_file")
  if [ -n "$current" ]; then
    if [ "$current" = "$LATEST_TAG" ]; then
      warn "Уже установлена актуальная версия ${BOLD}${current}${RESET}"
      read -rp "  Всё равно переустановить? [y/N]: " force </dev/tty
      [[ "$force" =~ ^[Yy]$ ]] || { info "Отменено."; exit 0; }
    else
      info "Установлена: ${BOLD}${current}${RESET} → доступна: ${BOLD}${LATEST_TAG}${RESET}"
    fi
  else
    info "Первая установка версии ${BOLD}${LATEST_TAG}${RESET}"
  fi
}

# ══════════════════════════════════════════════════════════════════════════════
# УСТАНОВКА / ОБНОВЛЕНИЕ СЕРВЕРА
# ══════════════════════════════════════════════════════════════════════════════
install_server() {
  require curl; require tar; require systemctl
  local arch; arch=$(detect_arch)
  [ "$arch" = "amd64" ] || error "Сервер поддерживается только на amd64"

  fetch_release_info "mon-server-linux-amd64.tar.gz"
  check_version "$VERSION_FILE"

  local tmp; tmp=$(mktemp -d)
  trap "rm -rf $tmp" EXIT

  download "$ASSET_URL" "$tmp/mon-server.tar.gz"
  tar -xzf "$tmp/mon-server.tar.gz" -C "$tmp"

  # Останавливаем если уже запущен
  if systemctl is-active --quiet mon-server 2>/dev/null; then
    info "Останавливаем mon-server для обновления..."
    systemctl stop mon-server
  fi

  # Бинарник
  install -Dm755 "$tmp/mon-server" /usr/local/bin/mon-server
  success "Бинарник обновлён: /usr/local/bin/mon-server"

  # Фронтенд
  mkdir -p /opt/linkmus-monitor/web
  rm -rf /opt/linkmus-monitor/web/*
  cp -r "$tmp/web-dist/." /opt/linkmus-monitor/web/
  success "Фронтенд обновлён: /opt/linkmus-monitor/web/"

  mkdir -p /opt/linkmus-monitor/data

  # Systemd unit (перезаписываем при каждой установке — вдруг изменился)
  cat > /etc/systemd/system/mon-server.service <<'EOF'
[Unit]
Description=LinkMus Monitor Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/linkmus-monitor/data
ExecStart=/usr/local/bin/mon-server
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable mon-server
  systemctl restart mon-server
  success "Служба mon-server запущена"

  # Nginx — только при первой установке
  if command -v nginx &>/dev/null; then
    if [ ! -f /etc/nginx/sites-available/linkmus-monitor ]; then
      cat > /etc/nginx/sites-available/linkmus-monitor <<'EOF'
server {
    listen 80;
    server_name _;

    root /opt/linkmus-monitor/web;
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
EOF
      ln -sf /etc/nginx/sites-available/linkmus-monitor /etc/nginx/sites-enabled/
      rm -f /etc/nginx/sites-enabled/default
      nginx -t && systemctl reload nginx
      success "Nginx настроен"
    else
      systemctl reload nginx
      success "Nginx перезагружен"
    fi
  else
    warn "Nginx не найден — интерфейс доступен напрямую на :8080"
  fi

  # Сохраняем версию
  echo "$LATEST_TAG" > "$VERSION_FILE"

  install_mon_cli

  echo ""
  success "Сервер ${BOLD}${LATEST_TAG}${RESET} установлен!"
  echo -e "  Веб-интерфейс: ${BOLD}http://$(hostname -I | awk '{print $1}')${RESET}"
  echo -e "  Управление:    ${BOLD}mon server start|stop|restart|status|logs${RESET}"
}

# ══════════════════════════════════════════════════════════════════════════════
# УСТАНОВКА / ОБНОВЛЕНИЕ АГЕНТА (Linux)
# ══════════════════════════════════════════════════════════════════════════════
install_agent_linux() {
  require curl; require systemctl

  local arch; arch=$(detect_arch)
  local asset_name="mon-agent-linux"
  [ "$arch" = "arm64" ] && asset_name="mon-agent-linux-arm64"

  fetch_release_info "$asset_name"
  check_version "$AGENT_VERSION_FILE"

  local tmp; tmp=$(mktemp -d)
  trap "rm -rf $tmp" EXIT

  download "$ASSET_URL" "$tmp/mon-agent"

  # Останавливаем если запущен
  if systemctl is-active --quiet mon-agent 2>/dev/null; then
    info "Останавливаем mon-agent для обновления..."
    systemctl stop mon-agent
  fi

  install -Dm755 "$tmp/mon-agent" /usr/local/bin/mon-agent
  success "Бинарник обновлён: /usr/local/bin/mon-agent"

  # Конфиг — только при первой установке, при обновлении не трогаем
  mkdir -p /opt/mon-agent
  if [ ! -f /opt/mon-agent/agent-config.yaml ]; then
    echo ""
    echo -e "${BOLD}Настройка агента:${RESET}"
    read -rp "  URL сервера [http://10.10.10.10:8080]: " server_url </dev/tty
    server_url="${server_url:-http://10.10.10.10:8080}"
    read -rp "  Интервал отправки [5s]: " interval </dev/tty
    interval="${interval:-5s}"
    cat > /opt/mon-agent/agent-config.yaml <<EOF
server:
  url: "${server_url}/api/metrics"
  interval: ${interval}
EOF
    success "Конфиг создан: /opt/mon-agent/agent-config.yaml"
  else
    success "Конфиг сохранён без изменений"
  fi

  # Systemd unit
  cat > /etc/systemd/system/mon-agent.service <<'EOF'
[Unit]
Description=LinkMus Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/mon-agent
ExecStart=/usr/local/bin/mon-agent
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable mon-agent
  systemctl restart mon-agent
  success "Служба mon-agent запущена"

  # Сохраняем версию
  echo "$LATEST_TAG" > "$AGENT_VERSION_FILE"

  install_mon_cli

  echo ""
  success "Агент ${BOLD}${LATEST_TAG}${RESET} установлен!"
  echo -e "  Управление: ${BOLD}mon agent start|stop|restart|status|logs${RESET}"
}

# ══════════════════════════════════════════════════════════════════════════════
# УСТАНОВКА CLI-ОБЁРТКИ mon
# ══════════════════════════════════════════════════════════════════════════════
install_mon_cli() {
  cat > /usr/local/bin/mon <<'MONSCRIPT'
#!/usr/bin/env bash
# LinkMus Monitor — управление службами
# Использование: mon <server|agent> <команда>

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

usage() {
  echo -e "${BOLD}Использование:${RESET} mon <server|agent> <команда>"
  echo ""
  echo -e "  ${BOLD}Команды:${RESET}"
  echo "    start    — запустить службу"
  echo "    stop     — остановить службу"
  echo "    restart  — перезапустить службу"
  echo "    status   — статус службы"
  echo "    enable   — включить автозапуск"
  echo "    disable  — выключить автозапуск"
  echo "    logs     — следить за логами (Ctrl+C для выхода)"
  echo ""
  echo -e "  ${BOLD}Примеры:${RESET}"
  echo "    mon server start"
  echo "    mon agent logs"
  echo "    mon server enable"
  exit 1
}

[ $# -lt 2 ] && usage

case "$1" in
  server) SERVICE="mon-server" ;;
  agent)  SERVICE="mon-agent" ;;
  *)
    echo -e "${RED}Неизвестная цель:${RESET} $1 (используйте 'server' или 'agent')"
    usage
    ;;
esac

case "$2" in
  start)
    systemctl start "$SERVICE"
    echo -e "${GREEN}[OK]${RESET} $SERVICE запущен"
    ;;
  stop)
    systemctl stop "$SERVICE"
    echo -e "${GREEN}[OK]${RESET} $SERVICE остановлен"
    ;;
  restart)
    systemctl restart "$SERVICE"
    echo -e "${GREEN}[OK]${RESET} $SERVICE перезапущен"
    ;;
  status)
    systemctl status "$SERVICE" --no-pager
    ;;
  enable)
    systemctl enable "$SERVICE"
    echo -e "${GREEN}[OK]${RESET} Автозапуск $SERVICE включён"
    ;;
  disable)
    systemctl disable "$SERVICE"
    echo -e "${GREEN}[OK]${RESET} Автозапуск $SERVICE выключен"
    ;;
  logs)
    echo -e "${CYAN}Логи $SERVICE (Ctrl+C для выхода):${RESET}"
    journalctl -fu "$SERVICE"
    ;;
  *)
    echo -e "${RED}Неизвестная команда:${RESET} $2"
    usage
    ;;
esac
MONSCRIPT

  chmod +x /usr/local/bin/mon
  success "CLI-обёртка установлена: mon server|agent start|stop|restart|status|enable|disable|logs"
}

# ══════════════════════════════════════════════════════════════════════════════
# ГЛАВНОЕ МЕНЮ
# ══════════════════════════════════════════════════════════════════════════════
echo ""
echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${CYAN}║      LinkMus Monitor — Установщик       ║${RESET}"
echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════╝${RESET}"
echo ""

[ "$(id -u)" -eq 0 ] || error "Запустите от root: sudo bash install.sh"

echo -e "Что установить / обновить?"
echo -e "  ${BOLD}1${RESET}) Сервер мониторинга (mon-server + nginx)"
echo -e "  ${BOLD}2${RESET}) Агент — Linux (amd64 / arm64)"
echo -e "  ${BOLD}3${RESET}) Агент — Windows (инструкция)"
echo ""
read -rp "Выбор [1-3]: " choice </dev/tty

case "$choice" in
  1) install_server ;;
  2) install_agent_linux ;;
  3)
    echo ""
    echo -e "${BOLD}Установка / обновление агента на Windows:${RESET}"
    echo ""
    echo "  Запустите PowerShell от имени администратора:"
    echo -e "  ${CYAN}powershell -ExecutionPolicy Bypass -Command \"& { iwr https://raw.githubusercontent.com/${REPO}/main/install-agent.ps1 | iex }\"${RESET}"
    echo ""
    echo "  Скрипт сам определит: первая установка или обновление."
    ;;
  *) error "Неверный выбор: $choice" ;;
esac
