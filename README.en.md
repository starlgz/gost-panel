# gost-panel

**GOST v3 forwarding management panel** — manage port forwarding and transport tunnels via a Web UI, with hot-reload (no restart, no connection drop).

## Features

- 📡 **Port forwarding**: TCP/UDP rule management (CRUD, enable/disable, remarks)
- 🚇 **Transport tunnels**: WS / WSS / gRPC / QUIC / HTTP2 inbound disguise, auto-generated client commands
- ⚡ **Hot reload**: rules sync via gost Web API in real time, no restart
- 💾 **Persistence**: rules written to gost config file, auto-recover after gost restart
- 📊 **Status page**: gost engine/API status, active rule count
- 📋 **Audit log**: all login and rule operations tracked
- 🔐 **JWT auth**

## Quick Start (one command)

```bash
curl -fL --retry 3 https://github.com/kingchun/gost-panel/releases/latest/download/install.sh -o install.sh
sudo bash install.sh --host 0.0.0.0
```

The installer detects Linux amd64/arm64, downloads the matching prebuilt frontend/backend package from GitHub Releases, verifies SHA256, and installs the systemd services. No Go, Node.js, or git is needed on the server. Debian/Ubuntu and dnf-based systemd distributions are supported.

Custom options:

```bash
sudo bash install.sh --version v0.1.0 --user admin --panel-pass 'your-password' --host 0.0.0.0 --port 19090

# Download and verify only; no root required.
bash install.sh --version v0.1.0 --download-only ./packages
```

For offline installation, use `sudo bash install.sh --archive ./gost-panel-v0.1.0-linux-amd64.tar.gz --host 0.0.0.0`. Use `--source` explicitly to compile from source. Reinstallation preserves rules, GOST configuration, and existing credentials.

## Architecture

```
 Client ──► gost engine (listen/tunnel) ──► Target server
                 ▲
                 │ gost Web API (hot reload)
         ┌───────┴───────┐
         │  Panel Backend │  Go (Gin) + SQLite + JWT
         └───────┬───────┘
                 ▲
                 │ HTTPS (SSH tunnel / Nginx / Cloudflare Tunnel)
         ┌───────┴───────┐
         │  Panel Frontend │  Vue 3 + Vite
         └───────────────┘
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOST_API_URL` | `http://127.0.0.1:18080` | gost Web API URL |
| `GOST_API_USER` | `admin` | gost API username |
| `GOST_API_PASS` | `gost-panel-change-me` | gost API password (change it!) |
| `PANEL_USER` | `admin` | panel login username |
| `PANEL_PASS` | `gost-panel-change-me` | panel login password (change it!) |
| `JWT_SECRET` | auto-generated & persisted | JWT signing secret |
| `JWT_EXPIRE_HOURS` | `24` | Token lifetime (hours) |
| `PANEL_HOST` | `127.0.0.1` | panel listen address |
| `PANEL_PORT` | `19090` | panel listen port |

## Tech Stack

- Backend: Go / Gin / SQLite (pure-Go driver, no CGO) / golang-jwt
- Frontend: Vue 3 / Vue Router / Vite
- Engine: gost v3.2.6

## License

MIT
