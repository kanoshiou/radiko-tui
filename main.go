package main

import (
	"flag"
	"fmt"
	"os"

	"radikojp/api"
	"radikojp/hook"
	"radikojp/tui"
)

func main() {
	// 解析命令行参数
	volumePercent := flag.Int("volume", 80, "Initial volume (0-100)")
	flag.Parse()

	// 转换为 0.0-1.0 范围
	initialVolume := float64(*volumePercent) / 100.0
	if initialVolume < 0 {
		initialVolume = 0
	} else if initialVolume > 1 {
		initialVolume = 1
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

	// 运行 TUI
	fmt.Println("� 启动界面...")
	err = tui.Run(stations, authToken, initialVolume)
	if err != nil {
		fmt.Printf("❌ 界面错误: %v\n", err)
		os.Exit(1)
	}
}
