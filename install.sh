#!/bin/bash
# ============================================================
# gost-panel 一键安装脚本
# 用法：
#   curl -sL https://raw.githubusercontent.com/starlgz/gost-panel/main/install.sh | bash
#   或：bash install.sh [--gost-api-pass xxx] [--panel-pass xxx]
# ============================================================
set -euo pipefail

# ---------- 颜色 ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()   { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# ---------- 参数 ----------
GOST_API_PASS=""
PANEL_PASS=""
PANEL_PORT="19090"
GOST_API_PORT="18080"
REPO_URL="https://github.com/starlgz/gost-panel.git"
GOST_VERSION="v3.2.6"
INSTALL_DIR="/opt/gost-panel"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --gost-api-pass) GOST_API_PASS="$2"; shift 2 ;;
    --panel-pass)    PANEL_PASS="$2";    shift 2 ;;
    --panel-port)    PANEL_PORT="$2";    shift 2 ;;
    --install-dir)   INSTALL_DIR="$2";   shift 2 ;;
    *) warn "未知参数: $1"; shift ;;
  esac
done

# 密码：未指定则随机生成
gen_pass() { tr -dc 'a-zA-Z0-9' </dev/urandom | head -c 16; }
[ -z "$GOST_API_PASS" ] && GOST_API_PASS=$(gen_pass)
[ -z "$PANEL_PASS" ]    && PANEL_PASS=$(gen_pass)

# ---------- 前置检查 ----------
[[ $EUID -ne 0 ]] && err "请用 root 运行：sudo bash install.sh"
command -v curl >/dev/null || err "需要 curl"
command -v systemctl >/dev/null || err "需要 systemd"

# ---------- 1. 安装 gost ----------
if command -v gost >/dev/null 2>&1; then
  info "gost 已安装: $(gost -V 2>&1 | head -1)"
else
  info "下载 gost $GOST_VERSION ..."
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64)  G_ARCH="amd64" ;;
    aarch64|arm64) G_ARCH="arm64" ;;
    *) err "不支持的架构: $ARCH" ;;
  esac
  curl -sL -o /tmp/gost.tar.gz \
    "https://github.com/go-gost/gost/releases/download/${GOST_VERSION}/gost_${GOST_VERSION#v}_linux_${G_ARCH}.tar.gz"
  tar xzf /tmp/gost.tar.gz -C /tmp
  install -m 0755 /tmp/gost /usr/local/bin/gost
  info "gost 安装完成: $(gost -V 2>&1 | head -1)"
fi

# ---------- 2. gost 配置 + systemd ----------
mkdir -p /opt/gost
cat > /opt/gost/config.json <<EOF
{
  "api": {
    "addr": ":${GOST_API_PORT}",
    "auth": { "username": "admin", "password": "${GOST_API_PASS}" }
  },
  "services": []
}
EOF

cat > /etc/systemd/system/gost.service <<EOF
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

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now gost >/dev/null 2>&1
info "gost 服务已启动 (API: 127.0.0.1:${GOST_API_PORT})"

# ---------- 3. 部署面板代码 ----------
if [[ -d "$INSTALL_DIR/.git" ]]; then
  info "面板目录已存在，更新代码..."
  cd "$INSTALL_DIR" && git pull --ff-only || warn "git pull 失败，继续使用现有代码"
else
  info "克隆面板代码到 $INSTALL_DIR ..."
  git clone --depth 1 "$REPO_URL" "$INSTALL_DIR"
fi
cd "$INSTALL_DIR"

# ---------- 4. 后端依赖 ----------
if ! command -v python3 >/dev/null 2>&1; then
  err "需要 python3"
fi
if ! python3 -m venv --help >/dev/null 2>&1; then
  info "安装 python3-venv ..."
  apt-get update -qq && apt-get install -y -qq python3-venv >/dev/null
fi
if [[ ! -d venv ]]; then
  python3 -m venv venv
fi
./venv/bin/pip install -q --upgrade pip >/dev/null 2>&1 || true
./venv/bin/pip install -q fastapi "uvicorn[standard]" pyjwt passlib
info "后端依赖安装完成"

# ---------- 5. 前端构建 ----------
if command -v npm >/dev/null 2>&1 && command -v node >/dev/null 2>&1; then
  if [[ ! -d web/node_modules ]]; then
    info "安装前端依赖并构建..."
    (cd web && npm install --no-audit --no-fund >/dev/null 2>&1 && npm run build >/dev/null 2>&1)
  elif [[ ! -d static ]]; then
    (cd web && npm run build >/dev/null 2>&1)
  fi
  info "前端构建完成"
else
  warn "未检测到 node/npm，跳过前端构建（面板只能访问 API，无界面）"
  warn "请手动安装 Node.js 后执行：cd $INSTALL_DIR/web && npm install && npm run build"
fi

# ---------- 6. 面板 systemd 服务 ----------
cat > /etc/systemd/system/gost-panel.service <<EOF
[Unit]
Description=gost-panel - GOST forwarding management panel
After=network-online.target gost.service
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
Environment=GOST_API_PASS=${GOST_API_PASS}
Environment=PANEL_PASS=${PANEL_PASS}
Environment=PANEL_PORT=${PANEL_PORT}
ExecStart=$INSTALL_DIR/venv/bin/python $INSTALL_DIR/panel.py
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable --now gost-panel >/dev/null 2>&1
sleep 2
systemctl is-active gost-panel >/dev/null || err "面板启动失败，查看日志：journalctl -u gost-panel -n 50"
info "面板服务已启动 (127.0.0.1:${PANEL_PORT})"

# ---------- 完成 ----------
echo
echo "============================================"
echo "  gost-panel 安装完成！"
echo "============================================"
echo "  gost API:    http://127.0.0.1:${GOST_API_PORT}  (admin / ${GOST_API_PASS})"
echo "  面板地址:    http://127.0.0.1:${PANEL_PORT}    (admin / ${PANEL_PASS})"
echo "  面板目录:    $INSTALL_DIR"
echo "--------------------------------------------"
echo "  公网访问方式（任选）："
echo "  1) SSH 隧道:  ssh -L ${PANEL_PORT}:127.0.0.1:${PANEL_PORT} root@服务器IP"
echo "  2) 反代/隧道: 配置 nginx 或 cloudflared 指向 127.0.0.1:${PANEL_PORT}"
echo "============================================"
echo
warn "请务必修改面板密码：编辑 /etc/systemd/system/gost-panel.service 中的 PANEL_PASS"
warn "改完执行：systemctl daemon-reload && systemctl restart gost-panel"
