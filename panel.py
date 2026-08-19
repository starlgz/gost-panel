#!/usr/bin/env python3
"""
gost-panel - GOST v3 转发面板后端
FastAPI + SQLite + JWT 认证，规则通过 gost Web API 热更新（不断流）
"""
import json
import sqlite3
import time
import secrets
import subprocess
import urllib.request
import urllib.error
from datetime import datetime, timedelta, timezone
from pathlib import Path
from contextlib import contextmanager

import jwt
from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

# ---------- 配置（环境变量优先） ----------
import os

BASE_DIR = Path(os.environ.get("GOST_PANEL_DIR", "/opt/gost-panel"))
DATA_DIR = BASE_DIR / "data"
DB_PATH = DATA_DIR / "panel.db"
GOST_URL = os.environ.get("GOST_API_URL", "http://127.0.0.1:18080")
GOST_USER = os.environ.get("GOST_API_USER", "admin")
GOST_PASS = os.environ.get("GOST_API_PASS", "gost-panel-change-me")
PANEL_USER = os.environ.get("PANEL_USER", "admin")
PANEL_PASS = os.environ.get("PANEL_PASS", "gost-panel-change-me")
JWT_EXPIRE_HOURS = int(os.environ.get("JWT_EXPIRE_HOURS", "24"))
LISTEN_HOST = os.environ.get("PANEL_HOST", "127.0.0.1")
LISTEN_PORT = int(os.environ.get("PANEL_PORT", "19090"))

# JWT 密钥：持久化到 data 目录，避免每次重启 token 失效
def _load_jwt_secret() -> str:
    secret_file = DATA_DIR / "jwt_secret"
    env_secret = os.environ.get("JWT_SECRET", "")
    if env_secret:
        return env_secret
    if secret_file.exists():
        return secret_file.read_text().strip()
    s = secrets.token_hex(32)
    DATA_DIR.mkdir(parents=True, exist_ok=True)
    secret_file.write_text(s)
    return s

JWT_SECRET = _load_jwt_secret()

DATA_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="gost-panel", version="0.1.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # 面板经隧道访问，来源多变；仅本机监听
    allow_methods=["*"],
    allow_headers=["*"],
)
security = HTTPBearer()

# ---------- 数据库 ----------
def db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    return conn

def init_db():
    with db() as conn:
        conn.execute("""
            CREATE TABLE IF NOT EXISTS rules (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL,
                type TEXT NOT NULL DEFAULT 'forward',
                proto TEXT NOT NULL DEFAULT 'tcp',
                listen_port INTEGER NOT NULL,
                target_host TEXT NOT NULL,
                target_port INTEGER NOT NULL,
                tunnel_proto TEXT NOT NULL DEFAULT 'ws',
                tunnel_path TEXT DEFAULT '',
                tunnel_user TEXT DEFAULT '',
                tunnel_pass TEXT DEFAULT '',
                enabled INTEGER NOT NULL DEFAULT 1,
                remark TEXT DEFAULT '',
                created_at TEXT NOT NULL
            )
        """)
        # 旧库迁移：补列
        cols = [r[1] for r in conn.execute("PRAGMA table_info(rules)").fetchall()]
        for col, ddl in [
            ("type", "ALTER TABLE rules ADD COLUMN type TEXT NOT NULL DEFAULT 'forward'"),
            ("tunnel_proto", "ALTER TABLE rules ADD COLUMN tunnel_proto TEXT NOT NULL DEFAULT 'ws'"),
            ("tunnel_path", "ALTER TABLE rules ADD COLUMN tunnel_path TEXT DEFAULT ''"),
            ("tunnel_user", "ALTER TABLE rules ADD COLUMN tunnel_user TEXT DEFAULT ''"),
            ("tunnel_pass", "ALTER TABLE rules ADD COLUMN tunnel_pass TEXT DEFAULT ''"),
        ]:
            if col not in cols:
                conn.execute(ddl)
        conn.execute("""
            CREATE TABLE IF NOT EXISTS audit (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                ts TEXT NOT NULL,
                action TEXT NOT NULL,
                detail TEXT DEFAULT ''
            )
        """)

init_db()

# ---------- 工具 ----------
def audit(action: str, detail: str = ""):
    with db() as conn:
        conn.execute("INSERT INTO audit (ts, action, detail) VALUES (?, ?, ?)",
                     (datetime.now().isoformat(timespec="seconds"), action, detail))

def gost_request(method: str, path: str, body: dict | None = None):
    """调用 gost Web API"""
    url = GOST_URL + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", "Basic " + __import__("base64").b64encode(
        f"{GOST_USER}:{GOST_PASS}".encode()).decode())
    if data:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            raw = resp.read()
            return resp.status, json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        return e.code, None
    except Exception:
        return 0, None

def build_gost_config(rules) -> dict:
    """把面板规则编译成 gost v3 配置（全量热更新）"""
    services = []
    for r in rules:
        if not r["enabled"]:
            continue
        target = {"name": f"t-{r['id']}", "addr": f"{r['target_host']}:{r['target_port']}"}
        handler_type = "udp" if r["proto"] == "udp" else "tcp"
        services.append({
            "name": f"svc-{r['id']}",
            "addr": f":{r['listen_port']}",
            "handler": {"type": handler_type},
            "listener": {"type": handler_type},
            "forwarder": {"targets": [target]},
        })
    return {
        "api": {"addr": ":18080",
                "auth": {"username": GOST_USER, "password": GOST_PASS}},
        "services": services,
    }

def apply_to_gost() -> tuple[int, str]:
    """把数据库规则同步到 gost（对象级热更新，不断流）
    forward 规则 -> service + forwarder
    tunnel 规则 -> chain(转发链) + service(listener=隧道协议)
    """
    with db() as conn:
        rows = conn.execute("SELECT * FROM rules").fetchall()
    rules = [dict(r) for r in rows]

    # 当前 gost 上的服务/链列表
    code, cfg = gost_request("GET", "/config/services")
    if code != 200:
        return code, f"gost API 查询服务失败 ({code})"
    existing_svc = set()
    data = cfg.get("data") if isinstance(cfg, dict) else None
    if data and isinstance(data, dict):
        for s in (data.get("list") or []):
            existing_svc.add(s.get("name"))
    elif isinstance(cfg, list):
        for s in cfg:
            existing_svc.add(s.get("name"))

    code, cfg = gost_request("GET", "/config/chains")
    existing_chain = set()
    data = cfg.get("data") if isinstance(cfg, dict) else None
    if data and isinstance(data, dict):
        for c in (data.get("list") or []):
            existing_chain.add(c.get("name"))
    elif isinstance(cfg, list):
        for c in cfg:
            existing_chain.add(c.get("name"))

    wanted_svc = {}   # name -> service dict
    wanted_chain = {}  # name -> chain dict
    for r in rules:
        if not r["enabled"]:
            continue
        if r["type"] == "tunnel":
            # ---- 隧道规则：chain 指向目标，listener 用隧道协议 ----
            chain_name = f"chn-{r['id']}"
            svc_name = f"svc-{r['id']}"
            node = {"name": f"n-{r['id']}", "addr": f"{r['target_host']}:{r['target_port']}",
                    "connector": {"type": "tcp"}, "dialer": {"type": "tcp"}}
            # 客户端认证：服务端 listener 需要 auther
            listener_cfg = {"type": r["tunnel_proto"]}
            if r["tunnel_proto"] == "ws" and r["tunnel_path"]:
                listener_cfg = {"type": "ws", "metadata": {"path": r["tunnel_path"]}}
            handler_cfg = {"type": "auto", "chain": chain_name}
            if r["tunnel_user"]:
                handler_cfg["auth"] = {"username": r["tunnel_user"], "password": r["tunnel_pass"]}
            wanted_chain[chain_name] = {
                "name": chain_name,
                "hops": [{"name": f"hop-{r['id']}", "nodes": [node]}],
            }
            wanted_svc[svc_name] = {
                "name": svc_name,
                "addr": f":{r['listen_port']}",
                "handler": handler_cfg,
                "listener": listener_cfg,
            }
        else:
            # ---- 普通端口转发 ----
            svc_name = f"svc-{r['id']}"
            handler_type = "udp" if r["proto"] == "udp" else "tcp"
            wanted_svc[svc_name] = {
                "name": svc_name,
                "addr": f":{r['listen_port']}",
                "handler": {"type": handler_type},
                "listener": {"type": handler_type},
                "forwarder": {"nodes": [{"name": f"n-{r['id']}", "addr": f"{r['target_host']}:{r['target_port']}"}]},
            }

    # 删除多余服务/链
    for name in existing_svc - set(wanted_svc):
        gost_request("DELETE", f"/config/services/{name}")
    for name in existing_chain - set(wanted_chain):
        gost_request("DELETE", f"/config/chains/{name}")

    # 新增/更新服务与链
    for name, ch in wanted_chain.items():
        if name in existing_chain:
            gost_request("PUT", f"/config/chains/{name}", ch)
        else:
            gost_request("POST", "/config/chains", ch)
    for name, svc in wanted_svc.items():
        if name in existing_svc:
            gost_request("PUT", f"/config/services/{name}", svc)
        else:
            gost_request("POST", "/config/services", svc)

    # 持久化：把当前完整配置写回 gost 配置文件，防止 gost 重启后丢失
    persist_gost_config()

    return 200, f"已同步 {len(wanted_svc)} 条规则"


def build_full_config() -> dict:
    """从数据库生成完整 gost 配置（services + chains + api），用于持久化"""
    with db() as conn:
        rows = conn.execute("SELECT * FROM rules").fetchall()
    rules = [dict(r) for r in rows]
    services = []
    chains = []
    for r in rules:
        if not r["enabled"]:
            continue
        svc_name = f"svc-{r['id']}"
        if r["type"] == "tunnel":
            chain_name = f"chn-{r['id']}"
            node = {"name": f"n-{r['id']}", "addr": f"{r['target_host']}:{r['target_port']}",
                    "connector": {"type": "tcp"}, "dialer": {"type": "tcp"}}
            chains.append({"name": chain_name,
                           "hops": [{"name": f"hop-{r['id']}", "nodes": [node]}]})
            listener_cfg = {"type": r["tunnel_proto"]}
            if r["tunnel_proto"] == "ws" and r["tunnel_path"]:
                listener_cfg = {"type": "ws", "metadata": {"path": r["tunnel_path"]}}
            handler_cfg = {"type": "auto", "chain": chain_name}
            if r["tunnel_user"]:
                handler_cfg["auth"] = {"username": r["tunnel_user"], "password": r["tunnel_pass"]}
            services.append({"name": svc_name, "addr": f":{r['listen_port']}",
                             "handler": handler_cfg, "listener": listener_cfg})
        else:
            handler_type = "udp" if r["proto"] == "udp" else "tcp"
            services.append({"name": svc_name, "addr": f":{r['listen_port']}",
                             "handler": {"type": handler_type}, "listener": {"type": handler_type},
                             "forwarder": {"nodes": [{"name": f"n-{r['id']}", "addr": f"{r['target_host']}:{r['target_port']}"}]}})
    return {"api": {"addr": ":18080", "auth": {"username": GOST_USER, "password": GOST_PASS}},
            "services": services, "chains": chains}


def persist_gost_config():
    """把数据库规则完整配置写回 /opt/gost/config.json（gost 重启自动加载）"""
    try:
        cfg = build_full_config()
        Path("/opt/gost/config.json").write_text(json.dumps(cfg, indent=2, ensure_ascii=False))
    except Exception:
        pass

# ---------- 认证 ----------
def create_token(username: str) -> str:
    payload = {"sub": username,
               "exp": datetime.now(timezone.utc) + timedelta(hours=JWT_EXPIRE_HOURS)}
    return jwt.encode(payload, JWT_SECRET, algorithm="HS256")

def require_auth(cred: HTTPAuthorizationCredentials = Depends(security)):
    try:
        payload = jwt.decode(cred.credentials, JWT_SECRET, algorithms=["HS256"])
        return payload["sub"]
    except jwt.ExpiredSignatureError:
        raise HTTPException(401, "登录已过期")
    except Exception:
        raise HTTPException(401, "无效凭证")

# ---------- 模型 ----------
class LoginReq(BaseModel):
    username: str
    password: str

class RuleReq(BaseModel):
    name: str = Field(min_length=1, max_length=64)
    type: str = Field(default="forward", pattern="^(forward|tunnel)$")
    proto: str = Field(default="tcp", pattern="^(tcp|udp)$")
    listen_port: int = Field(ge=1, le=65535)
    target_host: str = Field(min_length=1)
    target_port: int = Field(ge=1, le=65535)
    tunnel_proto: str = Field(default="ws", pattern="^(ws|wss|grpc|quic|http2)$")
    tunnel_path: str = ""
    tunnel_user: str = ""
    tunnel_pass: str = ""
    enabled: bool = True
    remark: str = ""

class RulePatch(BaseModel):
    name: str | None = None
    type: str | None = None
    proto: str | None = None
    listen_port: int | None = None
    target_host: str | None = None
    target_port: int | None = None
    tunnel_proto: str | None = None
    tunnel_path: str | None = None
    tunnel_user: str | None = None
    tunnel_pass: str | None = None
    enabled: bool | None = None
    remark: str | None = None

# ---------- API ----------
@app.post("/api/login")
def login(req: LoginReq):
    if req.username == PANEL_USER and req.password == PANEL_PASS:
        audit("login", req.username)
        return {"token": create_token(req.username)}
    raise HTTPException(401, "用户名或密码错误")

@app.get("/api/status")
def status_api(_=Depends(require_auth)):
    # gost 进程状态
    proc = subprocess.run(["systemctl", "is-active", "gost"],
                          capture_output=True, text=True).stdout.strip()
    # gost 当前配置
    code, cfg = gost_request("GET", "/config")
    service_count = len(cfg.get("services", [])) if cfg else 0
    return {
        "gost": {"active": proc == "active", "service_count": service_count,
                 "api": "up" if code == 200 else "down"},
        "panel": {"version": "0.1.0", "time": datetime.now().isoformat(timespec="seconds")},
    }

def client_cmd_for(rule: dict) -> str:
    """生成客户端连接命令（入站隧道场景：用户在自己电脑上运行）"""
    if rule.get("type") != "tunnel":
        return ""
    proto = rule.get("tunnel_proto", "ws")
    auth = ""
    if rule.get("tunnel_user"):
        auth = f"{rule['tunnel_user']}:{rule.get('tunnel_pass', '')}@"
    path = f"?path={rule['tunnel_path']}" if rule.get("tunnel_path") else ""
    # 目标地址默认为本机公网 IP 的监听端口
    return f"gost -L tcp://:1080 -F \"{proto}://{auth}SERVER_IP:{rule['listen_port']}{path}\""

@app.get("/api/rules")
def list_rules(_=Depends(require_auth)):
    with db() as conn:
        rows = conn.execute("SELECT * FROM rules ORDER BY id DESC").fetchall()
    result = []
    for r in rows:
        d = dict(r)
        d["client_cmd"] = client_cmd_for(d)
        result.append(d)
    return result

@app.post("/api/rules")
def create_rule(req: RuleReq, _=Depends(require_auth)):
    # 端口占用检查
    with db() as conn:
        dup = conn.execute("SELECT id FROM rules WHERE listen_port=? AND enabled=1",
                           (req.listen_port,)).fetchone()
    if dup:
        raise HTTPException(400, f"端口 {req.listen_port} 已被占用")
    with db() as conn:
        cur = conn.execute(
            "INSERT INTO rules (name, type, proto, listen_port, target_host, target_port, tunnel_proto, tunnel_path, tunnel_user, tunnel_pass, enabled, remark, created_at) "
            "VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)",
            (req.name, req.type, req.proto, req.listen_port, req.target_host, req.target_port,
             req.tunnel_proto, req.tunnel_path, req.tunnel_user, req.tunnel_pass,
             int(req.enabled), req.remark, datetime.now().isoformat(timespec="seconds")))
        rid = cur.lastrowid
    audit("rule.create", f"#{rid} {req.name} [{req.type}] {req.proto}:{req.listen_port}->{req.target_host}:{req.target_port}")
    if req.enabled:
        apply_to_gost()
    return {"id": rid, "ok": True}

@app.put("/api/rules/{rid}")
def update_rule(rid: int, req: RulePatch, _=Depends(require_auth)):
    with db() as conn:
        row = conn.execute("SELECT * FROM rules WHERE id=?", (rid,)).fetchone()
        if not row:
            raise HTTPException(404, "规则不存在")
        data = dict(row)
        for k, v in req.model_dump(exclude_none=True).items():
            data[k] = int(v) if k in ("listen_port", "target_port", "enabled") else v        # 端口冲突检查（排除自己）
        if req.listen_port is not None:
            dup = conn.execute(
                "SELECT id FROM rules WHERE listen_port=? AND enabled=1 AND id!=?",
                (req.listen_port, rid)).fetchone()
            if dup:
                raise HTTPException(400, f"端口 {req.listen_port} 已被占用")
        conn.execute(
            "UPDATE rules SET name=?, type=?, proto=?, listen_port=?, target_host=?, target_port=?, "
            "tunnel_proto=?, tunnel_path=?, tunnel_user=?, tunnel_pass=?, enabled=?, remark=? WHERE id=?",
            (data["name"], data["type"], data["proto"], data["listen_port"], data["target_host"],
             data["target_port"], data["tunnel_proto"], data["tunnel_path"], data["tunnel_user"],
             data["tunnel_pass"], data["enabled"], data["remark"], rid))
    audit("rule.update", f"#{rid}")
    apply_to_gost()
    return {"ok": True}

@app.delete("/api/rules/{rid}")
def delete_rule(rid: int, _=Depends(require_auth)):
    with db() as conn:
        row = conn.execute("SELECT * FROM rules WHERE id=?", (rid,)).fetchone()
        if not row:
            raise HTTPException(404, "规则不存在")
        conn.execute("DELETE FROM rules WHERE id=?", (rid,))
    audit("rule.delete", f"#{rid} {row['name']}")
    apply_to_gost()
    return {"ok": True}

@app.post("/api/rules/apply")
def apply_rules(_=Depends(require_auth)):
    code, msg = apply_to_gost()
    audit("rule.apply", msg)
    if code != 200:
        raise HTTPException(502, msg)
    return {"ok": True, "message": msg}

@app.get("/api/audit")
def list_audit(limit: int = 50, _=Depends(require_auth)):
    with db() as conn:
        rows = conn.execute("SELECT * FROM audit ORDER BY id DESC LIMIT ?", (limit,)).fetchall()
    return [dict(r) for r in rows]

# 前端静态文件（构建产物）
STATIC_DIR = BASE_DIR / "static"
if STATIC_DIR.exists():
    app.mount("/", StaticFiles(directory=STATIC_DIR, html=True), name="static")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host=LISTEN_HOST, port=LISTEN_PORT, log_level="info")
