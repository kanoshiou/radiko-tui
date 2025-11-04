package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"radikojp/hook"
	"radikojp/player"
	"syscall"
	"time"

	"github.com/bluenviron/gohlslib/pkg/playlist"
	"github.com/eiannone/keyboard"
)

func main() {
	// 打印版本信息
	PrintVersion()

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

	url := "https://c-radiko.smartstream.ne.jp/QRR/_definst_/simul-stream.stream/playlist.m3u8?station_id=QRR&l=30&lsid=5e586af5ccb3b0b2498abfb19eaa8472&type=b"

	// 获取认证 token
	fmt.Println("Authenticating...")
	authToken := hook.Auth()
	fmt.Println("✓ Auth token obtained")

	// 获取播放列表
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Radiko-AuthToken", authToken)
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	// 解析播放列表
	byts, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		panic(err)
	}

	pl, err := playlist.Unmarshal(byts)
	if err != nil {
		panic(err)
	}

	streamUrl := ""

	switch pl := pl.(type) {
	case *playlist.Multivariant:
		fmt.Println("Multivariant playlist detected")
		if len(pl.Variants) > 0 {
			streamUrl = pl.Variants[0].URI
			fmt.Printf("Using stream: %s\n", streamUrl)
		}

	case *playlist.Media:
		fmt.Println("Media playlist detected")
		streamUrl = url
	}

	if streamUrl == "" {
		panic("No valid stream URL found")
	}

	// 创建并启动播放器
	fmt.Println("Starting ffmpeg player...")
	fmt.Println("Note: This requires ffmpeg to be installed and in PATH")
	fmt.Printf("Initial volume: %d%%\n", *volumePercent)
	fmt.Println()

	ffmpegPlayer := player.NewFFmpegPlayer(authToken, initialVolume)
	
	// 设置重连回调函数
	ffmpegPlayer.SetReconnectCallback(func() string {
		return hook.Auth()
	})

	err = ffmpegPlayer.Play(streamUrl)
	if err != nil {
		panic(fmt.Sprintf("Failed to start player: %v", err))
	}

	// 等待播放器完全启动
	time.Sleep(500 * time.Millisecond)

	fmt.Println()
	fmt.Println("🎵 Playing...")
	fmt.Println()
	printControls()
	printVolumeStatus(ffmpegPlayer)

	// 初始化键盘监听
	if err := keyboard.Open(); err != nil {
		fmt.Printf("Warning: Could not open keyboard: %v\n", err)
		fmt.Println("Volume control disabled. Press Ctrl+C to stop")

		// 等待中断信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
	} else {
		defer keyboard.Close()

		// 等待中断信号或键盘输入
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// 启动键盘监听
		go handleKeyboard(ffmpegPlayer)

		<-sigChan
	}

	fmt.Println("\nStopping player...")
	ffmpegPlayer.Stop()
	fmt.Println("Stopped")
}

func printControls() {
	fmt.Println("Controls:")
	fmt.Println("  ↑ / + / e     Increase volume")
	fmt.Println("  ↓ / - / q     Decrease volume")
	fmt.Println("  m             Mute/Unmute")
	fmt.Println("  r             Reconnect/Replay")
	fmt.Println("  0-9           Set volume to 0%-90%")
	fmt.Println("  Ctrl+C        Stop and exit")
	fmt.Println()
}

func printVolumeStatus(p *player.FFmpegPlayer) {
	volume := int(p.GetVolume() * 100)
	muted := p.IsMuted()

	status := fmt.Sprintf("Volume: %3d%%", volume)
	if muted {
		status += " [MUTED]"
	} else {
		status += "        " // 补齐空格，确保覆盖 [MUTED]
	}

	// 音量条
	barLength := 20
	filledLength := int(float64(barLength) * p.GetVolume())
	bar := ""
	for i := 0; i < barLength; i++ {
		if i < filledLength && !muted {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	// 使用固定长度的输出，确保完全覆盖之前的内容
	output := fmt.Sprintf("%s [%s]", status, bar)
	fmt.Printf("\r%-60s", output) // 左对齐，总宽度 60 字符
}

func handleKeyboard(p *player.FFmpegPlayer) {
	lastUpdate := time.Now()
	updateInterval := 50 * time.Millisecond

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return
		}

		needsRestart := false

		switch key {
		case keyboard.KeyArrowUp:
			p.IncreaseVolume(0.05)
			needsRestart = true
		case keyboard.KeyArrowDown:
			p.DecreaseVolume(0.05)
			needsRestart = true
		}

		switch char {
		case '+', '=':
			p.IncreaseVolume(0.05)
			needsRestart = true
		case '-', '_':
			p.DecreaseVolume(0.05)
			needsRestart = true
		case 'e', 'E':
			p.IncreaseVolume(0.05)
			needsRestart = true
		case 'q', 'Q':
			p.DecreaseVolume(0.05)
			needsRestart = true
		case 'm', 'M':
			p.ToggleMute()
			needsRestart = true
		case 'r', 'R':
			fmt.Println("\n🔄 手动重连中...")
			go func() {
				err := p.Reconnect()
				if err != nil {
					fmt.Printf("\n❌ 重连失败: %v\n", err)
				}
			}()
			continue
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			volume := float64(char-'0') / 10.0
			p.SetVolume(volume)
			needsRestart = true
		}

		if needsRestart && time.Since(lastUpdate) > updateInterval {
			printVolumeStatus(p)
			lastUpdate = time.Now()
		}
	}
}
