package main

import (
	"flag"
	"fmt"
	"os"

	"radiko-tui/api"
	"radiko-tui/config"
	"radiko-tui/server"
	"radiko-tui/tui"
)

func main() {
	// Parse command line arguments
	volumePercent := flag.Int("volume", -1, "Initial volume (0-100), -1 means use saved config")
	serverMode := flag.Bool("server", false, "Run in server mode (HTTP streaming)")
	port := flag.Int("port", 8080, "Server port (server mode only)")
	graceSeconds := flag.Int("grace", 10, "Seconds to keep ffmpeg alive after last client disconnects (server mode only)")
	flag.Parse()

	// Server mode
	if *serverMode {
		runServer(*port, *graceSeconds)
		return
	}

	// TUI mode
	runTUI(*volumePercent)
}

// runServer starts the HTTP streaming server
func runServer(port int, graceSeconds int) {
	fmt.Println("🚀 サーバーモードで起動中...")
	s := server.NewServer(port, graceSeconds)
	if err := s.Start(); err != nil {
		fmt.Printf("❌ サーバーエラー: %v\n", err)
		os.Exit(1)
	}
}

// runTUI starts the terminal UI mode
func runTUI(volumePercent int) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("⚠ 設定の読み込みに失敗しました。デフォルト設定を使用します: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// If volume is specified via command line, override config
	if volumePercent >= 0 {
		cfg.Volume = float64(volumePercent) / 100.0
		if cfg.Volume < 0 {
			cfg.Volume = 0
		} else if cfg.Volume > 1 {
			cfg.Volume = 1
		}
	}

	// Get authentication token
	fmt.Println("🔐 認証中...")
	authToken := api.Auth(cfg.AreaID)
	fmt.Println("✓ 認証成功")

	// Get station list
	fmt.Printf("📡 %s 地域の放送局リストを取得中...\n", cfg.AreaID)
	stations, err := api.GetStations(cfg.AreaID)
	if err != nil {
		fmt.Printf("❌ 放送局リストの取得に失敗しました: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ %d 局を検出しました\n", len(stations))

	if len(stations) == 0 {
		fmt.Println("❌ 利用可能な放送局がありません")
		os.Exit(1)
	}

	// Display last played station
	if cfg.LastStationID != "" {
		fmt.Printf("📻 前回再生: %s\n", cfg.LastStationID)
	}

	// Run TUI
	fmt.Println("🚀 インターフェースを起動中...")
	err = tui.Run(stations, authToken, cfg)
	if err != nil {
		fmt.Printf("❌ インターフェースエラー: %v\n", err)
		os.Exit(1)
	}
}

