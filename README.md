<div align="center">

# 🖤 gost-panel

**GOST v3 转发管理面板** — Web 界面管理端口转发与传输隧道，热更新不断流

![Python](https://img.shields.io/badge/Python-3.11-3776AB?logo=python&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)
![FastAPI](https://img.shields.io/badge/FastAPI-0.141-009688?logo=fastapi&logoColor=white)
![gost](https://img.shields.io/badge/gost-v3.2.6-000000)
![License](https://img.shields.io/badge/License-MIT-green)

简体中文 | [English](./README.en.md)

</div>

---

## 📌 简介

gost-panel 是一个基于 [gost v3](https://github.com/go-gost/gost) 的转发管理面板：

- **端口转发**：TCP / UDP 规则增删改查，秒级生效
- **传输隧道**：WS / WSS / gRPC / QUIC / HTTP2 入站伪装，自动生成客户端命令
- **热更新**：规则变更通过 gost Web API 实时同步，**不重启引擎、不断已有连接**
- **配置持久化**：规则写入 gost 配置文件，gost 意外重启后自动恢复

典型场景：`客户端 → 中转机(gost-panel) → 落地机`，把服务器变成一个可管理的转发/中转节点。

## ✨ 功能特性

| 功能 | 说明 |
|------|------|
| 📡 端口转发 | TCP/UDP 规则管理：增删改查、启停、备注、端口占用检查 |
| 🚇 传输隧道 | WS/WSS/gRPC/QUIC/HTTP2 入站伪装，可选认证，自动生成客户端连接命令 |
| ⚡ 热更新 | 对象级 API 同步（services/chains），不断流 |
| 💾 持久化 | 配置落盘，gost 重启自动恢复 |
| 📊 状态页 | gost 引擎/API 状态、活动规则数实时查看 |
| 📋 审计日志 | 登录、规则操作全程留痕 |
| 🔐 安全 | JWT 登录鉴权，密码环境变量化，默认仅本机监听 |

## 🏗 架构

```
 客户端 ──► gost 引擎 (监听端口/隧道) ──► 目标服务器
                 ▲
                 │ gost Web API (对象级热更新)
         ┌───────┴───────┐
         │  面板后端      │  FastAPI + SQLite + JWT
         │  (127.0.0.1)  │
         └───────┬───────┘
                 ▲
                 │ HTTPS (SSH隧道 / Nginx / Cloudflare Tunnel)
         ┌───────┴───────┐
         │  面板前端      │  Vue 3 + Vite 单页应用
         └───────────────┘
```

## 🚀 快速开始

### 一键安装（推荐）

```bash
curl -sL https://raw.githubusercontent.com/starlgz/gost-panel/master/install.sh | bash
```

脚本自动完成：安装 gost → 配置 API → 克隆代码 → 安装依赖 → 构建前端 → 注册 systemd 服务 → 打印随机生成的密码。

自定义参数：

```bash
curl -sL https://raw.githubusercontent.com/starlgz/gost-panel/master/install.sh | bash -s -- \
  --panel-pass mypassword --panel-port 19090
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--gost-api-pass` | 随机生成 | gost Web API 密码 |
| `--panel-pass` | 随机生成 | 面板登录密码 |
| `--panel-port` | `19090` | 面板监听端口 |
| `--install-dir` | `/opt/gost-panel` | 安装目录 |

### 手动安装

<details>
<summary>展开详细步骤</summary>

**1. 安装 gost v3**

```bash
curl -sL -o /tmp/gost.tar.gz https://github.com/go-gost/gost/releases/download/v3.2.6/gost_3.2.6_linux_amd64.tar.gz
tar xzf /tmp/gost.tar.gz -C /tmp && install -m 0755 /tmp/gost /usr/local/bin/gost
```

**2. gost 配置（启用 Web API）**

```bash
mkdir -p /opt/gost
cat > /opt/gost/config.json <<'EOF'
{
  "api": {
    "addr": ":18080",
    "auth": { "username": "admin", "password": "your-api-password" }
  },
  "services": []
}
EOF
```

**3. gost systemd 服务**

```bash
cat > /etc/systemd/system/gost.service <<'EOF'
[Unit]
Description=GOST v3 proxy/tunnel engine
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gost -C /opt/gost/config.json
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now gost
```

**4. 部署面板**

```bash
git clone https://github.com/starlgz/gost-panel.git /opt/gost-panel
cd /opt/gost-panel

python3 -m venv venv
./venv/bin/pip install fastapi "uvicorn[standard]" pyjwt passlib

cd web && npm install && npm run build && cd ..   # 产物输出到 ../static/
```

**5. 面板 systemd 服务**

```bash
cat > /etc/systemd/system/gost-panel.service <<'EOF'
[Unit]
Description=gost-panel - GOST forwarding management panel
After=network-online.target gost.service

[Service]
Type=simple
WorkingDirectory=/opt/gost-panel
Environment=GOST_API_PASS=your-api-password
Environment=PANEL_PASS=your-panel-password
ExecStart=/opt/gost-panel/venv/bin/python /opt/gost-panel/panel.py
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now gost-panel
```

</details>

### 访问面板

面板默认监听 `127.0.0.1:19090`，请通过以下任一方式访问：

- **SSH 隧道**：`ssh -L 19090:127.0.0.1:19090 root@服务器IP`，然后浏览器打开 `http://127.0.0.1:19090`
- **Nginx 反代**：将域名反代到 `127.0.0.1:19090`
- **Cloudflare Tunnel**：`cloudflared tunnel --url http://127.0.0.1:19090`

## 📖 使用指南

### 端口转发（普通转发）

```
规则类型：端口转发
监听端口：8080
目标主机：1.2.3.4
目标端口：80
```

保存后 gost 秒级监听 8080，流量直达 `1.2.3.4:80`。

### 传输隧道（入站伪装）

适合科学上网/内网穿透，链路：

```
客户端(net=ws) → 中转机:47700 (gost WS监听) → 落地机:47700 (vmess/ss)
```

1. 规则类型选「传输隧道」，协议选 WS，填监听端口和目标地址
2. 保存后点「命令」按钮，复制客户端连接命令
3. 在客户端机器上运行 gost，即可通过隧道访问落地机

> 💡 隧道认证可选：v2rayN 等客户端不会在 WS 层带认证头，**用 vmess/ss 等加密协议时建议不设隧道认证**（协议自身已加密）。

## ⚙️ 配置（环境变量）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GOST_API_URL` | `http://127.0.0.1:18080` | gost Web API 地址 |
| `GOST_API_USER` | `admin` | gost API 用户名 |
| `GOST_API_PASS` | `gost-panel-change-me` | gost API 密码（必改） |
| `PANEL_USER` | `admin` | 面板登录用户名 |
| `PANEL_PASS` | `gost-panel-change-me` | 面板登录密码（必改） |
| `JWT_SECRET` | 自动生成并持久化 | JWT 签名密钥 |
| `JWT_EXPIRE_HOURS` | `24` | Token 有效期（小时） |
| `PANEL_HOST` | `127.0.0.1` | 面板监听地址 |
| `PANEL_PORT` | `19090` | 面板监听端口 |

## 🔌 API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/login` | 登录获取 JWT |
| GET | `/api/status` | gost/面板状态 |
| GET | `/api/rules` | 规则列表（含客户端命令） |
| POST | `/api/rules` | 新建规则 |
| PUT | `/api/rules/{id}` | 更新规则 |
| DELETE | `/api/rules/{id}` | 删除规则 |
| POST | `/api/rules/apply` | 手动同步到 gost |
| GET | `/api/audit` | 审计日志 |

所有接口（除 login 外）需 `Authorization: Bearer <token>`。

## 🗺 路线图

- [ ] 流量统计 / 限速（gost Limiter/Quota）
- [ ] 账号订阅体系（多用户）
- [ ] 多节点管理
- [ ] WSS 自动证书（Let's Encrypt）
- [ ] Docker 部署

## 🛠 技术栈

- **后端**：Python 3.11 / FastAPI / SQLite / PyJWT
- **前端**：Vue 3 / Vue Router / Vite
- **引擎**：gost v3.2.6

## 📄 License

[MIT](./LICENSE)
