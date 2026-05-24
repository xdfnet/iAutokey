// iautokey — 修饰键释放后自动模拟 Enter
// 配合语音输入法使用：按住键说话，松开即确认
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

var version = "dev"

type autoEnterConfig struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
	DelayMs int    `json:"delayMs"`
}

type config struct {
	AutoEnter *autoEnterConfig `json:"autoEnter"`
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "status":
			cmdStatus()
			return
		case "version", "-v", "--version":
			fmt.Println("iautokey", version)
			return
		case "restart":
			cmdRestart()
			return
		case "help", "-h", "--help":
			cmdHelp()
			return
		}
	}

	// 无参数 / 非命令参数 → 启动守护进程
	log.SetFlags(log.Ltime | log.Lshortfile)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.AutoEnter == nil || !cfg.AutoEnter.Enabled || cfg.AutoEnter.Key == "" {
		log.Printf("未启用，退出")
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Exit(0)
	}()

	log.Printf("按键=%s delay=%dms", cfg.AutoEnter.Key, cfg.AutoEnter.DelayMs)
	startAutoEnter(cfg.AutoEnter.Key, cfg.AutoEnter.DelayMs)
}

func cmdStatus() {
	out, err := exec.Command("pgrep", "-f", "iautokey").Output()
	if err == nil && len(out) > 0 {
		fmt.Printf("状态: 运行中 (pid %s", strings.TrimSpace(string(out)))
		// pgrep 可能返回多行，只取第一行
		pid := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		fmt.Printf(")\n日志: ~/.config/iautokey/iautokey.log\n")
		_ = pid
	} else {
		fmt.Println("状态: 未运行")
	}
}

func cmdRestart() {
	home, _ := os.UserHomeDir()
	plist := filepath.Join(home, "Library", "LaunchAgents", "com.user.iautokey.plist")
	execCmd("launchctl", "unload", plist)
	execCmd("launchctl", "load", "-w", plist)
	fmt.Println("iautokey: 已重启")
}

func cmdHelp() {
	fmt.Print(`iautokey ` + version + ` — 修饰键释放后自动模拟 Enter

用法:
  iautokey                   启动守护进程
  iautokey status            服务状态
  iautokey restart           重启服务
  iautokey version          版本号
  iautokey help             帮助

配置: ~/.config/iautokey/config.json
`)
}

func execCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func pidFile() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, ".config", "iautokey", "iautokey.pid")
}

func loadConfig() (*config, error) {
	usr, _ := user.Current()

	// 本项目的独立配置
	ownPath := filepath.Join(usr.HomeDir, ".config", "iautokey", "config.json")
	if data, err := os.ReadFile(ownPath); err == nil {
		var cfg config
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}

	// 兼容旧方案：读取 iSpeak 的配置
	legacyPath := filepath.Join(usr.HomeDir, ".config", "ispeak", "config.json")
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("未找到配置，请创建 ~/.config/iautokey/config.json")
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
