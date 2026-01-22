//go:build !noaudio

package player

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// HTTPPlayer is a player that streams PCM audio from a remote server
type HTTPPlayer struct {
	serverURL    string
	stationID    string
	mu           sync.Mutex
	playing      bool
	ctx          context.Context
	cancel       context.CancelFunc
	httpClient   *http.Client
	response     *http.Response
	otoContext   *oto.Context
	otoPlayer    *oto.Player
	volume       float64
	muted        bool
	lastDataTime time.Time
}

// NewHTTPPlayer creates a new HTTP stream player
func NewHTTPPlayer(serverURL string, initialVolume float64) *HTTPPlayer {
	ctx, cancel := context.WithCancel(context.Background())

	if initialVolume < 0 {
		initialVolume = 0
	} else if initialVolume > 1 {
		initialVolume = 1
	}

	return &HTTPPlayer{
		serverURL: serverURL,
		ctx:       ctx,
		cancel:    cancel,
		volume:    initialVolume,
		muted:     false,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming
		},
	}
}

// Play starts playback of the specified station
func (p *HTTPPlayer) Play(stationID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.playing {
		return fmt.Errorf("already playing")
	}

	p.stationID = stationID

	// Initialize audio if needed
	if p.otoContext == nil {
		err := p.initAudio(48000, 2)
		if err != nil {
			return fmt.Errorf("failed to init audio: %w", err)
		}
	}

	// Build PCM stream URL
	streamURL := fmt.Sprintf("%s/api/play/%s/pcm", p.serverURL, stationID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(p.ctx, "GET", streamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	p.response = resp
	p.playing = true
	p.lastDataTime = time.Now()

	go p.pumpAudio(resp.Body)
	go p.monitorPlayback()

	return nil
}

func (p *HTTPPlayer) initAudio(sampleRate, channelCount int) error {
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

func (p *HTTPPlayer) pumpAudio(reader io.Reader) {
	// Use a buffered reader to absorb network jitter (64KB buffer)
	bufferedReader := bufio.NewReaderSize(reader, 65536)

	volumeReader := &HTTPVolumeReader{
		reader:  bufferedReader,
		player:  p,
		residue: make([]byte, 0, 4),
	}

	p.otoPlayer = p.otoContext.NewPlayer(volumeReader)
	p.otoPlayer.Play()

	<-p.ctx.Done()
}

// HTTPVolumeReader wraps io.Reader and applies volume control with frame alignment
type HTTPVolumeReader struct {
	reader  io.Reader
	player  *HTTPPlayer
	residue []byte // Buffer for incomplete PCM frames (max 3 bytes)
}

func (vr *HTTPVolumeReader) Read(p []byte) (n int, err error) {
	// PCM frame size: 2 bytes per sample * 2 channels = 4 bytes per frame
	const frameSize = 4

	// If we have residue, prepend it
	offset := 0
	if len(vr.residue) > 0 {
		offset = copy(p, vr.residue)
		vr.residue = vr.residue[:0]
	}

	// Read from network into the buffer after residue
	n, err = vr.reader.Read(p[offset:])
	n += offset

	if n > 0 {
		vr.player.mu.Lock()
		vr.player.lastDataTime = time.Now()
		vr.player.mu.Unlock()

		// Ensure frame alignment
		alignedLen := (n / frameSize) * frameSize
		if alignedLen < n {
			// Save incomplete bytes for next read
			vr.residue = append(vr.residue, p[alignedLen:n]...)
			n = alignedLen
		}

		// Apply volume to aligned data only
		if n > 0 {
			volume := vr.player.getEffectiveVolume()
			for i := 0; i < n; i += 2 {
				sample := int16(uint16(p[i]) | uint16(p[i+1])<<8)
				sample = int16(float64(sample) * volume)
				p[i] = byte(sample)
				p[i+1] = byte(sample >> 8)
			}
		}
	}
	return n, err
}

func (p *HTTPPlayer) Stop() {
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

	if p.response != nil {
		p.response.Body.Close()
		p.response = nil
	}

	p.playing = false
	p.ctx, p.cancel = context.WithCancel(context.Background())
}

func (p *HTTPPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

func (p *HTTPPlayer) SetVolume(volume float64) {
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

func (p *HTTPPlayer) GetVolume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *HTTPPlayer) IncreaseVolume(delta float64) {
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

func (p *HTTPPlayer) DecreaseVolume(delta float64) {
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

func (p *HTTPPlayer) ToggleMute() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.muted = !p.muted
}

func (p *HTTPPlayer) IsMuted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.muted
}

func (p *HTTPPlayer) getEffectiveVolume() float64 {
	if p.muted {
		return 0
	}
	return p.volume
}

// monitorPlayback monitors playback status and auto-reconnects
func (p *HTTPPlayer) monitorPlayback() {
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
					p.mu.Unlock()
					p.Reconnect()
					continue
				}
			}
			p.mu.Unlock()
		}
	}
}

// Reconnect attempts to reconnect to the stream
func (p *HTTPPlayer) Reconnect() error {
	p.mu.Lock()
	stationID := p.stationID
	volume := p.volume
	muted := p.muted
	p.mu.Unlock()

	p.Stop()
	time.Sleep(500 * time.Millisecond)

	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.volume = volume
	p.muted = muted
	p.mu.Unlock()

	return p.Play(stationID)
}

// GetStationID returns the current station ID
func (p *HTTPPlayer) GetStationID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stationID
}

// Recording methods (not supported in server mode)

func (p *HTTPPlayer) StartRecording(stationName string) error {
	return fmt.Errorf("サーバーモードでは録音機能はサポートされていません")
}

func (p *HTTPPlayer) StopRecording() (string, error) {
	return "", fmt.Errorf("サーバーモードでは録音機能はサポートされていません")
}

func (p *HTTPPlayer) IsRecording() bool {
	return false
}

func (p *HTTPPlayer) GetRecordingInfo() (filePath string, duration time.Duration, stationName string) {
	return "", 0, ""
}

func (p *HTTPPlayer) ToggleRecording(stationName string) (started bool, filePath string, err error) {
	return false, "", fmt.Errorf("サーバーモードでは録音機能はサポートされていません")
}
