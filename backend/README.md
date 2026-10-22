# gost-panel backend（Go 版后端）

Go（Gin）版后端 — GOST v3 转发管理面板：端口转发 + 传输隧道 + 热更新 + 审计。

可直接托管根目录 `static/` 前端构建产物。
**纯 Go 实现：SQLite 使用 `modernc.org/sqlite` 驱动，无 CGO 依赖，可任意交叉编译。**

## 项目结构

```
gost-panel/
├── backend/                  # ← 本目录（Go 版后端）
│   ├── cmd/
│   │   └── gost-panel/
│   │       └── main.go      # 入口：装配 config → store → gostapi → auth → api
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go    # 配置加载（环境变量优先 + 默认值）
│   │   ├── store/
│   │   │   └── store.go     # SQLite 存储层：rules/audit 表、CRUD、迁移
│   │   ├── gostapi/
│   │   │   └── gostapi.go   # gost Web API 客户端：热更新 diff 同步、配置编译与持久化
│   │   ├── auth/
│   │   │   └── auth.go      # JWT 签发/校验中间件、密钥持久化
│   │   ├── api/
│   │   │   └── server.go    # 路由注册与 HTTP 处理函数
│   │   └── webui/
│   │       └── webui.go     # 前端静态文件托管（SPA fallback）
│   ├── Makefile
│   ├── go.mod
│   └── README.md            # 本文档
├── web/                      # Vue 前端源码
└── static/                   # 前端构建产物（后端托管目录）
```

## 技术栈

| 功能 | 库 |
|------|-----|
| Web 框架 | [gin-gonic/gin](https://github.com/gin-gonic/gin) |
| SQLite（纯 Go） | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) |
| JWT | [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt/v5) |

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/login` | 登录，返回 JWT |
| GET | `/api/status` | gost 引擎状态 + 活动规则数 |
| GET | `/api/rules` | 规则列表（隧道规则含 `client_cmd`） |
| POST | `/api/rules` | 创建规则（端口占用检查，自动同步 gost） |
| PUT | `/api/rules/:rid` | 更新规则 |
| DELETE | `/api/rules/:rid` | 删除规则 |
| POST | `/api/rules/apply` | 手动全量同步 |
| GET | `/api/audit?limit=50` | 审计日志 |

所有 `/api`（除 login）需 `Authorization: Bearer <token>`。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GOST_PANEL_DIR` | `/opt/gost-panel` | 工作目录（data、static 所在） |
| `GOST_API_URL` | `http://127.0.0.1:18080` | gost Web API 地址 |
| `GOST_API_USER` / `GOST_API_PASS` | `admin` / `gost-panel-change-me` | gost API 凭证 |
| `PANEL_USER` / `PANEL_PASS` | `admin` / `gost-panel-change-me` | 面板登录凭证 |
| `JWT_SECRET` | 自动生成并持久化到 `data/jwt_secret` | JWT 密钥 |
| `JWT_EXPIRE_HOURS` | `24` | token 有效期 |
| `PANEL_HOST` / `PANEL_PORT` | `127.0.0.1` / `19090` | 面板监听地址 |
| `GOST_CONFIG_PATH` | `/opt/gost/config.json` | 持久化写回的 gost 配置 |

## 构建与运行

```bash
make build          # 当前平台
make linux          # 交叉编译 Linux amd64（部署服务器用）

# 运行（默认 127.0.0.1:19090）
GOST_PANEL_DIR=/opt/gost-panel ./gost-panel-linux-amd64
```

把原项目构建好的前端放到 `$GOST_PANEL_DIR/static/` 即可由本服务直接托管（SPA fallback 已内置）。

## systemd 示例

```ini
[Unit]
Description=gost-panel
After=network.target gost.service

[Service]
Environment=GOST_PANEL_DIR=/opt/gost-panel
Environment=PANEL_PASS=change-me
ExecStart=/opt/gost-panel/gost-panel-linux-amd64
Restart=always

[Install]
WantedBy=multi-user.target
```
