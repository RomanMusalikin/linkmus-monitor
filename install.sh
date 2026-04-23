#!/usr/bin/env bash
set -euo pipefail

REPO="RomanMusalikin/linkmus-monitor"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

SERVER_BIN="/usr/local/bin/mon-server"
SERVER_DIR="/opt/linkmus-monitor"
SERVER_DATA="$SERVER_DIR/data"
SERVER_WEB="$SERVER_DIR/web"
SERVER_VERSION="$SERVER_DIR/.version"

AGENT_DIR="/opt/mon-agent"
AGENT_BIN="$AGENT_DIR/mon-agent"
AGENT_CFG="$AGENT_DIR/agent-config.yaml"
AGENT_VERSION="$AGENT_DIR/.version"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info()    { echo -e "${CYAN}[INFO]${RESET} $*"; }
ok()      { echo -e "${GREEN}[ OK ]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET} $*"; }
die()     { echo -e "${RED}[ERR]${RESET}  $*" >&2; exit 1; }
confirm() { read -rp "$1 [y/N]: " _c </dev/tty; [[ "$_c" =~ ^[Yy]$ ]]; }
require() { command -v "$1" &>/dev/null || die "Требуется '$1', но не установлен"; }

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) die "Неподдерживаемая архитектура: $(uname -m)" ;;
  esac
}

fetch_latest() {
  local filter="$1"
  require curl
  local json
  json=$(curl -fsSL "$GITHUB_API")
  LATEST_TAG=$(echo "$json" | grep -o '"tag_name": *"[^"]*"' | head -1 | grep -o '"[^"]*"$' | tr -d '"')
  ASSET_URL=$(echo "$json" | grep -o "\"browser_download_url\": *\"[^\"]*${filter}[^\"]*\"" | head -1 | grep -o 'https://[^"]*')
  [ -n "$LATEST_TAG" ] || die "Не удалось получить версию из GitHub"
  [ -n "$ASSET_URL"  ] || die "Артефакт '${filter}' не найден в релизе ${LATEST_TAG}"
}

need_update() {
  local vfile="$1"
  local current=""
  [ -f "$vfile" ] && current=$(cat "$vfile")
  if [ -n "$current" ] && [ "$current" = "$LATEST_TAG" ]; then
    warn "Уже установлена актуальная версия ${BOLD}${current}${RESET}"
    confirm "  Переустановить?" || { info "Отменено."; exit 0; }
  elif [ -n "$current" ]; then
    info "Обновление: ${BOLD}${current}${RESET} -> ${BOLD}${LATEST_TAG}${RESET}"
  else
    info "Первая установка ${BOLD}${LATEST_TAG}${RESET}"
  fi
}

# ──────────────────────────────────────────────────────────────────────────────
install_server() {
  require curl; require tar; require systemctl
  [ "$(detect_arch)" = "amd64" ] || die "Сервер поддерживается только на amd64"

  fetch_latest "mon-server-linux-amd64.tar.gz"
  need_update "$SERVER_VERSION"

  local tmp
  tmp=$(mktemp -d)
  trap "rm -rf $tmp" EXIT

  info "Загрузка сервера..."
  curl -fsSL --progress-bar -o "$tmp/server.tar.gz" "$ASSET_URL"
  tar -xzf "$tmp/server.tar.gz" -C "$tmp"

  systemctl is-active --quiet mon-server 2>/dev/null && systemctl stop mon-server && info "Служба остановлена"

  install -Dm755 "$tmp/mon-server" "$SERVER_BIN"
  ok "Бинарник: $SERVER_BIN"

  mkdir -p "$SERVER_WEB"
  rm -rf "${SERVER_WEB:?}"/*
  cp -r "$tmp/web-dist/." "$SERVER_WEB/"
  ok "Фронтенд: $SERVER_WEB"

  mkdir -p "$SERVER_DATA"

  cat > /etc/systemd/system/mon-server.service << SERVICE
[Unit]
Description=LinkMus Monitor Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$SERVER_DATA
ExecStart=$SERVER_BIN
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
SERVICE

  systemctl daemon-reload
  systemctl enable mon-server
  systemctl restart mon-server
  ok "Служба mon-server запущена"

  if command -v nginx &>/dev/null; then
    if [ ! -f /etc/nginx/sites-available/linkmus-monitor ]; then
      cat > /etc/nginx/sites-available/linkmus-monitor << NGINX
server {
    listen 80;
    server_name _;
    root $SERVER_WEB;
    index index.html;
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header Host \$host;
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
NGINX
      ln -sf /etc/nginx/sites-available/linkmus-monitor /etc/nginx/sites-enabled/
      rm -f /etc/nginx/sites-enabled/default
      nginx -t && systemctl reload nginx
      ok "Nginx настроен"
    else
      nginx -t && systemctl reload nginx
      ok "Nginx перезагружен"
    fi
  else
    warn "Nginx не найден — интерфейс доступен напрямую на :8080"
  fi

  echo "$LATEST_TAG" > "$SERVER_VERSION"
  install_mon_cli
  echo ""
  ok "Сервер ${BOLD}${LATEST_TAG}${RESET} установлен!"
  echo -e "  Адрес: ${BOLD}http://$(hostname -I | awk '{print $1}')${RESET}"
}

# ──────────────────────────────────────────────────────────────────────────────
install_agent() {
  require curl; require systemctl

  local arch
  arch=$(detect_arch)
  local asset="mon-agent-linux"
  [ "$arch" = "arm64" ] && asset="mon-agent-linux-arm64"

  fetch_latest "$asset"
  need_update "$AGENT_VERSION"

  local tmp
  tmp=$(mktemp -d)
  trap "rm -rf $tmp" EXIT

  info "Загрузка агента..."
  curl -fsSL --progress-bar -o "$tmp/mon-agent" "$ASSET_URL"

  systemctl is-active --quiet mon-agent 2>/dev/null && systemctl stop mon-agent && info "Служба остановлена"

  mkdir -p "$AGENT_DIR"
  install -Dm755 "$tmp/mon-agent" "$AGENT_BIN"
  ok "Бинарник: $AGENT_BIN"

  if [ ! -f "$AGENT_CFG" ]; then
    echo ""
    echo -e "${BOLD}Настройка агента:${RESET}"
    read -rp "  URL сервера [http://10.10.10.10:8080]: " srv </dev/tty
    srv="${srv:-http://10.10.10.10:8080}"
    read -rp "  Интервал в секундах [5]: " ivl </dev/tty
    ivl="${ivl:-5}"
    ivl="${ivl%s}s"
    printf 'server:\n  url: "%s/api/metrics"\n  interval: %s\n' "$srv" "$ivl" > "$AGENT_CFG"
    ok "Конфиг создан: $AGENT_CFG"
  else
    ok "Текущий конфиг:"
    cat "$AGENT_CFG"
    echo ""
    if confirm "  Изменить настройки?"; then
      read -rp "  URL сервера: " srv </dev/tty
      read -rp "  Интервал в секундах: " ivl </dev/tty
      ivl="${ivl%s}s"
      printf 'server:\n  url: "%s/api/metrics"\n  interval: %s\n' "$srv" "$ivl" > "$AGENT_CFG"
      ok "Конфиг обновлён"
    fi
  fi

  cat > /etc/systemd/system/mon-agent.service << SERVICE
[Unit]
Description=LinkMus Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$AGENT_DIR
ExecStart=$AGENT_BIN
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
SERVICE

  systemctl daemon-reload
  systemctl enable mon-agent
  systemctl restart mon-agent
  ok "Служба mon-agent запущена"

  echo "$LATEST_TAG" > "$AGENT_VERSION"
  install_mon_cli
  echo ""
  ok "Агент ${BOLD}${LATEST_TAG}${RESET} установлен!"
}

# ──────────────────────────────────────────────────────────────────────────────
install_mon_cli() {
  cat > /usr/local/bin/mon << 'MONEOF'
#!/usr/bin/env bash
GREEN='\033[0;32m'; RED='\033[0;31m'; CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'
usage() {
  echo -e "${BOLD}${CYAN}LinkMus Monitor${RESET}"
  echo -e "Использование: mon <server|agent> <start|stop|restart|status|enable|disable|logs>"
  exit 1
}
[ $# -lt 2 ] && usage
case "$1" in
  server) SVC="mon-server" ;;
  agent)  SVC="mon-agent" ;;
  help|--help|-h) usage ;;
  *) echo -e "${RED}Неизвестная цель:${RESET} $1"; usage ;;
esac
case "$2" in
  start)   systemctl start   "$SVC" && echo -e "${GREEN}[OK]${RESET} $SVC запущен" ;;
  stop)    systemctl stop    "$SVC" && echo -e "${GREEN}[OK]${RESET} $SVC остановлен" ;;
  restart) systemctl restart "$SVC" && echo -e "${GREEN}[OK]${RESET} $SVC перезапущен" ;;
  status)  systemctl status  "$SVC" --no-pager ;;
  enable)  systemctl enable  "$SVC" && echo -e "${GREEN}[OK]${RESET} Автозапуск включён" ;;
  disable) systemctl disable "$SVC" && echo -e "${GREEN}[OK]${RESET} Автозапуск выключен" ;;
  logs)    journalctl -fu    "$SVC" ;;
  *) echo -e "${RED}Неизвестная команда:${RESET} $2"; usage ;;
esac
MONEOF
  chmod +x /usr/local/bin/mon
  ok "CLI: mon server|agent start|stop|restart|status|enable|disable|logs"
}

# ──────────────────────────────────────────────────────────────────────────────
uninstall_server() {
  echo ""
  warn "Будет удалено ВСЁ: бинарник, фронтенд, база данных, логи, конфиг nginx."
  confirm "  Продолжить?" || { info "Отменено."; exit 0; }

  systemctl stop    mon-server 2>/dev/null || true
  systemctl disable mon-server 2>/dev/null || true
  rm -f /etc/systemd/system/mon-server.service
  systemctl daemon-reload

  rm -f "$SERVER_BIN"
  rm -rf "$SERVER_DIR"

  rm -f /etc/nginx/sites-enabled/linkmus-monitor
  rm -f /etc/nginx/sites-available/linkmus-monitor
  command -v nginx &>/dev/null && nginx -t && systemctl reload nginx 2>/dev/null || true

  [ ! -f "$AGENT_BIN" ] && rm -f /usr/local/bin/mon

  ok "Сервер полностью удалён."
}

# ──────────────────────────────────────────────────────────────────────────────
uninstall_agent() {
  echo ""
  warn "Будет удалено ВСЁ: бинарник, конфиг, логи агента."
  confirm "  Продолжить?" || { info "Отменено."; exit 0; }

  systemctl stop    mon-agent 2>/dev/null || true
  systemctl disable mon-agent 2>/dev/null || true
  rm -f /etc/systemd/system/mon-agent.service
  systemctl daemon-reload

  rm -rf "$AGENT_DIR"

  [ ! -f "$SERVER_BIN" ] && rm -f /usr/local/bin/mon

  ok "Агент полностью удалён."
}

# ──────────────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}${CYAN}║      LinkMus Monitor — Установщик       ║${RESET}"
echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════╝${RESET}"
echo ""

[ "$(id -u)" -eq 0 ] || die "Запустите от root: sudo bash install.sh"

echo -e "  ${BOLD}1${RESET}) Установить / обновить сервер"
echo -e "  ${BOLD}2${RESET}) Установить / обновить агент Linux"
echo -e "  ${BOLD}3${RESET}) Удалить сервер (полная очистка)"
echo -e "  ${BOLD}4${RESET}) Удалить агент Linux (полная очистка)"
echo ""
read -rp "Выбор [1-4]: " choice </dev/tty

case "$choice" in
  1) install_server ;;
  2) install_agent ;;
  3) uninstall_server ;;
  4) uninstall_agent ;;
  *) die "Неверный выбор: $choice" ;;
esac
