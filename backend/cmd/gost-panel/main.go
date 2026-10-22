// gost-panel — GOST v3 转发面板后端（Gin + 纯 Go SQLite 驱动，无 CGO）
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gost-panel/internal/api"
	"gost-panel/internal/auth"
	"gost-panel/internal/config"
	"gost-panel/internal/gostapi"
	"gost-panel/internal/store"
)

func main() {
	// 子命令：version 打印版本号（构建时由 ldflags 注入）
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("gost-panel %s\n", api.Version)
			return
		}
	}

	// 子命令：set-pass 修改面板用户名密码
	if len(os.Args) > 1 && os.Args[1] == "set-pass" {
		runSetPass(os.Args[2:])
		return
	}

	cfg := config.FromEnv()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer st.Close()

	// 数据库里的凭据优先级最高（覆盖环境变量和文件）
	if u, _ := st.GetSetting("panel_user"); u != "" {
		cfg.PanelUser = u
	}
	if p, _ := st.GetSetting("panel_pass"); p != "" {
		cfg.PanelPass = p
	}

	gost := gostapi.New(cfg.GostURL, cfg.GostUser, cfg.GostPass, cfg.GostConfigPath)

	jwtMgr := &auth.Manager{
		Secret:      auth.LoadSecret(cfg.DataDir),
		ExpireHours: cfg.JWTExpireHours,
	}

	srv := api.NewServer(cfg, st, gost, jwtMgr)
	log.Printf("gost-panel listening on %s:%d", cfg.ListenHost, cfg.ListenPort)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

// runSetPass 处理 `gost-panel set-pass` 子命令。
func runSetPass(args []string) {
	fs := flag.NewFlagSet("set-pass", flag.ExitOnError)
	user := fs.String("user", "", "新的面板用户名（留空则不修改）")
	pass := fs.String("pass", "", "新的面板密码（留空则不修改）")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "用法: gost-panel set-pass [--user 新用户名] [--pass 新密码]")
		fmt.Fprintln(os.Stderr, "示例: gost-panel set-pass --user admin --pass MyNewPass123")
	}
	fs.Parse(args)

	if *user == "" && *pass == "" {
		fs.Usage()
		os.Exit(1)
	}

	cfg := config.FromEnv()
	if *user == "" {
		*user = cfg.PanelUser
	}
	if *pass == "" {
		*pass = cfg.PanelPass
	}

	st, err := store.Open(filepath.Join(cfg.BaseDir, "data", "panel.db"))
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer st.Close()

	if err := st.SetSetting("panel_user", *user); err != nil {
		log.Fatalf("保存用户名失败: %v", err)
	}
	if err := st.SetSetting("panel_pass", *pass); err != nil {
		log.Fatalf("保存密码失败: %v", err)
	}

	fmt.Printf("已更新面板凭据: 用户=%s\n", *user)
	fmt.Println("请重启面板服务生效: systemctl restart gost-panel")
}
