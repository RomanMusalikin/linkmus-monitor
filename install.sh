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

# Маркеры нашего блока в Caddyfile — удаляем ТОЛЬКО между ними
CADDY_MARKER_BEGIN="# BEGIN linkmus-monitor"
CADDY_MARKER_END="# END linkmus-monitor"

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

# Валидация порта: только цифры, диапазон 1-65535
validate_port() {
  local p="$1"
  [[ "$p" =~ ^[0-9]+$ ]] || die "Порт должен быть числом: $p"
  [ "$p" -ge 1 ] && [ "$p" -le 65535 ] || die "Порт вне диапазона 1-65535: $p"
}

# Валидация домена: нет пробелов, спецсимволов shell, минимальная длина
validate_domain() {
  local d="$1"
  [[ "$d" =~ ^[a-zA-Z0-9._-]+$ ]] || die "Недопустимые символы в домене: $d"
  [ "${#d}" -ge 3 ] || die "Домен слишком короткий: $d"
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
  # Проверяем что URL ведёт на GitHub
  [[ "$ASSET_URL" == https://github.com/* ]] || die "Неожиданный URL артефакта: $ASSET_URL"
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

  echo ""
  read -rp "  Порт сервера [8080]: " srv_port </dev/tty
  srv_port="${srv_port:-8080}"
  validate_port "$srv_port"

  local tmp
  tmp=$(mktemp -d)

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

  rm -rf "$tmp"

  mkdir -p "$SERVER_DATA"

  cat > /etc/systemd/system/mon-server.service << SERVICE
[Unit]
Description=LinkMus Monitor Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$SERVER_DATA
Environment=PORT=$srv_port
Environment=DB_PATH=$SERVER_DATA/monitor.db
Environment=WEB_PATH=$SERVER_WEB
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
  ok "Служба mon-server запущена на порту $srv_port"

  echo ""
  echo -e "  Настроить обратный прокси?"
  echo -e "  ${BOLD}1${RESET}) Nginx"
  echo -e "  ${BOLD}2${RESET}) Caddy"
  echo -e "  ${BOLD}3${RESET}) Пропустить"
  read -rp "  Выбор [1-3]: " proxy_choice </dev/tty

  case "$proxy_choice" in
    1) setup_nginx "$srv_port" ;;
    2) setup_caddy "$srv_port" ;;
    *) warn "Прокси не настроен — сервер доступен напрямую на :${srv_port}" ;;
  esac

  echo "$LATEST_TAG" > "$SERVER_VERSION"
  install_mon_cli
  echo ""
  ok "Сервер ${BOLD}${LATEST_TAG}${RESET} установлен!"
  echo -e "  Адрес: ${BOLD}http://$(hostname -I | awk '{print $1}')${RESET}"
}

setup_nginx() {
  local port="$1"
  if ! command -v nginx &>/dev/null; then
    warn "Nginx не найден — установите его вручную"
    return
  fi
  if [ ! -f /etc/nginx/sites-available/linkmus-monitor ]; then
    cat > /etc/nginx/sites-available/linkmus-monitor << NGINX
server {
    listen 80;
    server_name _;
    root $SERVER_WEB;
    index index.html;
    location /api/ {
        proxy_pass http://127.0.0.1:${port};
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header Host \$host;
    }
    location / { try_files \$uri \$uri/ /index.html; }
}
NGINX
    ln -sf /etc/nginx/sites-available/linkmus-monitor /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
  else
    sed -i "s|proxy_pass http://127.0.0.1:[0-9]*|proxy_pass http://127.0.0.1:${port}|g" \
      /etc/nginx/sites-available/linkmus-monitor
  fi
  nginx -t && systemctl reload nginx
  ok "Nginx настроен"
}

setup_caddy() {
  local port="$1"

  local caddy_mode=""
  local caddyfile=""
  local caddy_container=""
  local proxy_host=""

  if command -v caddy &>/dev/null; then
    caddy_mode="local"
    caddyfile="/etc/caddy/Caddyfile"
    proxy_host="localhost"
  elif command -v docker &>/dev/null; then
    caddy_container=$(docker ps --format '{{.Names}}' | grep -i caddy | head -1)
    if [ -n "$caddy_container" ]; then
      caddy_mode="docker"
      caddyfile=$(docker inspect "$caddy_container" \
        --format '{{range .Mounts}}{{if eq .Destination "/etc/caddy/Caddyfile"}}{{.Source}}{{end}}{{end}}')
      proxy_host=$(hostname -I | awk '{print $1}')
    fi
  fi

  if [ -z "$caddy_mode" ]; then
    warn "Caddy не найден (ни локально, ни в Docker) — настройте прокси вручную"
    return
  fi

  if [ -z "$caddyfile" ] || [ ! -f "$caddyfile" ]; then
    warn "Caddyfile не найден — настройте прокси вручную"
    return
  fi

  info "Caddy обнаружен (${caddy_mode}), Caddyfile: ${caddyfile}"

  read -rp "  Домен (например monitor.example.com): " caddy_domain </dev/tty
  [ -n "$caddy_domain" ] || { warn "Домен не указан — Caddy не настроен"; return; }
  validate_domain "$caddy_domain"

  # Удаляем только наш маркированный блок — передаём путь через аргумент, не интерполяцию
  python3 - "$caddyfile" << 'PYEOF'
import sys

path = sys.argv[1]
begin = "# BEGIN linkmus-monitor"
end   = "# END linkmus-monitor"

text = open(path).read()
lines = text.split('\n')
out = []
skip = False
for line in lines:
    if line.strip() == begin:
        skip = True
        continue
    if line.strip() == end:
        skip = False
        continue
    if not skip:
        out.append(line)

open(path, 'w').write('\n'.join(out).rstrip() + '\n')
PYEOF

  # Вставляем новый блок с маркерами перед финальным catch-all (:443/:80) если есть
  # Домен и proxy_host передаём через аргументы — никакой shell-интерполяции в Python-коде
  python3 - "$caddyfile" "$caddy_domain" "${proxy_host}:${port}" << 'PYEOF'
import sys, re

path, domain, upstream = sys.argv[1], sys.argv[2], sys.argv[3]
text = open(path).read()

block = (
    "\n# BEGIN linkmus-monitor\n"
    "https://" + domain + " {\n"
    "    reverse_proxy " + upstream + "\n"
    "}\n"
    "# END linkmus-monitor\n"
)

m = re.search(r'\n:[0-9]+ \{', text)
if m:
    text = text[:m.start()] + block + text[m.start():]
else:
    text = text.rstrip() + '\n' + block

open(path, 'w').write(text)
PYEOF

  # При Caddy в Docker — разрешаем Docker-сетям доступ к конкретному порту на хосте
  if [ "$caddy_mode" = "docker" ] && command -v ufw &>/dev/null; then
    ufw allow from 172.16.0.0/12 to any port "$port" comment "linkmus-monitor caddy-docker" >/dev/null 2>&1 || true
    ok "UFW: разрешён доступ из Docker-сети на порт ${port}"
  fi

  # Перезагружаем Caddy
  if [ "$caddy_mode" = "docker" ]; then
    docker exec "$caddy_container" caddy reload --config /etc/caddy/Caddyfile
  else
    caddy fmt --overwrite "$caddyfile" 2>/dev/null || true
    caddy reload --config "$caddyfile" 2>/dev/null || systemctl reload caddy
  fi

  ok "Caddy настроен: https://${caddy_domain}"
  echo -e "  Убедитесь что DNS A-запись ${BOLD}${caddy_domain}${RESET} указывает на ${BOLD}${proxy_host}${RESET}"
}

# Удаляет ТОЛЬКО наш маркированный блок из Caddyfile — чужие блоки не трогает
caddy_cleanup_block() {
  local caddyfile=""
  local caddy_container=""
  local caddy_mode=""
  local srv_port=""

  # Читаем порт из systemd-юнита чтобы точно удалить нужное UFW-правило
  if [ -f /etc/systemd/system/mon-server.service ]; then
    srv_port=$(grep -o 'PORT=[0-9]*' /etc/systemd/system/mon-server.service | grep -o '[0-9]*' | head -1)
  fi

  if command -v caddy &>/dev/null; then
    caddy_mode="local"
    caddyfile="/etc/caddy/Caddyfile"
  elif command -v docker &>/dev/null; then
    caddy_container=$(docker ps --format '{{.Names}}' | grep -i caddy | head -1)
    if [ -n "$caddy_container" ]; then
      caddy_mode="docker"
      caddyfile=$(docker inspect "$caddy_container" \
        --format '{{range .Mounts}}{{if eq .Destination "/etc/caddy/Caddyfile"}}{{.Source}}{{end}}{{end}}')
    fi
  fi

  [ -f "$caddyfile" ] && command -v python3 &>/dev/null || return

  # Удаляем только наш маркированный блок
  python3 - "$caddyfile" << 'PYEOF'
import sys

path  = sys.argv[1]
begin = "# BEGIN linkmus-monitor"
end   = "# END linkmus-monitor"

text  = open(path).read()
lines = text.split('\n')
out   = []
skip  = False
for line in lines:
    if line.strip() == begin:
        skip = True
        continue
    if line.strip() == end:
        skip = False
        continue
    if not skip:
        out.append(line)

open(path, 'w').write('\n'.join(out).rstrip() + '\n')
PYEOF

  # Удаляем UFW-правило только для конкретного порта нашего сервера
  if command -v ufw &>/dev/null && [ -n "$srv_port" ]; then
    ufw delete allow from 172.16.0.0/12 to any port "$srv_port" >/dev/null 2>&1 || true
  fi

  if [ "$caddy_mode" = "docker" ] && [ -n "$caddy_container" ]; then
    docker exec "$caddy_container" caddy reload --config /etc/caddy/Caddyfile 2>/dev/null || true
  elif [ "$caddy_mode" = "local" ]; then
    caddy reload --config "$caddyfile" 2>/dev/null || systemctl reload caddy 2>/dev/null || true
  fi
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

  info "Загрузка агента..."
  curl -fsSL --progress-bar -o "$tmp/mon-agent" "$ASSET_URL"

  systemctl is-active --quiet mon-agent 2>/dev/null && systemctl stop mon-agent && info "Служба остановлена"

  mkdir -p "$AGENT_DIR"
  install -Dm755 "$tmp/mon-agent" "$AGENT_BIN"
  ok "Бинарник: $AGENT_BIN"

  rm -rf "$tmp"

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
GREEN='\033[0;32m'; RED='\033[0;31m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'; BOLD='\033[1m'; RESET='\033[0m'

REPO="RomanMusalikin/linkmus-monitor"
SERVER_DIR="/opt/linkmus-monitor"
SERVER_BIN="/usr/local/bin/mon-server"
SERVER_WEB="$SERVER_DIR/web"
SERVER_VERSION="$SERVER_DIR/.version"
AGENT_DIR="/opt/mon-agent"
AGENT_BIN="$AGENT_DIR/mon-agent"
AGENT_VERSION="$AGENT_DIR/.version"

CADDY_MARKER_BEGIN="# BEGIN linkmus-monitor"
CADDY_MARKER_END="# END linkmus-monitor"

usage() {
  echo -e "${BOLD}${CYAN}LinkMus Monitor${RESET}"
  echo -e "Использование: ${BOLD}mon <server|agent> <команда>${RESET}"
  echo ""
  echo -e "  ${BOLD}start${RESET}    Запустить службу"
  echo -e "  ${BOLD}stop${RESET}     Остановить службу"
  echo -e "  ${BOLD}restart${RESET}  Перезапустить службу"
  echo -e "  ${BOLD}status${RESET}   Показать статус службы"
  echo -e "  ${BOLD}enable${RESET}   Включить автозапуск"
  echo -e "  ${BOLD}disable${RESET}  Выключить автозапуск"
  echo -e "  ${BOLD}logs${RESET}     Следить за логами (journalctl -f)"
  echo -e "  ${BOLD}update${RESET}   Проверить обновления и установить при наличии"
  echo -e "  ${BOLD}delete${RESET}   Полностью деинсталировать (служба, файлы, конфиг)"
  echo ""
  exit 1
}

caddy_remove_block() {
  local caddyfile="" caddy_container="" caddy_mode="" srv_port=""

  [ -f /etc/systemd/system/mon-server.service ] && \
    srv_port=$(grep -o 'PORT=[0-9]*' /etc/systemd/system/mon-server.service | grep -o '[0-9]*' | head -1)

  if command -v caddy &>/dev/null; then
    caddy_mode="local"; caddyfile="/etc/caddy/Caddyfile"
  elif command -v docker &>/dev/null; then
    caddy_container=$(docker ps --format '{{.Names}}' | grep -i caddy | head -1)
    [ -n "$caddy_container" ] && {
      caddy_mode="docker"
      caddyfile=$(docker inspect "$caddy_container" \
        --format '{{range .Mounts}}{{if eq .Destination "/etc/caddy/Caddyfile"}}{{.Source}}{{end}}{{end}}')
    }
  fi

  [ -f "$caddyfile" ] && command -v python3 &>/dev/null || return

  python3 - "$caddyfile" << 'PYEOF'
import sys
path  = sys.argv[1]
begin = "# BEGIN linkmus-monitor"
end   = "# END linkmus-monitor"
lines = open(path).read().split('\n')
out, skip = [], False
for line in lines:
    if line.strip() == begin: skip = True; continue
    if line.strip() == end:   skip = False; continue
    if not skip: out.append(line)
open(path, 'w').write('\n'.join(out).rstrip() + '\n')
PYEOF

  if command -v ufw &>/dev/null && [ -n "$srv_port" ]; then
    ufw delete allow from 172.16.0.0/12 to any port "$srv_port" >/dev/null 2>&1 || true
  fi

  if [ "$caddy_mode" = "docker" ] && [ -n "$caddy_container" ]; then
    docker exec "$caddy_container" caddy reload --config /etc/caddy/Caddyfile 2>/dev/null || true
  elif [ "$caddy_mode" = "local" ]; then
    caddy reload --config "$caddyfile" 2>/dev/null || systemctl reload caddy 2>/dev/null || true
  fi
}

do_update() {
  local target="$1"
  [ "$(id -u)" -eq 0 ] || { echo -e "${RED}[ERR]${RESET}  Нужны права root: sudo mon $target update"; exit 1; }

  local json latest_tag
  json=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) \
    || { echo -e "${RED}[ERR]${RESET}  Нет доступа к GitHub"; exit 1; }
  latest_tag=$(echo "$json" | grep -o '"tag_name": *"[^"]*"' | head -1 | grep -o '"[^"]*"$' | tr -d '"')
  [ -n "$latest_tag" ] || { echo -e "${RED}[ERR]${RESET}  Не удалось получить версию"; exit 1; }

  if [ "$target" = "server" ]; then
    local current=""
    [ -f "$SERVER_VERSION" ] && current=$(cat "$SERVER_VERSION")
    if [ "$current" = "$latest_tag" ]; then
      echo -e "${YELLOW}[WARN]${RESET} Уже установлена актуальная версия ${BOLD}$current${RESET}"
      read -rp "  Переустановить? [y/N]: " ans </dev/tty
      [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    else
      echo -e "  Доступно обновление: ${BOLD}${current:-не установлен}${RESET} -> ${BOLD}${latest_tag}${RESET}"
      read -rp "  Обновить? [y/N]: " ans </dev/tty
      [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    fi
    local asset_url
    asset_url=$(echo "$json" | grep -o '"browser_download_url": *"[^"]*mon-server-linux-amd64\.tar\.gz[^"]*"' | grep -o 'https://[^"]*')
    [ -n "$asset_url" ] || { echo -e "${RED}[ERR]${RESET}  Артефакт сервера не найден в релизе $latest_tag"; exit 1; }
    local tmp; tmp=$(mktemp -d)
    echo -e "${CYAN}[INFO]${RESET} Загрузка $latest_tag..."
    curl -fsSL --progress-bar -o "$tmp/server.tar.gz" "$asset_url"
    tar -xzf "$tmp/server.tar.gz" -C "$tmp"
    systemctl is-active --quiet mon-server 2>/dev/null && systemctl stop mon-server
    install -Dm755 "$tmp/mon-server" "$SERVER_BIN"
    mkdir -p "$SERVER_WEB"; rm -rf "${SERVER_WEB:?}"/*
    cp -r "$tmp/web-dist/." "$SERVER_WEB/"
    echo "$latest_tag" > "$SERVER_VERSION"
    rm -rf "$tmp"
    systemctl start mon-server
    echo -e "${GREEN}[ OK ]${RESET} Сервер обновлён до ${BOLD}$latest_tag${RESET}"
    curl -fsSL "https://raw.githubusercontent.com/$REPO/main/install.sh" | bash -s -- --update-cli 2>/dev/null && \
      echo -e "${GREEN}[ OK ]${RESET} CLI mon обновлён" || true

  else
    local current=""
    [ -f "$AGENT_VERSION" ] && current=$(cat "$AGENT_VERSION")
    if [ "$current" = "$latest_tag" ]; then
      echo -e "${YELLOW}[WARN]${RESET} Уже установлена актуальная версия ${BOLD}$current${RESET}"
      read -rp "  Переустановить? [y/N]: " ans </dev/tty
      [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    else
      echo -e "  Доступно обновление: ${BOLD}${current:-не установлен}${RESET} -> ${BOLD}${latest_tag}${RESET}"
      read -rp "  Обновить? [y/N]: " ans </dev/tty
      [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    fi
    local arch asset
    arch=$(uname -m)
    case "$arch" in
      x86_64|amd64)  asset="mon-agent-linux" ;;
      aarch64|arm64) asset="mon-agent-linux-arm64" ;;
      *) echo -e "${RED}[ERR]${RESET}  Неподдерживаемая архитектура: $arch"; exit 1 ;;
    esac
    local asset_url
    asset_url=$(echo "$json" | grep -o "\"browser_download_url\": *\"[^\"]*${asset}[^\"]*\"" | head -1 | grep -o 'https://[^"]*')
    [ -n "$asset_url" ] || { echo -e "${RED}[ERR]${RESET}  Артефакт агента не найден в релизе $latest_tag"; exit 1; }
    local tmp; tmp=$(mktemp -d)
    echo -e "${CYAN}[INFO]${RESET} Загрузка $latest_tag..."
    curl -fsSL --progress-bar -o "$tmp/mon-agent" "$asset_url"
    systemctl is-active --quiet mon-agent 2>/dev/null && systemctl stop mon-agent
    install -Dm755 "$tmp/mon-agent" "$AGENT_BIN"
    echo "$latest_tag" > "$AGENT_VERSION"
    rm -rf "$tmp"
    systemctl start mon-agent
    echo -e "${GREEN}[ OK ]${RESET} Агент обновлён до ${BOLD}$latest_tag${RESET}"
    curl -fsSL "https://raw.githubusercontent.com/$REPO/main/install.sh" | bash -s -- --update-cli 2>/dev/null && \
      echo -e "${GREEN}[ OK ]${RESET} CLI mon обновлён" || true
  fi
}

do_delete() {
  local target="$1"
  [ "$(id -u)" -eq 0 ] || { echo -e "${RED}[ERR]${RESET}  Нужны права root: sudo mon $target delete"; exit 1; }

  if [ "$target" = "server" ]; then
    echo -e "${YELLOW}[WARN]${RESET} Будет удалено ВСЁ: бинарник, фронтенд, база данных, служба."
    read -rp "  Подтвердить удаление? [y/N]: " ans </dev/tty
    [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    systemctl stop    mon-server 2>/dev/null || true
    systemctl disable mon-server 2>/dev/null || true
    rm -f /etc/systemd/system/mon-server.service
    systemctl daemon-reload
    rm -f "$SERVER_BIN"
    rm -rf "$SERVER_DIR"
    rm -f /etc/nginx/sites-enabled/linkmus-monitor
    rm -f /etc/nginx/sites-available/linkmus-monitor
    command -v nginx &>/dev/null && nginx -t 2>/dev/null && systemctl reload nginx 2>/dev/null || true
    caddy_remove_block
    [ ! -f "$AGENT_BIN" ] && rm -f /usr/local/bin/mon /usr/bin/mon
    echo -e "${GREEN}[ OK ]${RESET} Сервер полностью удалён."

  else
    echo -e "${YELLOW}[WARN]${RESET} Будет удалено ВСЁ: бинарник, конфиг, директория агента."
    read -rp "  Подтвердить удаление? [y/N]: " ans </dev/tty
    [[ "$ans" =~ ^[Yy]$ ]] || { echo "Отменено."; exit 0; }
    systemctl stop    mon-agent 2>/dev/null || true
    systemctl disable mon-agent 2>/dev/null || true
    rm -f /etc/systemd/system/mon-agent.service
    systemctl daemon-reload
    rm -rf "$AGENT_DIR"
    [ ! -f "$SERVER_BIN" ] && rm -f /usr/local/bin/mon /usr/bin/mon
    echo -e "${GREEN}[ OK ]${RESET} Агент полностью удалён."
  fi
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
  update)  do_update "$1" ;;
  delete)  do_delete "$1" ;;
  *) echo -e "${RED}Неизвестная команда:${RESET} $2"; usage ;;
esac
MONEOF
  chmod +x /usr/local/bin/mon
  ln -sf /usr/local/bin/mon /usr/bin/mon
  ok "CLI: mon server|agent start|stop|restart|status|enable|disable|logs|update|delete"
}

# ──────────────────────────────────────────────────────────────────────────────
uninstall_server() {
  echo ""
  warn "Будет удалено ВСЁ: бинарник, фронтенд, база данных, служба."
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

  caddy_cleanup_block

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
# Режим --update-cli: только обновить /usr/local/bin/mon (вызывается из do_update)
if [ "${1:-}" = "--update-cli" ]; then
  install_mon_cli
  exit 0
fi

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
