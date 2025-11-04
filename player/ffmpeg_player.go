package player

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// FFmpegPlayer 使用 ffmpeg 解码的播放器
type FFmpegPlayer struct {
	authToken   string
	streamURL   string
	mu          sync.Mutex
	playing     bool
	ctx         context.Context
	cancel      context.CancelFunc
	cmd         *exec.Cmd
	otoContext  *oto.Context
	otoPlayer   *oto.Player
	volume      float64 // 音量 0.0 - 1.0
	muted       bool
	volumeBeforeMute float64
	lastDataTime time.Time // 最后接收数据的时间
	onReconnect  func() string // 重连回调函数，返回新的 authToken
}

// NewFFmpegPlayer 创建 ffmpeg 播放器
func NewFFmpegPlayer(authToken string, initialVolume float64) *FFmpegPlayer {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 确保音量在有效范围内
	if initialVolume < 0 {
		initialVolume = 0
	} else if initialVolume > 1 {
		initialVolume = 1
	}
	
	return &FFmpegPlayer{
		authToken: authToken,
		ctx:       ctx,
		cancel:    cancel,
		volume:    initialVolume,
		muted:     false,
	}
}

// SetReconnectCallback 设置重连回调函数
func (p *FFmpegPlayer) SetReconnectCallback(callback func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onReconnect = callback
}

// Play 开始播放
func (p *FFmpegPlayer) Play(streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.playing {
		return fmt.Errorf("already playing")
	}
	
	p.streamURL = streamURL

	// 检查 ffmpeg 是否可用
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH. Please install ffmpeg: %w", err)
	}

	// 初始化音频输出
	err = p.initAudio(48000, 2)
	if err != nil {
		return fmt.Errorf("failed to init audio: %w", err)
	}

	// 创建 ffmpeg 命令
	// 使用 ffmpeg 解码 HLS 流并输出 PCM
	// 不使用 ffmpeg 的音量滤镜，而是在代码中实时控制
	p.cmd = exec.CommandContext(p.ctx, "ffmpeg",
		"-headers", fmt.Sprintf("X-Radiko-AuthToken: %s", p.authToken),
		"-i", streamURL,
		"-f", "s16le",      // 16-bit PCM
		"-ar", "48000",     // 48kHz
		"-ac", "2",         // 立体声
		"-loglevel", "error", // 只显示错误
		"pipe:1",           // 输出到 stdout
	)

	// 获取 stdout
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// 启动 ffmpeg
	err = p.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	p.playing = true
	p.lastDataTime = time.Now()

	// 启动数据传输
	go p.pumpAudio(stdout)
	
	// 启动监控协程
	go p.monitorPlayback()

	return nil
}

// initAudio 初始化音频输出
func (p *FFmpegPlayer) initAudio(sampleRate, channelCount int) error {
	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       oto.FormatSignedInt16LE,
	}
	
	var ready chan struct{}
	var err error
	p.otoContext, ready, err = oto.NewContext(op)
	if err != nil {
		return fmt.Errorf("failed to create oto context: %w", err)
	}
	
	<-ready
	return nil
}

// pumpAudio 传输音频数据
func (p *FFmpegPlayer) pumpAudio(reader io.Reader) {
	// 创建一个带音量控制的 reader
	volumeReader := &VolumeReader{
		reader: reader,
		player: p,
	}
	
	// 创建播放器
	p.otoPlayer = p.otoContext.NewPlayer(volumeReader)
	p.otoPlayer.Play()

	// 等待上下文取消
	<-p.ctx.Done()
}

// VolumeReader 包装 io.Reader 并应用音量控制
type VolumeReader struct {
	reader io.Reader
	player *FFmpegPlayer
}

// Read 读取音频数据并应用音量
func (vr *VolumeReader) Read(p []byte) (n int, err error) {
	n, err = vr.reader.Read(p)
	if n > 0 {
		// 更新最后接收数据的时间
		vr.player.mu.Lock()
		vr.player.lastDataTime = time.Now()
		vr.player.mu.Unlock()
		
		// 获取当前有效音量
		volume := vr.player.getEffectiveVolume()
		
		// 对 16-bit PCM 数据应用音量
		// 每个样本是 2 字节（int16）
		for i := 0; i < n-1; i += 2 {
			// 读取 16-bit 样本（小端序）
			sample := int16(p[i]) | int16(p[i+1])<<8
			
			// 应用音量
			sample = int16(float64(sample) * volume)
			
			// 写回
			p[i] = byte(sample)
			p[i+1] = byte(sample >> 8)
		}
	}
	return n, err
}

// Stop 停止播放
func (p *FFmpegPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.playing {
		return
	}
	
	p.cancel()
	
	if p.otoPlayer != nil {
		p.otoPlayer.Close()
	}
	
	// oto v3 会在 context 关闭时自动清理
	
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}

	p.playing = false
}

// IsPlaying 是否正在播放
func (p *FFmpegPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// SetVolume 设置音量 (0.0 - 1.0)
func (p *FFmpegPlayer) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	
	p.volume = volume
	if p.muted {
		p.muted = false
	}
}

// GetVolume 获取当前音量
func (p *FFmpegPlayer) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// IncreaseVolume 增加音量
func (p *FFmpegPlayer) IncreaseVolume(delta float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.volume += delta
	if p.volume > 1 {
		p.volume = 1
	}
	if p.muted {
		p.muted = false
	}
}

// DecreaseVolume 减少音量
func (p *FFmpegPlayer) DecreaseVolume(delta float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.volume -= delta
	if p.volume < 0 {
		p.volume = 0
	}
	if p.muted {
		p.muted = false
	}
}

// ToggleMute 切换静音状态
func (p *FFmpegPlayer) ToggleMute() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.muted {
		p.muted = false
	} else {
		p.volumeBeforeMute = p.volume
		p.muted = true
	}
}

// IsMuted 是否静音
func (p *FFmpegPlayer) IsMuted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.muted
}

// getEffectiveVolume 获取有效音量（考虑静音状态）
func (p *FFmpegPlayer) getEffectiveVolume() float64 {
	if p.muted {
		return 0
	}
	return p.volume
}

// monitorPlayback 监控播放状态，检测卡顿并自动重连
func (p *FFmpegPlayer) monitorPlayback() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.playing {
				// 检查是否超过 10 秒没有接收到数据
				if time.Since(p.lastDataTime) > 10*time.Second {
					fmt.Println("\n⚠ 检测到播放卡顿，正在自动重连...")
					p.mu.Unlock()
					p.Reconnect()
					continue
				}
			}
			p.mu.Unlock()
		}
	}
}

// Reconnect 重新连接并播放
func (p *FFmpegPlayer) Reconnect() error {
	fmt.Println("🔄 停止当前播放...")
	
	// 保存当前音量设置
	p.mu.Lock()
	volume := p.volume
	muted := p.muted
	streamURL := p.streamURL
	onReconnect := p.onReconnect
	p.mu.Unlock()
	
	// 停止当前播放
	p.Stop()
	
	// 等待一下确保资源释放
	time.Sleep(500 * time.Millisecond)
	
	// 获取新的 auth token
	var newAuthToken string
	if onReconnect != nil {
		fmt.Println("🔑 重新获取认证...")
		newAuthToken = onReconnect()
		if newAuthToken == "" {
			return fmt.Errorf("failed to get new auth token")
		}
		fmt.Println("✓ 认证成功")
	} else {
		newAuthToken = p.authToken
	}
	
	// 创建新的上下文
	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.authToken = newAuthToken
	p.volume = volume
	p.muted = muted
	p.mu.Unlock()
	
	// 重新开始播放
	fmt.Println("▶ 重新开始播放...")
	err := p.Play(streamURL)
	if err != nil {
		return fmt.Errorf("failed to restart playback: %w", err)
	}
	
	fmt.Println("✓ 重连成功")
	fmt.Println()
	return nil
}
