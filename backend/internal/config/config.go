// Package config 汇总所有配置项，环境变量优先，其余用默认值。
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Config 持有面板全部运行配置。
type Config struct {
	BaseDir        string // 工作目录（data、static 所在）
	DataDir        string // 数据目录（SQLite、jwt_secret）
	DBPath         string // SQLite 文件路径
	GostURL        string // gost Web API 地址
	GostUser       string // gost API 用户名
	GostPass       string // gost API 密码
	PanelUser      string // 面板登录用户名
	PanelPass      string // 面板登录密码
	JWTExpireHours int    // token 有效期（小时）
	ListenHost     string // 面板监听地址
	ListenPort     int    // 面板监听端口
	GostConfigPath string // 持久化写回的 gost 配置文件路径
}

// Credentials 持久化存储的面板登录凭据（覆盖环境变量默认值）。
type Credentials struct {
	PanelUser string `json:"panel_user"`
	PanelPass string `json:"panel_pass"`
}

// credentialsPath 返回凭据文件路径。
func (c *Config) credentialsPath() string {
	return filepath.Join(c.DataDir, "credentials.json")
}

// LoadCredentials 从文件加载凭据，存在则覆盖 PanelUser/PanelPass。
func (c *Config) LoadCredentials() {
	b, err := os.ReadFile(c.credentialsPath())
	if err != nil {
		return
	}
	var cred Credentials
	if json.Unmarshal(b, &cred) != nil {
		return
	}
	if cred.PanelUser != "" {
		c.PanelUser = cred.PanelUser
	}
	if cred.PanelPass != "" {
		c.PanelPass = cred.PanelPass
	}
}

// SaveCredentials 把当前 PanelUser/PanelPass 写到凭据文件。
func (c *Config) SaveCredentials() error {
	if err := os.MkdirAll(c.DataDir, 0o700); err != nil {
		return err
	}
	cred := Credentials{PanelUser: c.PanelUser, PanelPass: c.PanelPass}
	b, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.credentialsPath(), b, 0o600)
}

// FromEnv 从环境变量构建配置。
func FromEnv() *Config {
	c := &Config{
		BaseDir:        envOr("GOST_PANEL_DIR", "/opt/gost-panel"),
		GostURL:        envOr("GOST_API_URL", "http://127.0.0.1:18080"),
		GostUser:       envOr("GOST_API_USER", "admin"),
		GostPass:       envOr("GOST_API_PASS", "gost-panel-change-me"),
		PanelUser:      envOr("PANEL_USER", "admin"),
		PanelPass:      envOr("PANEL_PASS", "gost-panel-change-me"),
		JWTExpireHours: envIntOr("JWT_EXPIRE_HOURS", 24),
		ListenHost:     envOr("PANEL_HOST", "127.0.0.1"),
		ListenPort:     envIntOr("PANEL_PORT", 19090),
		GostConfigPath: envOr("GOST_CONFIG_PATH", "/opt/gost/config.json"),
	}
	c.DataDir = filepath.Join(c.BaseDir, "data")
	c.DBPath = filepath.Join(c.DataDir, "panel.db")
	// 凭据文件优先级最高：覆盖环境变量默认值。
	c.LoadCredentials()
	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
