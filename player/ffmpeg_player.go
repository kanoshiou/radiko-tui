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

// ReconnectStatus represents the reconnection state
type ReconnectStatus int

const (
	ReconnectNone ReconnectStatus = iota
	ReconnectStarted
	ReconnectAuth
	ReconnectPlaying
	ReconnectSuccess
	ReconnectFailed
)

// FFmpegPlayer is a player that uses ffmpeg for decoding
type FFmpegPlayer struct {
	authToken        string
	streamURL        string
	mu               sync.Mutex
	playing          bool
	ctx              context.Context
	cancel           context.CancelFunc
	cmd              *exec.Cmd
	otoContext       *oto.Context
	otoPlayer        *oto.Player
	volume           float64
	muted            bool
	volumeBeforeMute float64
	lastDataTime     time.Time
	onReconnect      func() string
	reconnectStatus  ReconnectStatus // Reconnection status (for TUI to query)
	lastError        string          // Last error message
}

// NewFFmpegPlayer creates a new ffmpeg player
func NewFFmpegPlayer(authToken string, initialVolume float64) *FFmpegPlayer {
	ctx, cancel := context.WithCancel(context.Background())

	if initialVolume < 0 {
		initialVolume = 0
	} else if initialVolume > 1 {
		initialVolume = 1
	}

	return &FFmpegPlayer{
		authToken:       authToken,
		ctx:             ctx,
		cancel:          cancel,
		volume:          initialVolume,
		muted:           false,
		reconnectStatus: ReconnectNone,
	}
}

// SetReconnectCallback sets the reconnection callback function
func (p *FFmpegPlayer) SetReconnectCallback(callback func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onReconnect = callback
}

// UpdateAuthToken updates the authentication token (used when switching stations)
func (p *FFmpegPlayer) UpdateAuthToken(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.authToken = token
}

// GetReconnectStatus returns the current reconnection status
func (p *FFmpegPlayer) GetReconnectStatus() ReconnectStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.reconnectStatus
}

// GetLastError returns the last error message
func (p *FFmpegPlayer) GetLastError() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastError
}

// ClearReconnectStatus clears the reconnection status
func (p *FFmpegPlayer) ClearReconnectStatus() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reconnectStatus = ReconnectNone
	p.lastError = ""
}

// Play starts playback
func (p *FFmpegPlayer) Play(streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.playing {
		return fmt.Errorf("already playing")
	}

	p.streamURL = streamURL
	p.reconnectStatus = ReconnectNone
	p.lastError = ""

	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH. Please install ffmpeg: %w", err)
	}

	if p.otoContext == nil {
		err = p.initAudio(48000, 2)
		if err != nil {
			return fmt.Errorf("failed to init audio: %w", err)
		}
	}

	p.cmd = exec.CommandContext(p.ctx, "ffmpeg",
		"-headers", fmt.Sprintf("X-Radiko-AuthToken: %s", p.authToken),
		"-i", streamURL,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-loglevel", "error",
		"pipe:1",
	)

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	err = p.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	p.playing = true
	p.lastDataTime = time.Now()

	go p.pumpAudio(stdout)
	go p.monitorPlayback()

	return nil
}

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

func (p *FFmpegPlayer) pumpAudio(reader io.Reader) {
	volumeReader := &VolumeReader{
		reader: reader,
		player: p,
	}

	p.otoPlayer = p.otoContext.NewPlayer(volumeReader)
	p.otoPlayer.Play()

	<-p.ctx.Done()
}

// VolumeReader wraps io.Reader and applies volume control
type VolumeReader struct {
	reader io.Reader
	player *FFmpegPlayer
}

func (vr *VolumeReader) Read(p []byte) (n int, err error) {
	n, err = vr.reader.Read(p)
	if n > 0 {
		vr.player.mu.Lock()
		vr.player.lastDataTime = time.Now()
		vr.player.mu.Unlock()

		volume := vr.player.getEffectiveVolume()

		for i := 0; i < n-1; i += 2 {
			sample := int16(p[i]) | int16(p[i+1])<<8
			sample = int16(float64(sample) * volume)
			p[i] = byte(sample)
			p[i+1] = byte(sample >> 8)
		}
	}
	return n, err
}

func (p *FFmpegPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.playing {
		return
	}

	p.cancel()

	if p.otoPlayer != nil {
		p.otoPlayer.Close()
		p.otoPlayer = nil
	}

	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}

	p.playing = false
	p.ctx, p.cancel = context.WithCancel(context.Background())
}

func (p *FFmpegPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

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

func (p *FFmpegPlayer) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

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

func (p *FFmpegPlayer) IsMuted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.muted
}

func (p *FFmpegPlayer) getEffectiveVolume() float64 {
	if p.muted {
		return 0
	}
	return p.volume
}

// monitorPlayback monitors playback status (silent version, no terminal output)
func (p *FFmpegPlayer) monitorPlayback() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.playing {
				if time.Since(p.lastDataTime) > 5*time.Second {
					p.reconnectStatus = ReconnectStarted
					p.mu.Unlock()
					p.Reconnect()
					continue
				}
			}
			p.mu.Unlock()
		}
	}
}

// Reconnect attempts to reconnect (silent version)
func (p *FFmpegPlayer) Reconnect() error {
	p.mu.Lock()
	p.reconnectStatus = ReconnectStarted
	volume := p.volume
	muted := p.muted
	streamURL := p.streamURL
	onReconnect := p.onReconnect
	p.mu.Unlock()

	p.Stop()
	time.Sleep(500 * time.Millisecond)

	var newAuthToken string
	if onReconnect != nil {
		p.mu.Lock()
		p.reconnectStatus = ReconnectAuth
		p.mu.Unlock()

		newAuthToken = onReconnect()
		if newAuthToken == "" {
			p.mu.Lock()
			p.reconnectStatus = ReconnectFailed
			p.lastError = "認証の取得に失敗しました"
			p.mu.Unlock()
			return fmt.Errorf("failed to get new auth token")
		}
	} else {
		newAuthToken = p.authToken
	}

	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.authToken = newAuthToken
	p.volume = volume
	p.muted = muted
	p.reconnectStatus = ReconnectPlaying
	p.mu.Unlock()

	err := p.Play(streamURL)
	if err != nil {
		p.mu.Lock()
		p.reconnectStatus = ReconnectFailed
		p.lastError = err.Error()
		p.mu.Unlock()
		return fmt.Errorf("failed to restart playback: %w", err)
	}

	p.mu.Lock()
	p.reconnectStatus = ReconnectSuccess
	p.mu.Unlock()

	return nil
}
