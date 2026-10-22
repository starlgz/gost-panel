// Package gostapi 封装 gost v3 Web API 调用与配置编译（热更新 + 持久化）。
package gostapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gost-panel/internal/store"
)

// Client 是 gost Web API 客户端。
type Client struct {
	BaseURL    string
	User, Pass string
	ConfigPath string // 持久化写回的 gost 配置文件路径
	HTTPClient *http.Client
}

// New 创建客户端。
func New(baseURL, user, pass, configPath string) *Client {
	return &Client{
		BaseURL:    baseURL,
		User:       user,
		Pass:       pass,
		ConfigPath: configPath,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Request 调用 gost Web API，返回 (httpCode, body)。
func (c *Client) Request(method, path string, body any) (int, json.RawMessage) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rdr)
	if err != nil {
		return 0, nil
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(c.User+":"+c.Pass)))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) == 0 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.RawMessage(raw)
}

// listNames 从 /config/services、/config/chains 提取名字集合，
// 兼容 {"data":{"list":[...]}} 与裸数组两种返回格式。
func listNames(code int, raw json.RawMessage) map[string]bool {
	names := map[string]bool{}
	if code != 200 || raw == nil {
		return names
	}
	type obj struct {
		Name string `json:"name"`
	}
	var wrapped struct {
		Data struct {
			List []obj `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Data.List) > 0 {
		for _, o := range wrapped.Data.List {
			names[o.Name] = true
		}
		return names
	}
	var list []obj
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, o := range list {
			names[o.Name] = true
		}
	}
	return names
}

func handlerTypeOf(r store.Rule) string {
	if r.Proto == "udp" {
		return "udp"
	}
	return "tcp"
}

// svcName 返回规则对应的 gost 服务名：优先用 GostName（导入时保留原服务名），否则按 ID 生成。
func svcName(r store.Rule) string {
	if r.GostName != "" {
		return r.GostName
	}
	return fmt.Sprintf("svc-%d", r.ID)
}

// chainName 返回规则对应的 gost 链名。
func chainName(r store.Rule) string {
	if r.GostName != "" {
		// svc-N -> chn-N
		return strings.Replace(r.GostName, "svc-", "chn-", 1)
	}
	return fmt.Sprintf("chn-%d", r.ID)
}

// nodeName 返回节点名。
func nodeName(r store.Rule) string {
	if r.GostName != "" {
		return strings.Replace(r.GostName, "svc-", "n-", 1)
	}
	return fmt.Sprintf("n-%d", r.ID)
}

// buildTunnel 构造隧道规则对应的 chain + service。
func buildTunnel(r store.Rule) (chain, svc map[string]any) {
	sname := svcName(r)
	cname := chainName(r)
	nname := nodeName(r)
	node := map[string]any{
		"name":      nname,
		"addr":      fmt.Sprintf("%s:%d", r.TargetHost, r.TargetPort),
		"connector": map[string]any{"type": "tcp"},
		"dialer":    map[string]any{"type": "tcp"},
	}
	chain = map[string]any{
		"name": cname,
		"hops": []any{map[string]any{
			"name":  "hop-" + nname,
			"nodes": []any{node},
		}},
	}
	listener := map[string]any{"type": r.TunnelProto}
	if r.TunnelProto == "ws" && r.TunnelPath != "" {
		listener = map[string]any{"type": "ws", "metadata": map[string]any{"path": r.TunnelPath}}
	}
	handler := map[string]any{"type": "auto", "chain": cname}
	if r.TunnelUser != "" {
		handler["auth"] = map[string]any{"username": r.TunnelUser, "password": r.TunnelPass}
	}
	svc = map[string]any{
		"name":     sname,
		"addr":     fmt.Sprintf(":%d", r.ListenPort),
		"handler":  handler,
		"listener": listener,
	}
	return chain, svc
}

// buildForward 构造普通端口转发 service。
func buildForward(r store.Rule) map[string]any {
	ht := handlerTypeOf(r)
	return map[string]any{
		"name":     svcName(r),
		"addr":     fmt.Sprintf(":%d", r.ListenPort),
		"handler":  map[string]any{"type": ht},
		"listener": map[string]any{"type": ht},
		"forwarder": map[string]any{
			"nodes": []any{map[string]any{
				"name": nodeName(r),
				"addr": fmt.Sprintf("%s:%d", r.TargetHost, r.TargetPort),
			}},
		},
	}
}

// buildFullConfig 生成完整 gost 配置（services + chains + api），用于持久化。
func (c *Client) buildFullConfig(rules []store.Rule) map[string]any {
	// 与面板实际连接的 API 地址一致，避免保存规则后把本机 API 改成公网监听。
	apiAddr := "127.0.0.1:18080"
	if endpoint, err := url.Parse(c.BaseURL); err == nil && endpoint.Host != "" {
		apiAddr = endpoint.Host
	}
	services := []any{}
	chains := []any{}
	for _, r := range rules {
		if r.Enabled == 0 {
			continue
		}
		if r.Type == "tunnel" {
			ch, svc := buildTunnel(r)
			chains = append(chains, ch)
			services = append(services, svc)
		} else {
			services = append(services, buildForward(r))
		}
	}
	return map[string]any{
		"api": map[string]any{
			"addr": apiAddr,
			"auth": map[string]any{"username": c.User, "password": c.Pass},
		},
		"services": services,
		"chains":   chains,
	}
}

// Apply 把数据库规则同步到 gost（对象级热更新，不断流），并持久化配置文件。
func (c *Client) Apply(st *store.Store) (int, string) {
	rules, err := st.ListRules()
	if err != nil {
		return 500, "读取规则失败: " + err.Error()
	}

	code, raw := c.Request("GET", "/config/services", nil)
	if code != 200 {
		return code, fmt.Sprintf("gost API 查询服务失败 (%d)", code)
	}
	existingSvc := listNames(code, raw)
	code, raw = c.Request("GET", "/config/chains", nil)
	existingChain := listNames(code, raw)

	wantedSvc := map[string]map[string]any{}
	wantedChain := map[string]map[string]any{}
	for _, r := range rules {
		if r.Enabled == 0 {
			continue
		}
		sname := svcName(r)
		cname := chainName(r)
		if r.Type == "tunnel" {
			ch, svc := buildTunnel(r)
			wantedChain[cname] = ch
			wantedSvc[sname] = svc
		} else {
			wantedSvc[sname] = buildForward(r)
		}
	}

	// 删除多余服务/链
	for name := range existingSvc {
		if _, ok := wantedSvc[name]; !ok {
			c.Request("DELETE", "/config/services/"+name, nil)
		}
	}
	for name := range existingChain {
		if _, ok := wantedChain[name]; !ok {
			c.Request("DELETE", "/config/chains/"+name, nil)
		}
	}
	// 新增/更新链与服务
	for name, ch := range wantedChain {
		if existingChain[name] {
			c.Request("PUT", "/config/chains/"+name, ch)
		} else {
			c.Request("POST", "/config/chains", ch)
		}
	}
	for name, svc := range wantedSvc {
		if existingSvc[name] {
			c.Request("PUT", "/config/services/"+name, svc)
		} else {
			c.Request("POST", "/config/services", svc)
		}
	}

	// 持久化：写回 gost 配置文件，防止 gost 重启后丢失
	c.persist(rules)
	return 200, fmt.Sprintf("已同步 %d 条规则", len(wantedSvc))
}

func (c *Client) persist(rules []store.Rule) {
	b, err := json.MarshalIndent(c.buildFullConfig(rules), "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.ConfigPath, b, 0o600)
}

// ServiceCount 查询 gost 当前 service 数量（status 页用）。
func (c *Client) ServiceCount() (int, bool) {
	code, raw := c.Request("GET", "/config", nil)
	if code != 200 || raw == nil {
		return 0, false
	}
	var cfg struct {
		Services []any `json:"services"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return 0, false
	}
	return len(cfg.Services), true
}

// PullServices 从 gost 拉取当前运行的服务列表，返回原始 JSON。
// 用于面板比对：让用户看到 gost 实际在跑什么，和数据库规则是否一致。
func (c *Client) PullServices() (int, json.RawMessage) {
	return c.Request("GET", "/config/services", nil)
}

// ClientCmd 生成客户端连接命令（入站隧道场景，用户在自己电脑上运行）。
func ClientCmd(r store.Rule) string {
	if r.Type != "tunnel" {
		return ""
	}
	auth := ""
	if r.TunnelUser != "" {
		auth = fmt.Sprintf("%s:%s@", r.TunnelUser, r.TunnelPass)
	}
	path := ""
	if r.TunnelPath != "" {
		path = "?path=" + r.TunnelPath
	}
	return fmt.Sprintf("gost -L tcp://:1080 -F \"%s://%sSERVER_IP:%d%s\"",
		r.TunnelProto, auth, r.ListenPort, path)
}
