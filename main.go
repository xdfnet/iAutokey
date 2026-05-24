// iAutokey — 独立按键事件工具
// 监听指定修饰键的释放事件，自动模拟 Enter 键
// 配合语音输入法使用：按住键说话，松开即确认
package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
)

type autoEnterConfig struct {
	Enabled bool   `json:"enabled"`
	Key     string `json:"key"`
	DelayMs int    `json:"delayMs"`
}

type config struct {
	AutoEnter *autoEnterConfig `json:"autoEnter"`
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.AutoEnter == nil || !cfg.AutoEnter.Enabled || cfg.AutoEnter.Key == "" {
		log.Printf("未启用，退出")
		return
	}

	// 响应退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Exit(0)
	}()

	log.Printf("按键=%s delay=%dms", cfg.AutoEnter.Key, cfg.AutoEnter.DelayMs)
	startAutoEnter(cfg.AutoEnter.Key, cfg.AutoEnter.DelayMs)
}

func loadConfig() (*config, error) {
	// 优先读取本项目的独立配置
	usr, _ := user.Current()
	ownPath := filepath.Join(usr.HomeDir, ".config", "iAutokey", "config.json")
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
		return nil, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
