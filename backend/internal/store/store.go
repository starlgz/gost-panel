// Package store 提供 xorm ORM 存储层（SQLite，modernc.org/sqlite 纯 Go 驱动，无 CGO）。
package store

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("记录不存在")

// ErrPortInUse 宿主机端口已被系统内其他进程监听。
var ErrPortInUse = errors.New("端口已被其他程序占用")

// Rule 一条转发/隧道规则。
type Rule struct {
	ID          int    `xorm:"pk autoincr 'id'" json:"id"`
	Name        string `xorm:"'name'" json:"name"`
	Type        string `xorm:"'type'" json:"type"` // forward | tunnel
	Proto       string `xorm:"'proto'" json:"proto"`
	ListenPort  int    `xorm:"'listen_port'" json:"listen_port"`
	TargetHost  string `xorm:"'target_host'" json:"target_host"`
	TargetPort  int    `xorm:"'target_port'" json:"target_port"`
	TunnelProto string `xorm:"'tunnel_proto'" json:"tunnel_proto"`
	TunnelPath  string `xorm:"'tunnel_path'" json:"tunnel_path"`
	TunnelUser  string `xorm:"'tunnel_user'" json:"tunnel_user"`
	TunnelPass  string `xorm:"'tunnel_pass'" json:"tunnel_pass"`
	Enabled     int    `xorm:"'enabled'" json:"enabled"`
	Remark      string `xorm:"'remark'" json:"remark"`
	GostName    string `xorm:"'gost_name'" json:"gost_name"` // 对应的 gost 服务名（svc-N），导入时保留
	CreatedAt   string `xorm:"'created_at'" json:"created_at"`
}

// TableName 固定表名为 rules（与历史版本数据库保持一致）。
func (Rule) TableName() string { return "rules" }

// AuditEntry 一条审计记录。
type AuditEntry struct {
	ID     int    `xorm:"pk autoincr 'id'" json:"id"`
	TS     string `xorm:"'ts'" json:"ts"`
	Action string `xorm:"'action'" json:"action"`
	Detail string `xorm:"'detail'" json:"detail"`
}

// Setting 键值对设置（面板用户名密码等）。
type Setting struct {
	Key   string `xorm:"pk 'key'" json:"key"`
	Value string `xorm:"'value'" json:"value"`
}

// Store 封装 xorm 引擎。
type Store struct {
	engine *xorm.Engine
}

// Open 打开（必要时创建）数据库并同步表结构。
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	// journal_mode=WAL + busy_timeout，避免并发写锁冲突
	engine, err := xorm.NewEngine("sqlite",
		fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath))
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	engine.SetMaxOpenConns(1)
	// 同步表结构：表不存在则建表，存在则补缺失列
	if err := engine.Sync(new(Rule), new(AuditEntry), new(Setting)); err != nil {
		engine.Close()
		return nil, fmt.Errorf("同步表结构失败: %w", err)
	}
	// 补列后旧行的新列为 NULL，回填默认值（旧库升级兼容）
	for _, m := range [][2]string{
		{"type", "'forward'"},
		{"tunnel_proto", "'ws'"},
		{"tunnel_path", "''"},
		{"tunnel_user", "''"},
		{"tunnel_pass", "''"},
	} {
		if _, err := engine.Exec(fmt.Sprintf("UPDATE rules SET %s=%s WHERE %s IS NULL", m[0], m[1], m[0])); err != nil {
			engine.Close()
			return nil, fmt.Errorf("回填 %s 失败: %w", m[0], err)
		}
	}
	return &Store{engine: engine}, nil
}

// Close 关闭数据库。
func (s *Store) Close() error { return s.engine.Close() }

// ListRules 返回全部规则（按 id 倒序）。
func (s *Store) ListRules() ([]Rule, error) {
	var out []Rule
	err := s.engine.Desc("id").Find(&out)
	return out, err
}

// RuleQuery 规则分页查询条件。
type RuleQuery struct {
	Page     int    // 从 1 开始；<1 视为 1
	PageSize int    // <=0 表示不分页（返回全部）
	Keyword  string // 模糊匹配 name/remark/target_host/listen_port/target_port
}

// ListRulesPaged 按条件分页查询规则，返回当前页数据与过滤后总条数。
func (s *Store) ListRulesPaged(q RuleQuery) ([]Rule, int64, error) {
	// Count 会关闭 engine 创建的自动会话，筛选和排序不能在 Find 中复用。
	countSession := s.ruleQuery(q.Keyword)
	defer countSession.Close()
	total, err := countSession.Count(new(Rule))
	if err != nil {
		return nil, 0, err
	}
	sess := s.ruleQuery(q.Keyword)
	defer sess.Close()
	sess.Desc("id")
	if q.PageSize > 0 {
		page := q.Page
		if page < 1 {
			page = 1
		}
		sess = sess.Limit(q.PageSize, (page-1)*q.PageSize)
	}
	out := make([]Rule, 0)
	if err := sess.Find(&out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *Store) ruleQuery(keyword string) *xorm.Session {
	sess := s.engine.NewSession()
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		sess.Where(
			"name LIKE ? OR remark LIKE ? OR target_host LIKE ? OR CAST(listen_port AS TEXT) LIKE ? OR CAST(target_port AS TEXT) LIKE ?",
			like, like, like, like, like,
		)
	}
	return sess
}

// RuleStats 在数据库聚合统计，规则列表无需额外读取全量数据。
type RuleStats struct {
	Total   int64 `json:"total"`
	Enabled int64 `json:"enabled"`
	TCP     int64 `json:"tcp" xorm:"'tcp'"`
	UDP     int64 `json:"udp" xorm:"'udp'"`
	Tunnel  int64 `json:"tunnel"`
}

func (s *Store) GetRuleStats() (RuleStats, error) {
	var stats RuleStats
	_, err := s.engine.SQL(`SELECT COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN enabled = 1 THEN 1 ELSE 0 END), 0) AS enabled,
		COALESCE(SUM(CASE WHEN type != 'tunnel' AND proto = 'tcp' THEN 1 ELSE 0 END), 0) AS tcp,
		COALESCE(SUM(CASE WHEN type != 'tunnel' AND proto = 'udp' THEN 1 ELSE 0 END), 0) AS udp,
		COALESCE(SUM(CASE WHEN type = 'tunnel' THEN 1 ELSE 0 END), 0) AS tunnel
		FROM rules`).Get(&stats)
	return stats, err
}

// GetRule 按 ID 取单条规则。
func (s *Store) GetRule(id int) (Rule, error) {
	var r Rule
	has, err := s.engine.ID(id).Get(&r)
	if err != nil {
		return r, err
	}
	if !has {
		return r, ErrNotFound
	}
	return r, nil
}

// PortBusy 检查端口是否已被其他启用规则占用（excludeID<0 时不排除）。
func (s *Store) PortBusy(port, excludeID int) (bool, error) {
	q := s.engine.Where("listen_port=? AND enabled=1", port)
	if excludeID >= 0 {
		q = q.And("id<>?", excludeID)
	}
	return q.Exist(new(Rule))
}

// CheckHostPort 试绑定宿主机端口，检测是否被系统内其他进程（sshd/nginx 等）占用。
// 绑定成功立即关闭，视为空闲；占用返回 ErrPortInUse。
//
// Windows 上普通 bind 会与未设 SO_EXCLUSIVEADDRUSE 的 socket 静默共存
// （试绑成功但实际不可用），因此 Windows 路径必须以 SO_EXCLUSIVEADDRUSE 绑定。
func CheckHostPort(proto string, port int) error {
	network := "tcp"
	if proto == "udp" {
		network = "udp"
	}
	addr := &net.UDPAddr{Port: port}
	var err error
	if network == "udp" {
		var pc *net.UDPConn
		pc, err = listenUDPExclusive(addr)
		if pc != nil {
			pc.Close()
		}
	} else {
		var ln net.Listener
		ln, err = listenTCPExclusive(port)
		if ln != nil {
			ln.Close()
		}
	}
	if err != nil {
		if isAddrInUse(err) {
			return ErrPortInUse
		}
		return err
	}
	return nil
}

// isAddrInUse 跨平台识别"地址已被占用"类错误。
func isAddrInUse(err error) bool {
	var oe *net.OpError
	if errors.As(err, &oe) {
		msg := oe.Err.Error()
		return errors.Is(oe.Err, os.ErrPermission) ||
			strings.Contains(msg, "address already in use") || // Linux EADDRINUSE
			strings.Contains(msg, "Only one usage of each socket address") || // Windows WSAEADDRINUSE
			strings.Contains(msg, "An attempt was made to access a socket") // Windows WSAEACCES
	}
	return false
}

func fmtAddr(port int) string { return fmt.Sprintf(":%d", port) }

// CreateRule 新建规则，返回新 ID。
func (s *Store) CreateRule(r Rule) (int, error) {
	r.CreatedAt = time.Now().Format("2006-01-02T15:04:05")
	_, err := s.engine.Insert(&r)
	return r.ID, err
}

// UpdateRule 全量更新规则字段。
func (s *Store) UpdateRule(r Rule) error {
	_, err := s.engine.ID(r.ID).AllCols().Update(&r)
	return err
}

// DeleteRule 删除规则。
func (s *Store) DeleteRule(id int) error {
	_, err := s.engine.ID(id).Delete(new(Rule))
	return err
}

// AddAudit 写入一条审计记录。
func (s *Store) AddAudit(action, detail string) {
	_, _ = s.engine.Insert(&AuditEntry{
		TS:     time.Now().Format("2006-01-02T15:04:05"),
		Action: action,
		Detail: detail,
	})
}

// ListAudit 返回最近 limit 条审计记录。
func (s *Store) ListAudit(limit int) ([]AuditEntry, error) {
	var out []AuditEntry
	err := s.engine.Desc("id").Limit(limit).Find(&out)
	return out, err
}

// ListAuditPaged 分页返回审计记录，返回当前页数据与总条数。
func (s *Store) ListAuditPaged(page, pageSize int, keyword string) ([]AuditEntry, int64, error) {
	query := func() *xorm.Session {
		sess := s.engine.NewSession()
		if kw := strings.TrimSpace(keyword); kw != "" {
			like := "%" + kw + "%"
			sess.Where("action LIKE ? OR detail LIKE ? OR ts LIKE ?", like, like, like)
		}
		return sess
	}
	countSession := query()
	defer countSession.Close()
	total, err := countSession.Count(new(AuditEntry))
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	out := make([]AuditEntry, 0)
	sess := query()
	defer sess.Close()
	err = sess.Desc("id").Limit(pageSize, (page-1)*pageSize).Find(&out)
	return out, total, err
}

// ClearAudit 清空全部审计记录，返回删除条数。
func (s *Store) ClearAudit() (int64, error) {
	res, err := s.engine.Where("1 = 1").Delete(new(AuditEntry))
	return res, err
}

// GetSetting 读取键值设置，不存在返回空字符串。
func (s *Store) GetSetting(key string) (string, error) {
	s2 := new(Setting)
	has, err := s.engine.Where("`key` = ?", key).Get(s2)
	if err != nil {
		return "", err
	}
	if !has {
		return "", nil
	}
	return s2.Value, nil
}

// SetSetting 写入键值设置（upsert）。
func (s *Store) SetSetting(key, value string) error {
	_, err := s.engine.Insert(&Setting{Key: key, Value: value})
	if err == nil {
		return nil
	}
	// 主键冲突则更新
	_, err = s.engine.Where("`key` = ?", key).Update(&Setting{Value: value})
	return err
}
