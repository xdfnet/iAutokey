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
	"time"
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
		case "setup":
			cmdSetup()
			return
		case "update":
			cmdUpdate()
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
	log.SetOutput(os.Stdout)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.AutoEnter == nil || !cfg.AutoEnter.Enabled || cfg.AutoEnter.Key == "" {
		log.Printf("未启用，退出")
		return
	}
	if !hasAccessibilityPermission() {
		log.Printf("未授予辅助功能权限。请在 系统设置→隐私与安全性→辅助功能 中允许 iautokey，然后执行 iautokey restart")
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
	plist := plistPath()
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.user.iautokey", os.Getuid())).Run()
	if err := ensurePlist(); err != nil {
		fmt.Fprintf(os.Stderr, "plist 写入失败: %v\n", err)
		os.Exit(1)
	}
	if err := execCmd("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), plist); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap 失败: %v\n", err)
		os.Exit(1)
	}
	if err := execCmd("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/com.user.iautokey", os.Getuid())); err != nil {
		fmt.Fprintf(os.Stderr, "kickstart 失败: %v\n", err)
		os.Exit(1)
	}
	if !hasAccessibilityPermission() {
		fmt.Println("⚠️  尚未授予辅助功能权限，已跳过健康检查。授权后执行: iautokey restart")
		return
	}
	if err := waitForHealth(); err != nil {
		fmt.Fprintln(os.Stderr, "⚠️  服务已重启但进程未启动，请检查日志")
		fmt.Fprintf(os.Stderr, "  ~/.config/iautokey/iautokey_error.log\n")
		printCmd := exec.Command("launchctl", "print", fmt.Sprintf("gui/%d/com.user.iautokey", os.Getuid()))
		printCmd.Stdout = os.Stderr
		printCmd.Stderr = os.Stderr
		_ = printCmd.Run()
		os.Exit(1)
	}
	fmt.Println("✅ iautokey 已重启")
}
func cmdSetup() {
	if err := ensureConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "setup 失败: %v\n", err)
		os.Exit(1)
	}
	if err := ensurePlist(); err != nil {
		fmt.Fprintf(os.Stderr, "plist 写入失败: %v\n", err)
		os.Exit(1)
	}
	plist := plistPath()
	_ = exec.Command("launchctl", "bootout", fmt.Sprintf("gui/%d/com.user.iautokey", os.Getuid())).Run()
	if err := execCmd("launchctl", "bootstrap", fmt.Sprintf("gui/%d", os.Getuid()), plist); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap 失败: %v\n", err)
		os.Exit(1)
	}
	if err := execCmd("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/com.user.iautokey", os.Getuid())); err != nil {
		fmt.Fprintf(os.Stderr, "kickstart 失败: %v\n", err)
		os.Exit(1)
	}
	if !hasAccessibilityPermission() {
		fmt.Println("⚠️  尚未授予辅助功能权限。请先授权，再执行: iautokey restart")
		return
	}

	if err := waitForHealth(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  服务启动中，请检查日志\n")
	} else {
		fmt.Println("✅ iautokey 服务已启动")
	}
}

func cmdUpdate() {
	fmt.Println("正在检查更新...")
	cmd := exec.Command("npm", "install", "-g", "@xdfnet/iautokey")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "npm install 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 二进制已更新")
	if !hasAccessibilityPermission() {
		fmt.Println("⚠️  检测到尚未授予辅助功能权限，正在打开系统设置...")
		openAccessibilitySettings()
		fmt.Println("请在系统设置中允许 iautokey，已为你等待授权（最多 60 秒）...")
		if waitForAccessibilityPermission(60 * time.Second) {
			fmt.Println("✅ 已检测到权限授权，正在自动重启服务...")
		} else {
			fmt.Println("⚠️  未检测到授权完成。你授权后执行一次: iautokey restart")
			return
		}
	}
	cmdRestart()
}

func ensureConfig() error {
	usr, _ := user.Current()
	dir := filepath.Join(usr.HomeDir, ".config", "iautokey")
	path := filepath.Join(dir, "config.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	cfg := map[string]any{
		"autoEnter": map[string]any{
			"enabled": true,
			"key":     "right_command",
			"delayMs": 600,
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Println("✅ 配置文件已创建:", path)
	return nil
}

func ensurePlist() error {
	usr, _ := user.Current()
	configDir := filepath.Join(usr.HomeDir, ".config", "iautokey")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.iautokey</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/iautokey.log</string>
    <key>StandardErrorPath</key>
    <string>%s/iautokey_error.log</string>
</dict>
</plist>
`, binaryPath(), configDir, configDir)
	return os.WriteFile(plistPath(), []byte(plist), 0o600)
}

func binaryPath() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, ".local", "bin", "iautokey")
}

func plistPath() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, "Library", "LaunchAgents", "com.user.iautokey.plist")
}

func waitForHealth() error {
	for range 30 {
		out, err := exec.Command("pgrep", "-f", "iautokey").Output()
		if err == nil && len(out) > 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("health check timeout")
}

func openAccessibilitySettings() {
	_ = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}

func waitForAccessibilityPermission(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hasAccessibilityPermission() {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}


func cmdHelp() {
	fmt.Print(`iautokey ` + version + ` — 修饰键释放后自动模拟 Enter

用法:
  iautokey                   启动守护进程
  iautokey status            服务状态
  iautokey restart           重启服务
  iautokey update            升级到最新版
  iautokey setup             首次安装：配置 + 开机自启
  iautokey version           版本号
  iautokey help              帮助

配置: ~/.config/iautokey/config.json
`)
}

func execCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pidFile() string {
	usr, _ := user.Current()
	return filepath.Join(usr.HomeDir, ".config", "iautokey", "iautokey.pid")
}

func loadConfig() (*config, error) {
	usr, _ := user.Current()
	path := filepath.Join(usr.HomeDir, ".config", "iautokey", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("未找到配置，请创建 ~/.config/iautokey/config.json")
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
