<div align="center">

# 🖤 gost-panel

**GOST v3 转发管理面板** — Web 界面管理端口转发与传输隧道，热更新不断流

![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-v1.12-008ECF?logo=goframeworksdotorg&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)
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
         │  面板后端      │  Go (Gin) + SQLite + JWT
         │  (127.0.0.1)  │
         └───────┬───────┘
                 ▲
                 │ HTTPS (SSH隧道 / Nginx / Cloudflare Tunnel)
         ┌───────┴───────┐
         │  面板前端      │  Vue 3 + Vite 单页应用
         └───────────────┘
```

## 🚀 快速开始

### 一键安装（预编译包，无需 Go / Node.js）

```bash
curl -fL --retry 3 https://github.com/kingchun/gost-panel/releases/latest/download/install.sh -o install.sh
sudo bash install.sh --host 0.0.0.0
```

脚本自动识别 Linux **amd64（x86_64）/ arm64（aarch64）**，从 [GitHub Releases](https://github.com/kingchun/gost-panel/releases) 下载对应架构的前后端发布包，校验 SHA256 后安装 gost、注册 systemd 并验证登录与 API。服务器无需 git、Go 或 Node.js。

> 支持 Debian/Ubuntu 和使用 dnf 的 systemd 系统。默认仅监听 `127.0.0.1`；命令中的 `--host 0.0.0.0` 开启外部访问。

自定义参数：

```bash
sudo bash install.sh --version v0.1.0 --user admin --panel-pass '替换成你的密码' --host 0.0.0.0 --port 19090

# 也可以只下载并校验发布包，稍后离线安装（不需要 root）
bash install.sh --version v0.1.0 --download-only ./packages
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--version` | 最新正式版 | 指定 GitHub Release 版本，如 `v0.1.0` |
| `--download-only` | 不启用 | 仅下载并校验发布包到指定目录 |
| `--archive` | 不启用 | 使用本地 tar.gz 包部署 |
| `--package-dir` | 不启用 | 使用已解包目录部署 |
| `--source` | 不启用 | 从源码构建；需满足 `backend/go.mod` 要求的 Go，缺少前端产物时还需 npm |
| `--user` | `admin` | 首次安装的面板用户名 |
| `--gost-api-pass` | 随机生成 | gost Web API 密码 |
| `--panel-pass` | 随机生成 | 面板登录密码 |
| `--panel-port` | `19090` | 面板监听端口 |
| `--host` | `127.0.0.1` | 面板监听地址；直接公网访问使用 `0.0.0.0` |
| `--install-dir` | `/opt/gost-panel` | 安装目录 |

### 离线部署（预编译发布包，生产推荐）

服务器**无需安装 Go / Node.js**，直接用打包好的二进制与前端产物：

```bash
# 1. 本地打包（Windows）
cd web
npm ci
npm run build
cd ..
pwsh -File scripts/build-release.ps1 -Version v0.1.0
#   产出两个架构的 tar.gz、install.sh 和 SHA256SUMS.txt，作为同一 Release 的附件上传

# 2. 上传到服务器
scp release/gost-panel-v0.1.0-linux-amd64.tar.gz root@服务器IP:/root/
scp release/install.sh release/SHA256SUMS.txt root@服务器IP:/root/

# 3. 服务器上一键部署
ssh root@服务器IP
cd /root
grep '  gost-panel-v0.1.0-linux-amd64.tar.gz$' SHA256SUMS.txt | sha256sum -c -
sudo bash install.sh --archive /root/gost-panel-v0.1.0-linux-amd64.tar.gz --host 0.0.0.0
```

安装入口调用包内 `deploy.sh`，自动完成：安装缺失的 curl/tar/python3 → 下载并校验 gost → 写本机 API 配置 → 部署二进制与前端 → 注册并启用 systemd → 验证首页、面板登录、gost 引擎和 API。需要 Debian/Ubuntu 或使用 dnf 的 systemd 系统；安装 gost 时需要联网。

重复执行保留规则数据库、gost 配置和已有密码，并重启服务使用新版本。旧二进制与静态文件保存在安装目录的 `backups/`。凭据保存于 `/etc/gost-panel/install.json` 和 `/etc/gost-panel/panel.env`，权限 `0600`，不会打印到终端或写在可公开读取的 unit 文件中。安装器遇到未接管的现有 gost 配置会停止，避免覆盖。

也可自行解包，在包目录运行 `sudo bash deploy.sh --host 0.0.0.0`。两种入口使用同一部署流程。

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--dir` | `/opt/gost-panel` | 安装目录 |
| `--port` | `19090` | 面板监听端口 |
| `--host` | `127.0.0.1` | 面板监听地址 |
| `--user` | `admin` | 面板登录用户名 |
| `--panel-pass` | 随机生成 | 面板登录密码 |
| `--gost-api-pass` | 随机生成 | gost API 密码 |

> 后端为**静态编译**（`CGO_ENABLED=0`、`-trimpath`），不依赖服务器的 Go、Node.js 或动态 C 库；此部署脚本需要 systemd，不能直接用于默认使用 OpenRC 的 Alpine。
> 包内 `BUILD_INFO.txt` 记录版本、构建时间与 git commit。

在完整源码目录执行 `sudo bash install.sh --source --host 0.0.0.0` 可显式使用源码构建。安装器回归测试：`python3 scripts/tests/test_install.py`（Linux，需 bash、python3、sha256sum；安装路径测试需 root，测试中的部署脚本为隔离夹具，不修改系统服务）。

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
git clone https://github.com/kingchun/gost-panel.git /opt/gost-panel
cd /opt/gost-panel

# 编译后端（纯 Go，无 CGO）
cd backend && CGO_ENABLED=0 go build -ldflags "-s -w" -o gost-panel ./cmd/gost-panel && cd ..

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
Environment=GOST_PANEL_DIR=/opt/gost-panel
Environment=GOST_API_PASS=your-api-password
Environment=PANEL_PASS=your-panel-password
ExecStart=/opt/gost-panel/backend/gost-panel
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
| GET | `/api/status` | gost/面板状态（面板版本由构建注入） |
| GET | `/api/rules` | 规则列表（含客户端命令）；带 `page`/`page_size`/`keyword` 时返回分页结构 |
| GET | `/api/rules/stats` | 规则聚合统计（总数/启用/TCP/UDP/隧道） |
| POST | `/api/rules` | 新建规则 |
| PUT | `/api/rules/{id}` | 更新规则 |
| DELETE | `/api/rules/{id}` | 删除规则 |
| POST | `/api/rules/apply` | 手动同步到 gost |
| GET | `/api/gost/services` | 拉取 gost 当前运行的服务列表（用于比对） |
| POST | `/api/gost/import` | 把 gost 现有服务导入为面板规则 |
| GET | `/api/audit` | 审计日志（支持 `limit` 或分页参数） |
| DELETE | `/api/audit` | **一键清空审计日志** |
| POST | `/api/change-password` | 修改面板登录密码 |

所有接口（除 login 外）需 `Authorization: Bearer <token>`。
响应统一为 `{"code":0,"message":"ok","data":...}`，失败时 `code != 0`。

## 🗺 路线图

- [ ] 流量统计 / 限速（gost Limiter/Quota）
- [ ] 账号订阅体系（多用户）
- [ ] 多节点管理
- [ ] WSS 自动证书（Let's Encrypt）
- [ ] Docker 部署

## 🛠 技术栈

- **后端**：Go / Gin / SQLite（modernc.org/sqlite 纯 Go 驱动，无 CGO）/ golang-jwt
- **前端**：Vue 3 / Vue Router / Vite
- **引擎**：gost v3.2.6

## 📄 License

[MIT](./LICENSE)
