#!/usr/bin/env bash
# 本地发布包一键部署。无需 Go / Node.js；重复运行保留规则、配置和登录凭据。
# bash deploy.sh [--dir /opt/gost-panel] [--host 0.0.0.0] [--port 19090]
set -euo pipefail
umask 077

info() { printf '[INFO] %s\n' "$*"; }
die() { printf '[ERROR] %s\n' "$*" >&2; exit 1; }
PKG_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR=/etc/gost-panel
INSTALL_DIR=''; PANEL_HOST=''; PANEL_PORT=''; PANEL_USER=''
PANEL_PASS="${PANEL_PASS:-}"; GOST_API_PASS="${GOST_API_PASS:-}"
GOST_VERSION=v3.2.6
GOST_API_PORT=18080

while (($#)); do
  case "$1" in
    --dir|--install-dir|--host|--panel-host|--port|--panel-port|--user|--panel-pass|--gost-api-pass)
      (($# >= 2)) || die "$1 缺少参数值"
      case "$1" in
        --dir|--install-dir) INSTALL_DIR="$2" ;;
        --host|--panel-host) PANEL_HOST="$2" ;;
        --port|--panel-port) PANEL_PORT="$2" ;;
        --user) PANEL_USER="$2" ;;
        --panel-pass) PANEL_PASS="$2" ;;
        --gost-api-pass) GOST_API_PASS="$2" ;;
      esac
      shift 2 ;;
    -h|--help) sed -n '2,3p' "$0"; exit 0 ;;
    *) die "未知参数: $1" ;;
  esac
done

[[ $EUID == 0 ]] || die '请使用 root 运行'
command -v systemctl >/dev/null || die '需要运行 systemd 的 Linux 系统'
[[ -d /run/systemd/system ]] || die 'systemd 未运行'
[[ -f "$PKG_DIR/gost-panel" && -f "$PKG_DIR/static/index.html" ]] || die '发布包缺少二进制或前端产物'

missing=()
for cmd in curl tar python3; do command -v "$cmd" >/dev/null || missing+=("$cmd"); done
if ((${#missing[@]})); then
  if command -v apt-get >/dev/null; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq curl ca-certificates tar python3
  elif command -v dnf >/dev/null; then
    dnf install -y curl ca-certificates tar python3
  else
    die "缺少依赖: ${missing[*]}，且未找到 apt-get/dnf"
  fi
fi

# 只读取本安装器管理的状态；避免重装时重新生成密码。
if [[ -f "$STATE_DIR/install.json" ]]; then
  eval "$(python3 - "$STATE_DIR/install.json" <<'PY'
import json, shlex, sys
state = json.load(open(sys.argv[1]))
for key in ('INSTALL_DIR', 'PANEL_HOST', 'PANEL_PORT', 'PANEL_USER', 'PANEL_PASS', 'GOST_API_PASS'):
    print('SAVED_' + key + '=' + shlex.quote(str(state[key])))
PY
  )"
fi
INSTALL_DIR="${INSTALL_DIR:-${SAVED_INSTALL_DIR:-/opt/gost-panel}}"
PANEL_HOST="${PANEL_HOST:-${SAVED_PANEL_HOST:-127.0.0.1}}"
PANEL_PORT="${PANEL_PORT:-${SAVED_PANEL_PORT:-19090}}"
PANEL_USER="${PANEL_USER:-${SAVED_PANEL_USER:-admin}}"
PANEL_PASS="${PANEL_PASS:-${SAVED_PANEL_PASS:-}}"
GOST_API_PASS="${GOST_API_PASS:-${SAVED_GOST_API_PASS:-}}"
[[ "$INSTALL_DIR" =~ ^/[a-zA-Z0-9_/-]+$ && "$INSTALL_DIR" != / && "$INSTALL_DIR" != */../* ]] || die '安装目录必须为绝对路径且不能含特殊字符'
case "$INSTALL_DIR" in /opt|/etc|/usr|/var|/root|/home|/tmp) die '请指定独立的安装子目录' ;; esac
[[ -z "${SAVED_INSTALL_DIR:-}" || "$SAVED_INSTALL_DIR" == "$INSTALL_DIR" ]] || die '已安装在其他目录，请先迁移服务再更改安装目录'
[[ "$PANEL_HOST" =~ ^[a-zA-Z0-9.:-]+$ ]] || die '监听地址格式不正确'
[[ "$PANEL_PORT" =~ ^[0-9]{1,5}$ ]] && ((10#$PANEL_PORT > 0 && 10#$PANEL_PORT < 65536)) || die '端口范围为 1-65535'
[[ "$PANEL_PORT" != "$GOST_API_PORT" ]] || die '面板端口与 gost API 端口冲突'
for value in "$PANEL_USER" "$PANEL_PASS" "$GOST_API_PASS"; do
  [[ "$value" != *$'\n'* && "$value" != *$'\r'* ]] || die '用户名和密码不能包含换行'
done
[[ -n "$PANEL_PASS" ]] || PANEL_PASS="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
[[ -n "$GOST_API_PASS" ]] || GOST_API_PASS="$(python3 -c 'import secrets; print(secrets.token_urlsafe(24))')"
if [[ -n "${SAVED_PANEL_PASS:-}" && "$PANEL_PASS" != "$SAVED_PANEL_PASS" ]]; then
  die '重装不更改已有登录密码，请使用面板中的修改密码功能'
fi
if [[ -n "${SAVED_GOST_API_PASS:-}" && "$GOST_API_PASS" != "$SAVED_GOST_API_PASS" ]]; then
  die '重装不更改现有 gost API 密码'
fi
[[ ! -f /opt/gost/config.json || -f "$STATE_DIR/install.json" ]] || die '/opt/gost/config.json 已存在且非本安装器管理，已停止以保留原配置'

case "$(uname -m)" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) die '不支持的 CPU 架构' ;; esac
# 先运行 version 检查架构与可执行位，失败时尚未修改现有服务。
chmod 755 "$PKG_DIR/gost-panel"
version="$("$PKG_DIR/gost-panel" version)" || die '后端二进制与本机架构不兼容'
info "部署 $version / linux/$arch"
work_dir="$(mktemp -d /tmp/gost-panel-install.XXXXXX)"
trap 'rm -rf -- "$work_dir"' EXIT

if [[ ! -x /usr/local/bin/gost ]]; then
  info "下载 gost $GOST_VERSION"
  asset="gost_${GOST_VERSION#v}_linux_${arch}.tar.gz"
  base="https://github.com/go-gost/gost/releases/download/$GOST_VERSION"
  curl -fLsS --retry 3 --connect-timeout 20 --max-time 300 "$base/$asset" -o "$work_dir/$asset"
  curl -fLsS --retry 3 --connect-timeout 20 --max-time 60 "$base/checksums.txt" -o "$work_dir/checksums.txt"
  (cd "$work_dir" && awk -v name="$asset" '$2 == name { print; found=1 } END { if (!found) exit 1 }' checksums.txt | sha256sum -c -)
  tar -xzf "$work_dir/$asset" -C "$work_dir" gost
  install -m 755 "$work_dir/gost" /usr/local/bin/gost
fi
info "$(/usr/local/bin/gost -V)"

export INSTALL_DIR PANEL_HOST PANEL_PORT PANEL_USER PANEL_PASS GOST_API_PASS GOST_API_PORT STATE_DIR
mkdir -p "$INSTALL_DIR" "$STATE_DIR" /opt/gost
chmod 700 "$STATE_DIR" /opt/gost
backup_dir="$INSTALL_DIR/backups/$(date -u +%Y%m%dT%H%M%S)-$$"
mkdir -p "$backup_dir"
for unit in gost gost-panel; do
  [[ ! -f /etc/systemd/system/$unit.service ]] || cp -p "/etc/systemd/system/$unit.service" "$backup_dir/$unit.service"
done
[[ ! -f "$INSTALL_DIR/gost-panel" ]] || cp -p "$INSTALL_DIR/gost-panel" "$backup_dir/gost-panel"
install -m 755 "$PKG_DIR/gost-panel" "$INSTALL_DIR/gost-panel.next"
mv -f "$INSTALL_DIR/gost-panel.next" "$INSTALL_DIR/gost-panel"
cp -R "$PKG_DIR/static" "$work_dir/static"
[[ ! -d "$INSTALL_DIR/static" ]] || mv "$INSTALL_DIR/static" "$backup_dir/static"
cp -R "$work_dir/static" "$INSTALL_DIR/static"
find "$INSTALL_DIR/static" -type d -exec chmod 755 {} +
find "$INSTALL_DIR/static" -type f -exec chmod 644 {} +

python3 <<'PY'
import json, os, pathlib
env = os.environ
state_dir = pathlib.Path(env['STATE_DIR'])
def private_write(path, text):
    path.write_text(text)
    path.chmod(0o600)
config_path = pathlib.Path('/opt/gost/config.json')
if not config_path.exists():
    private_write(config_path, json.dumps({
        'api': {'addr': '127.0.0.1:' + env['GOST_API_PORT'],
                'auth': {'username': 'admin', 'password': env['GOST_API_PASS']}},
        'services': []}, indent=2) + '\n')
values = {k: env[k] for k in ('INSTALL_DIR', 'PANEL_HOST', 'PANEL_PORT', 'PANEL_USER', 'PANEL_PASS', 'GOST_API_PASS')}
private_write(state_dir / 'install.json', json.dumps(values, indent=2) + '\n')
values = {k: env[k] for k in ('PANEL_HOST', 'PANEL_PORT', 'PANEL_USER', 'PANEL_PASS', 'GOST_API_PASS')}
values.update(GOST_PANEL_DIR=env['INSTALL_DIR'], GOST_API_USER='admin',
              GOST_API_URL='http://127.0.0.1:' + env['GOST_API_PORT'], GOST_CONFIG_PATH=str(config_path))
private_write(state_dir / 'panel.env', ''.join(k + '=' + json.dumps(v, ensure_ascii=False) + '\n' for k, v in values.items()))
PY

cat > /etc/systemd/system/gost.service <<'UNIT'
[Unit]
Description=GOST v3 proxy/tunnel engine
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /opt/gost/config.json
Restart=always
RestartSec=3
LimitNOFILE=65535
UMask=0077

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/gost-panel.service <<UNIT
[Unit]
Description=gost-panel - GOST forwarding management panel
After=network-online.target gost.service
Wants=network-online.target gost.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$STATE_DIR/panel.env
ExecStart=$INSTALL_DIR/gost-panel
Restart=always
RestartSec=3
LimitNOFILE=65535
UMask=0077

[Install]
WantedBy=multi-user.target
UNIT
chmod 644 /etc/systemd/system/gost.service /etc/systemd/system/gost-panel.service
systemd-analyze verify /etc/systemd/system/gost.service /etc/systemd/system/gost-panel.service
systemctl daemon-reload
systemctl enable gost gost-panel
systemctl restart gost
systemctl restart gost-panel

# 自检不打印密码或 token；失败即退出，绝不仅凭 systemctl active 报安装成功。
python3 <<'PY'
import json, os, time, urllib.request
env = os.environ
base = 'http://127.0.0.1:' + env['PANEL_PORT']
error = None
for attempt in range(30):
    try:
        with urllib.request.urlopen(base + '/', timeout=3) as r:
            assert r.status == 200 and '<html' in r.read().decode().lower(), '前端首页错误'
        req = urllib.request.Request(base + '/api/login', method='POST',
              data=json.dumps({'username': env['PANEL_USER'], 'password': env['PANEL_PASS']}).encode(),
              headers={'Content-Type': 'application/json'})
        with urllib.request.urlopen(req, timeout=3) as r:
            login = json.load(r)
        assert login['code'] == 0, '面板登录失败'
        req = urllib.request.Request(base + '/api/status', headers={'Authorization': 'Bearer ' + login['data']['token']})
        with urllib.request.urlopen(req, timeout=3) as r:
            status = json.load(r)['data']
        assert status['gost']['active'] and status['gost']['api'] == 'up', 'gost 引擎或 API 未就绪'
        print('[INFO] 自检通过：前端 HTTP、面板登录、gost 引擎、gost API')
        break
    except Exception as e:
        error = type(e).__name__ + ': ' + str(e)
        time.sleep(1)
else:
    raise SystemExit('[ERROR] 自检失败（' + str(error) + '），请检查 journalctl -u gost-panel -u gost')
PY
systemctl is-active --quiet gost
systemctl is-active --quiet gost-panel
info "安装完成：监听 $PANEL_HOST:$PANEL_PORT，版本 $version"
info "登录用户名：$PANEL_USER；凭据保存于 $STATE_DIR/install.json（仅 root 可读）"
info '服务：systemctl status gost gost-panel；开机自启和异常自动重启已开启'
