// Package api 注册全部 HTTP 路由与处理函数。
// 响应统一走 internal/resp：{"code":0,"message":"ok","data":...}
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gost-panel/internal/auth"
	"gost-panel/internal/config"
	"gost-panel/internal/gostapi"
	"gost-panel/internal/resp"
	"gost-panel/internal/store"
	"gost-panel/internal/webui"
)

// Version 面板版本号。构建时可通过
// -ldflags "-X gost-panel/internal/api.Version=v1.2.3" 覆盖。
var Version = "0.1.0"

// Server 组装依赖并持有路由引擎。
type Server struct {
	cfg    *config.Config
	st     *store.Store
	gost   *gostapi.Client
	jwtMgr *auth.Manager
	router *gin.Engine
}

// NewServer 创建并注册全部路由。
func NewServer(cfg *config.Config, st *store.Store, gost *gostapi.Client, jwtMgr *auth.Manager) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	s := &Server{cfg: cfg, st: st, gost: gost, jwtMgr: jwtMgr, router: r}
	s.registerRoutes()
	return s
}

// Run 启动 HTTP 监听。
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.ListenHost, s.cfg.ListenPort)
	return s.router.Run(addr)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ---------- 请求模型 ----------

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ruleReq struct {
	Name        string `json:"name" binding:"required,max=64"`
	Type        string `json:"type" binding:"omitempty,oneof=forward tunnel"`
	Proto       string `json:"proto" binding:"omitempty,oneof=tcp udp"`
	ListenPort  int    `json:"listen_port" binding:"required,min=1,max=65535"`
	TargetHost  string `json:"target_host" binding:"required,min=1"`
	TargetPort  int    `json:"target_port" binding:"required,min=1,max=65535"`
	TunnelProto string `json:"tunnel_proto" binding:"omitempty,oneof=ws wss grpc quic http2"`
	TunnelPath  string `json:"tunnel_path"`
	TunnelUser  string `json:"tunnel_user"`
	TunnelPass  string `json:"tunnel_pass"`
	Enabled     *bool  `json:"enabled"`
	Remark      string `json:"remark"`
}

type rulePatch struct {
	Name        *string `json:"name"`
	Type        *string `json:"type" binding:"omitempty,oneof=forward tunnel"`
	Proto       *string `json:"proto" binding:"omitempty,oneof=tcp udp"`
	ListenPort  *int    `json:"listen_port" binding:"omitempty,min=1,max=65535"`
	TargetHost  *string `json:"target_host"`
	TargetPort  *int    `json:"target_port" binding:"omitempty,min=1,max=65535"`
	TunnelProto *string `json:"tunnel_proto" binding:"omitempty,oneof=ws wss grpc quic http2"`
	TunnelPath  *string `json:"tunnel_path"`
	TunnelUser  *string `json:"tunnel_user"`
	TunnelPass  *string `json:"tunnel_pass"`
	Enabled     *bool   `json:"enabled"`
	Remark      *string `json:"remark"`
}

// ---------- 路由注册 ----------

func (s *Server) registerRoutes() {
	api := s.router.Group("/api")

	api.POST("/login", s.handleLogin)

	authGroup := api.Group("", s.jwtMgr.Middleware())
	authGroup.GET("/status", s.handleStatus)
	authGroup.GET("/rules", s.handleListRules)
	authGroup.GET("/rules/stats", s.handleRuleStats)
	authGroup.POST("/rules/apply", s.handleApply)
	authGroup.GET("/gost/services", s.handleGostServices)
	authGroup.POST("/gost/import", s.handleImportGost)
	authGroup.PUT("/rules/:rid", s.handleUpdateRule)
	authGroup.DELETE("/rules/:rid", s.handleDeleteRule)
	authGroup.POST("/rules", s.handleCreateRule)
	authGroup.GET("/audit", s.handleListAudit)
	authGroup.DELETE("/audit", s.handleClearAudit)
	authGroup.POST("/change-password", s.handleChangePassword)

	webui.Register(s.router, filepath.Join(s.cfg.BaseDir, "static"))
}

// ---------- 处理函数 ----------

// POST /api/login
func (s *Server) handleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrArgs(c, "参数错误")
		return
	}
	uOk := subtle.ConstantTimeCompare([]byte(req.Username), []byte(s.cfg.PanelUser)) == 1
	pOk := subtle.ConstantTimeCompare([]byte(req.Password), []byte(s.cfg.PanelPass)) == 1
	if !(uOk && pOk) {
		resp.ErrUnauthorized(c, "用户名或密码错误")
		return
	}
	s.st.AddAudit("login", req.Username)
	tok, err := s.jwtMgr.CreateToken(req.Username)
	if err != nil {
		resp.ErrInternal(c, "生成 token 失败")
		return
	}
	resp.OK(c, gin.H{"token": tok})
}

// GET /api/status
func (s *Server) handleStatus(c *gin.Context) {
	active := false
	if out, err := exec.Command("systemctl", "is-active", "gost").Output(); err == nil {
		active = strings.TrimSpace(string(out)) == "active"
	}
	count, up := s.gost.ServiceCount()
	apiState := "down"
	if up {
		apiState = "up"
	}
	resp.OK(c, gin.H{
		"gost": gin.H{
			"active":        active,
			"service_count": count,
			"api":           apiState,
		},
		"panel": gin.H{
			"version": Version,
			"time":    time.Now().Format("2006-01-02T15:04:05"),
		},
	})
}

// GET /api/rules
// 不带分页和搜索参数：返回全量数组（总览页聚合用）；
// 带 page/page_size/keyword 任一参数：返回 {items,total,page,page_size}。
func (s *Server) handleListRules(c *gin.Context) {
	page, pageSize, paged := parsePagination(c)
	if !paged {
		rules, err := s.st.ListRules()
		if err != nil {
			resp.ErrInternal(c, "读取规则失败")
			return
		}
		out := make([]gin.H, 0, len(rules))
		for _, r := range rules {
			out = append(out, ruleToH(r, gostapi.ClientCmd(r)))
		}
		resp.OK(c, out)
		return
	}

	rules, total, err := s.st.ListRulesPaged(store.RuleQuery{
		Page: page, PageSize: pageSize, Keyword: c.Query("keyword"),
	})
	if err != nil {
		resp.ErrInternal(c, "读取规则失败")
		return
	}
	items := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		items = append(items, ruleToH(r, gostapi.ClientCmd(r)))
	}
	resp.OK(c, gin.H{
		"items": items, "total": total, "page": page, "page_size": pageSize,
	})
}

// GET /api/rules/stats 返回所有规则的聚合统计。
func (s *Server) handleRuleStats(c *gin.Context) {
	stats, err := s.st.GetRuleStats()
	if err != nil {
		resp.ErrInternal(c, "读取规则统计失败")
		return
	}
	resp.OK(c, stats)
}

// POST /api/rules
func (s *Server) handleCreateRule(c *gin.Context) {
	var req ruleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrArgs(c, "参数错误: "+err.Error())
		return
	}
	applyDefaults(&req)
	enabled := 1
	if req.Enabled != nil && !*req.Enabled {
		enabled = 0
	}
	// 规则互撞检查
	busy, err := s.st.PortBusy(req.ListenPort, -1)
	if err != nil {
		resp.ErrInternal(c, "查询端口失败")
		return
	}
	if busy {
		resp.ErrBadRequest(c, fmt.Sprintf("端口 %d 已被面板规则占用", req.ListenPort))
		return
	}
	// 宿主机端口检查（仅启用中的规则需要真正监听）
	if enabled == 1 {
		if err := store.CheckHostPort(req.Proto, req.ListenPort); err != nil {
			resp.ErrBadRequest(c, fmt.Sprintf("端口 %d %s", req.ListenPort, err.Error()))
			return
		}
	}
	rid, err := s.st.CreateRule(store.Rule{
		Name: req.Name, Type: req.Type, Proto: req.Proto,
		ListenPort: req.ListenPort, TargetHost: req.TargetHost, TargetPort: req.TargetPort,
		TunnelProto: req.TunnelProto, TunnelPath: req.TunnelPath,
		TunnelUser: req.TunnelUser, TunnelPass: req.TunnelPass,
		Enabled: enabled, Remark: req.Remark,
	})
	if err != nil {
		resp.ErrInternal(c, "写入失败")
		return
	}
	s.st.AddAudit("rule.create", fmt.Sprintf("#%d %s [%s] %s:%d->%s:%d",
		rid, req.Name, req.Type, req.Proto, req.ListenPort, req.TargetHost, req.TargetPort))
	if enabled == 1 {
		s.gost.Apply(s.st)
	}
	resp.OK(c, gin.H{"id": rid})
}

// PUT /api/rules/:rid
func (s *Server) handleUpdateRule(c *gin.Context) {
	rid, ok := parseID(c)
	if !ok {
		return
	}
	cur, err := s.st.GetRule(rid)
	if err != nil {
		getRuleErr(c, err)
		return
	}
	var p rulePatch
	if err := c.ShouldBindJSON(&p); err != nil {
		resp.ErrArgs(c, "参数错误: "+err.Error())
		return
	}
	beforePort := cur.ListenPort
	beforeEnabled := cur.Enabled
	applyPatch(&cur, &p)
	// 端口变化，或从停用切到启用时才需要检查
	portChanged := p.ListenPort != nil && *p.ListenPort != beforePort
	enabledTurnedOn := p.Enabled != nil && *p.Enabled && beforeEnabled == 0
	if portChanged || enabledTurnedOn {
		// 规则互撞检查
		busy, err := s.st.PortBusy(cur.ListenPort, rid)
		if err != nil {
			resp.ErrInternal(c, "查询端口失败")
			return
		}
		if busy {
			resp.ErrBadRequest(c, fmt.Sprintf("端口 %d 已被面板规则占用", cur.ListenPort))
			return
		}
		// 宿主机端口检查：仅启用中的规则需要真正监听
		if cur.Enabled == 1 {
			if err := store.CheckHostPort(cur.Proto, cur.ListenPort); err != nil {
				resp.ErrBadRequest(c, fmt.Sprintf("端口 %d %s", cur.ListenPort, err.Error()))
				return
			}
		}
	}
	if err := s.st.UpdateRule(cur); err != nil {
		resp.ErrInternal(c, "更新失败")
		return
	}
	s.st.AddAudit("rule.update", fmt.Sprintf("#%d", rid))
	s.gost.Apply(s.st)
	resp.OK(c, nil)
}

// DELETE /api/rules/:rid
func (s *Server) handleDeleteRule(c *gin.Context) {
	rid, ok := parseID(c)
	if !ok {
		return
	}
	cur, err := s.st.GetRule(rid)
	if err != nil {
		getRuleErr(c, err)
		return
	}
	if err := s.st.DeleteRule(rid); err != nil {
		resp.ErrInternal(c, "删除失败")
		return
	}
	s.st.AddAudit("rule.delete", fmt.Sprintf("#%d %s", rid, cur.Name))
	s.gost.Apply(s.st)
	resp.OK(c, nil)
}

// POST /api/rules/apply
func (s *Server) handleApply(c *gin.Context) {
	code, msg := s.gost.Apply(s.st)
	s.st.AddAudit("rule.apply", msg)
	if code != 200 {
		resp.ErrUpstream(c, msg)
		return
	}
	resp.OKMsg(c, msg, nil)
}

// GET /api/gost/services 从 gost 拉取当前运行的服务列表，用于面板比对。
func (s *Server) handleGostServices(c *gin.Context) {
	code, raw := s.gost.PullServices()
	if code != 200 {
		resp.ErrUpstream(c, fmt.Sprintf("gost API 查询失败 (%d)", code))
		return
	}
	// gost 返回 {"data":{"list":[...]}}，直接透传给前端展示。
	if raw == nil {
		resp.OK(c, []any{})
		return
	}
	var wrapped struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Data.List) > 0 {
		resp.OK(c, wrapped.Data.List)
		return
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err == nil {
		resp.OK(c, list)
		return
	}
	resp.OK(c, []any{})
}

// POST /api/gost/import 把 gost 当前运行的服务导入为面板规则（跳过已存在的端口）。
func (s *Server) handleImportGost(c *gin.Context) {
	code, raw := s.gost.PullServices()
	if code != 200 {
		resp.ErrUpstream(c, fmt.Sprintf("gost API 查询失败 (%d)", code))
		return
	}
	if raw == nil {
		resp.OKMsg(c, "gost 当前没有服务", gin.H{"imported": 0, "skipped": 0})
		return
	}

	var wrapped struct {
		Data struct {
			List []map[string]any `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		resp.ErrInternal(c, "解析 gost 服务列表失败")
		return
	}

	imported := 0
	skipped := 0
	for _, svc := range wrapped.Data.List {
		// 解析监听端口 :50100
		addr, _ := svc["addr"].(string)
		listenPort := 0
		if _, err := fmt.Sscanf(addr, ":%d", &listenPort); err != nil || listenPort < 1 {
			skipped++
			continue
		}

		// 解析目标节点
		nodes, _ := svc["forwarder"].(map[string]any)["nodes"].([]any)
		if len(nodes) == 0 {
			skipped++
			continue
		}
		node, _ := nodes[0].(map[string]any)
		targetAddr, _ := node["addr"].(string)
		host, port := splitHostPort(targetAddr)
		if host == "" || port < 1 {
			skipped++
			continue
		}

		// 跳过端口已被面板规则占用的
		busy, err := s.st.PortBusy(listenPort, -1)
		if err != nil || busy {
			skipped++
			continue
		}

		svcName, _ := svc["name"].(string)
		proto := "tcp"
		if listener, ok := svc["listener"].(map[string]any); ok {
			if lt, ok := listener["type"].(string); ok && lt == "udp" {
				proto = "udp"
			}
		}

		rid, err := s.st.CreateRule(store.Rule{
			Name:       fmt.Sprintf("导入-%s", svcName),
			Type:       "forward",
			Proto:      proto,
			ListenPort: listenPort,
			TargetHost: host,
			TargetPort: port,
			Enabled:    1,
			Remark:     "从 gost 导入",
			GostName:   svcName, // 保留原 gost 服务名，Apply 时不会删错
		})
		if err != nil {
			skipped++
			continue
		}
		s.st.AddAudit("rule.import", fmt.Sprintf("#%d %s :%d->%s:%d", rid, svcName, listenPort, host, port))
		imported++
	}

	s.st.AddAudit("gost.import", fmt.Sprintf("导入 %d 条，跳过 %d 条", imported, skipped))
	resp.OKMsg(c, fmt.Sprintf("已导入 %d 条规则，跳过 %d 条（端口已占用）", imported, skipped), gin.H{"imported": imported, "skipped": skipped})
}

// splitHostPort 把 "1.2.3.4:8080" 拆成 host 和 port。
func splitHostPort(addr string) (string, int) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return "", 0
	}
	host := addr[:idx]
	port, err := strconv.Atoi(addr[idx+1:])
	if err != nil {
		return "", 0
	}
	return host, port
}

// GET /api/audit?limit=50
func (s *Server) handleListAudit(c *gin.Context) {
	// 分页查询之外，兼容总览页的旧 limit 数组模式。
	page, pageSize, paged := parsePagination(c)
	if paged {
		entries, total, err := s.st.ListAuditPaged(page, pageSize, c.Query("keyword"))
		if err != nil {
			resp.ErrInternal(c, "读取审计失败")
			return
		}
		resp.OK(c, gin.H{"items": entries, "total": total, "page": page, "page_size": pageSize})
		return
	}
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := s.st.ListAudit(limit)
	if err != nil {
		resp.ErrInternal(c, "读取审计失败")
		return
	}
	resp.OK(c, entries)
}

// DELETE /api/audit 清空全部审计记录
func (s *Server) handleClearAudit(c *gin.Context) {
	n, err := s.st.ClearAudit()
	if err != nil {
		resp.ErrInternal(c, "清空审计失败")
		return
	}
	s.st.AddAudit("audit.clear", fmt.Sprintf("清空 %d 条审计记录", n))
	resp.OK(c, gin.H{"deleted": n})
}

// POST /api/change-password 修改面板登录密码
func (s *Server) handleChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrArgs(c, "新密码长度需 6-64 位")
		return
	}
	// 验证旧密码
	if subtle.ConstantTimeCompare([]byte(req.OldPassword), []byte(s.cfg.PanelPass)) != 1 {
		resp.ErrUnauthorized(c, "旧密码不正确")
		return
	}
	if err := s.st.SetSetting("panel_pass", req.NewPassword); err != nil {
		resp.ErrInternal(c, "保存密码失败")
		return
	}
	s.cfg.PanelPass = req.NewPassword
	s.st.AddAudit("panel.passwd", "修改面板登录密码")
	resp.OKMsg(c, "密码已修改，请重新登录", nil)
}

// ---------- 辅助 ----------

// parsePagination 统一分页参数；显式传入空参数也进入分页模式。
func parsePagination(c *gin.Context) (page, pageSize int, paged bool) {
	query := c.Request.URL.Query()
	paged = query.Has("page") || query.Has("page_size") || query.Has("keyword")
	page, _ = strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(query.Get("page_size"))
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 200 {
		pageSize = 200
	}
	// 防止极端页码导致 OFFSET 整数溢出。
	if maxPage := int(^uint(0)>>1) / pageSize; page > maxPage {
		page = maxPage
	}
	return
}

// getRuleErr 区分"不存在"与"查询失败"两种错误。
func getRuleErr(c *gin.Context, err error) {
	if errors.Is(err, store.ErrNotFound) {
		resp.ErrNotFoundResp(c, "规则不存在")
		return
	}
	resp.ErrInternal(c, "查询规则失败")
}

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("rid"))
	if err != nil || id <= 0 {
		resp.ErrBadRequest(c, "无效 ID")
		return 0, false
	}
	return id, true
}

func applyDefaults(req *ruleReq) {
	if req.Type == "" {
		req.Type = "forward"
	}
	if req.Proto == "" {
		req.Proto = "tcp"
	}
	if req.TunnelProto == "" {
		req.TunnelProto = "ws"
	}
}

// applyPatch 把 PATCH 请求中的非 nil 字段合并到规则上。
func applyPatch(cur *store.Rule, p *rulePatch) {
	if p.Name != nil {
		cur.Name = *p.Name
	}
	if p.Type != nil {
		cur.Type = *p.Type
	}
	if p.Proto != nil {
		cur.Proto = *p.Proto
	}
	if p.ListenPort != nil {
		cur.ListenPort = *p.ListenPort
	}
	if p.TargetHost != nil {
		cur.TargetHost = *p.TargetHost
	}
	if p.TargetPort != nil {
		cur.TargetPort = *p.TargetPort
	}
	if p.TunnelProto != nil {
		cur.TunnelProto = *p.TunnelProto
	}
	if p.TunnelPath != nil {
		cur.TunnelPath = *p.TunnelPath
	}
	if p.TunnelUser != nil {
		cur.TunnelUser = *p.TunnelUser
	}
	if p.TunnelPass != nil {
		cur.TunnelPass = *p.TunnelPass
	}
	if p.Enabled != nil {
		if *p.Enabled {
			cur.Enabled = 1
		} else {
			cur.Enabled = 0
		}
	}
	if p.Remark != nil {
		cur.Remark = *p.Remark
	}
}

func ruleToH(r store.Rule, clientCmd string) gin.H {
	h := gin.H{
		"id": r.ID, "name": r.Name, "type": r.Type, "proto": r.Proto,
		"listen_port": r.ListenPort, "target_host": r.TargetHost, "target_port": r.TargetPort,
		"tunnel_proto": r.TunnelProto, "tunnel_path": r.TunnelPath,
		"tunnel_user": r.TunnelUser, "tunnel_pass": r.TunnelPass,
		"enabled": r.Enabled, "remark": r.Remark, "gost_name": r.GostName, "created_at": r.CreatedAt,
	}
	if clientCmd != "" {
		h["client_cmd"] = clientCmd
	}
	return h
}
