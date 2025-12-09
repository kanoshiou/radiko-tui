package main

import (
	"flag"
	"fmt"
	"os"

	"radikojp/api"
	"radikojp/config"
	"radikojp/hook"
	"radikojp/tui"
)

func main() {
	// 解析命令行参数
	volumePercent := flag.Int("volume", -1, "Initial volume (0-100), -1 means use saved config")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("⚠ 加载配置失败，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// 如果命令行指定了音量，则覆盖配置
	if *volumePercent >= 0 {
		cfg.Volume = float64(*volumePercent) / 100.0
		if cfg.Volume < 0 {
			cfg.Volume = 0
		} else if cfg.Volume > 1 {
			cfg.Volume = 1
		}
	}

	// 获取认证 token
	fmt.Println("🔐 正在认证...")
	authToken := hook.Auth()
	fmt.Println("✓ 认证成功")

	// 获取电台列表
	fmt.Println("📡 正在获取电台列表...")
	stations, err := api.GetStations()
	if err != nil {
		fmt.Printf("❌ 获取电台列表失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ 发现 %d 个电台\n", len(stations))

	if len(stations) == 0 {
		fmt.Println("❌ 没有可用的电台")
		os.Exit(1)
	}

	// 显示上次播放的电台
	if cfg.LastStationID != "" {
		fmt.Printf("📻 上次播放: %s\n", cfg.LastStationID)
	}

	// 运行 TUI
	fmt.Println("🚀 启动界面...")
	err = tui.Run(stations, authToken, cfg)
	if err != nil {
		fmt.Printf("❌ 界面错误: %v\n", err)
		os.Exit(1)
	}
}
