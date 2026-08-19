# gost-panel

基于 [gost v3](https://github.com/go-gost/gost) 的转发管理面板：Web 界面管理端口转发与传输隧道规则，通过 gost Web API 热更新，**不重启引擎、不断流**。

## 功能

- 📡 **端口转发**：TCP/UDP 规则管理（增删改查、启停、备注）
- 🚇 **传输隧道**：WS / WSS / gRPC / QUIC / HTTP2 入站伪装，自动生成客户端连接命令
- ⚡ **热更新**：规则变更走 gost 对象级 API 实时同步，不断已有连接
- 💾 **配置持久化**：规则写入 gost 配置文件，gost 重启后自动恢复
- 📊 **状态页**：gost 引擎/API 状态、活动规则数
- 📋 **审计日志**：登录、规则操作全部留痕
- 🔐 **JWT 登录鉴权**

## 架构

```
客户端 ──► gost (监听端口, 转发/隧道) ──► 目标服务器
                ▲
                │ Web API (热更新)
            面板后端 (FastAPI + SQLite)
                ▲
                │ HTTPS
            面板前端 (Vue3 + Vite)
```

## 快速开始

### 1. 安装 gost

```bash
# 下载 gost v3（Linux amd64 示例）
curl -sL -o gost.tar.gz https://github.com/go-gost/gost/releases/download/v3.2.6/gost_3.2.6_linux_amd64.tar.gz
tar xzf gost.tar.gz && mv gost /usr/local/bin/gost

# 配置文件 /opt/gost/config.json（API 认证）
cat > /opt/gost/config.json <<'EOF'
{
  "api": {
    "addr": ":18080",
    "auth": { "username": "admin", "password": "your-gost-api-password" }
  },
  "services": []
}
EOF

# systemd 服务
cat > /etc/systemd/system/gost.service <<'EOF'
[Unit]
Description=GOST v3 proxy/tunnel engine
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /opt/gost/config.json
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now gost
```

### 2. 部署面板

```bash
git clone <本仓库> /opt/gost-panel
cd /opt/gost-panel

# 后端
python3 -m venv venv
./venv/bin/pip install fastapi "uvicorn[standard]" pyjwt passlib
cat > /etc/systemd/system/gost-panel.service <<'EOF'
[Unit]
Description=gost-panel
After=network-online.target gost.service

[Service]
Type=simple
WorkingDirectory=/opt/gost-panel
Environment=GOST_API_PASS=your-gost-api-password
Environment=PANEL_PASS=your-panel-password
ExecStart=/opt/gost-panel/venv/bin/python /opt/gost-panel/panel.py
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now gost-panel

# 前端（构建）
cd /opt/gost-panel/web
npm install && npm run build   # 产物输出到 ../static/
```

### 3. 访问

面板默认监听 `127.0.0.1:19090`，建议通过 SSH 隧道 / 反代 / Cloudflare Tunnel 访问。

## 配置（环境变量）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GOST_API_URL` | `http://127.0.0.1:18080` | gost Web API 地址 |
| `GOST_API_USER` | `admin` | gost API 用户名 |
| `GOST_API_PASS` | `gost-panel-change-me` | gost API 密码 |
| `PANEL_USER` | `admin` | 面板登录用户名 |
| `PANEL_PASS` | `gost-panel-change-me` | 面板登录密码 |
| `JWT_SECRET` | 自动生成并持久化 | JWT 签名密钥 |
| `JWT_EXPIRE_HOURS` | `24` | Token 有效期（小时） |
| `PANEL_HOST` | `127.0.0.1` | 面板监听地址 |
| `PANEL_PORT` | `19090` | 面板监听端口 |

> ⚠️ 生产环境务必通过环境变量设置 `PANEL_PASS` 和 `GOST_API_PASS`。

## 技术栈

- 后端：Python 3.11 / FastAPI / SQLite / PyJWT
- 前端：Vue 3 / Vue Router / Vite
- 引擎：gost v3.2.6

## License

MIT
