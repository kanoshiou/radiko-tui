package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"radiko-tui/api"
	"radiko-tui/model"
)

// Server represents the HTTP streaming server
type Server struct {
	port          int
	streamManager *StreamManager
	graceSeconds  int // Grace period before killing ffmpeg after last client disconnects
}

// NewServer creates a new streaming server
func NewServer(port int, graceSeconds int) *Server {
	if graceSeconds <= 0 {
		graceSeconds = 10 // Default 10 seconds grace period
	}
	return &Server{
		port:          port,
		streamManager: NewStreamManager(graceSeconds),
		graceSeconds:  graceSeconds,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/play/{stationID}", s.handlePlayRequest)
	mux.HandleFunc("/api/status", s.handleStatus)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("📡 サーバーを開始しました: http://localhost%s", addr)
	log.Printf("   使用例: vlc http://localhost%s/api/play/QRR", addr)
	log.Printf("   ffmpeg保持時間: %d秒", s.graceSeconds)

	return http.ListenAndServe(addr, mux)
}

// handleStatus returns the current stream status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := s.streamManager.GetStatus()
	w.Write([]byte(status))
}

// handlePlayRequest routes different HTTP methods
func (s *Server) handlePlayRequest(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("stationID")
	log.Printf("📥 リクエスト: %s %s (from %s)", r.Method, r.URL.Path, r.RemoteAddr)

	switch r.Method {
	case http.MethodHead:
		s.handleHead(w, r, stationID)
	case http.MethodGet:
		s.handlePlay(w, r, stationID)
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHead handles HEAD requests
func (s *Server) handleHead(w http.ResponseWriter, r *http.Request, stationID string) {
	w.Header().Set("Content-Type", "audio/aac")
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("icy-name", fmt.Sprintf("Radiko - %s", stationID))
	w.Header().Set("icy-genre", "Radio")
	w.WriteHeader(http.StatusOK)
}

// handlePlay handles GET requests - stream audio
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request, stationID string) {
	if stationID == "" {
		http.Error(w, "stationID is required", http.StatusBadRequest)
		return
	}

	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	log.Printf("🎵 クライアント接続: %s → %s", clientID, stationID)

	// Set headers
	w.Header().Set("Content-Type", "audio/aac")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("icy-name", fmt.Sprintf("Radiko - %s", stationID))
	w.Header().Set("icy-genre", "Radio")

	// Subscribe to stream
	err := s.streamManager.Subscribe(r.Context(), w, stationID, clientID)
	if err != nil {
		log.Printf("❌ ストリームエラー [%s]: %v", clientID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("👋 クライアント切断: %s", clientID)
}

// ============================================================================
// StreamManager - Manages ffmpeg instances per station
// ============================================================================

// StreamManager manages all active streams
type StreamManager struct {
	mu           sync.RWMutex
	streams      map[string]*StationStream
	graceSeconds int
}

// NewStreamManager creates a new stream manager
func NewStreamManager(graceSeconds int) *StreamManager {
	return &StreamManager{
		streams:      make(map[string]*StationStream),
		graceSeconds: graceSeconds,
	}
}

// GetStatus returns JSON status of all streams
func (sm *StreamManager) GetStatus() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := "{"
	first := true
	for stationID, stream := range sm.streams {
		if !first {
			result += ","
		}
		first = false
		stream.mu.RLock()
		clientCount := len(stream.clients)
		stream.mu.RUnlock()
		result += fmt.Sprintf(`"%s":{"clients":%d,"running":%t}`, stationID, clientCount, stream.running)
	}
	result += "}"
	return result
}

// Subscribe adds a client to a station stream
func (sm *StreamManager) Subscribe(ctx context.Context, w http.ResponseWriter, stationID, clientID string) error {
	stream, err := sm.getOrCreateStream(stationID)
	if err != nil {
		return err
	}

	return stream.AddClient(ctx, w, clientID)
}

// getOrCreateStream gets an existing stream or creates a new one
func (sm *StreamManager) getOrCreateStream(stationID string) (*StationStream, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if stream already exists
	if stream, exists := sm.streams[stationID]; exists {
		stream.CancelGracePeriod() // Cancel any pending shutdown
		if stream.running {
			log.Printf("♻️ 既存のffmpegを再利用: %s", stationID)
			return stream, nil
		}
	}

	// Create new stream
	log.Printf("🆕 新しいffmpegを開始: %s", stationID)
	stream, err := NewStationStream(stationID, sm.graceSeconds, func() {
		sm.removeStream(stationID)
	})
	if err != nil {
		return nil, err
	}

	sm.streams[stationID] = stream
	return stream, nil
}

// removeStream removes a stream from the manager
func (sm *StreamManager) removeStream(stationID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, stationID)
	log.Printf("🗑️ ストリーム削除: %s", stationID)
}

// ============================================================================
// StationStream - Manages a single station's ffmpeg process and clients
// ============================================================================

// Client represents a connected client
type Client struct {
	id     string
	writer http.ResponseWriter
	done   chan struct{}
}

// StationStream manages a single station's stream
type StationStream struct {
	stationID    string
	mu           sync.RWMutex
	clients      map[string]*Client
	running      bool
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	graceTimer   *time.Timer
	graceSeconds int
	onClose      func()

	// Broadcast channel
	broadcast chan []byte
}

// NewStationStream creates and starts a new station stream
func NewStationStream(stationID string, graceSeconds int, onClose func()) (*StationStream, error) {
	// Get area for this station
	areaID, err := api.GetStationArea(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get station area: %w", err)
	}
	log.Printf("📍 エリア: %s", areaID)

	// Authenticate
	log.Printf("🔐 認証中...")
	authToken := api.Auth(areaID)
	if authToken == "" {
		return nil, fmt.Errorf("authentication failed")
	}
	log.Printf("✓ 認証成功")

	// Get stream URLs
	playlistURLs, err := api.GetStreamURLs(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream URL: %w", err)
	}
	if len(playlistURLs) == 0 {
		return nil, fmt.Errorf("no stream URLs found")
	}

	// Build final stream URL
	lsid := model.GenLsid()
	lastURL := playlistURLs[len(playlistURLs)-1]
	streamURL := fmt.Sprintf("%s?station_id=%s&l=30&lsid=%s&type=b", lastURL, stationID, lsid)

	// Create stream
	stream := &StationStream{
		stationID:    stationID,
		clients:      make(map[string]*Client),
		graceSeconds: graceSeconds,
		onClose:      onClose,
		broadcast:    make(chan []byte, 100),
	}

	// Start ffmpeg
	if err := stream.startFFmpeg(streamURL, authToken); err != nil {
		return nil, err
	}

	return stream, nil
}

// startFFmpeg starts the ffmpeg process
func (ss *StationStream) startFFmpeg(streamURL, authToken string) error {
	ctx, cancel := context.WithCancel(context.Background())
	ss.cancel = cancel

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "10",
		"-timeout", "30000000",
		"-headers", fmt.Sprintf("X-Radiko-AuthToken: %s\r\n", authToken),
		"-i", streamURL,
		"-c:a", "copy",
		"-f", "adts",
		"-fflags", "+nobuffer+flush_packets",
		"-flags", "low_delay",
		"-loglevel", "warning",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	ss.cmd = cmd
	ss.running = true

	// Log ffmpeg errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("ffmpeg [%s]: %s", ss.stationID, scanner.Text())
		}
	}()

	// Read from ffmpeg and broadcast to clients
	go ss.readAndBroadcast(stdout)

	// Broadcast to clients
	go ss.broadcastLoop()

	log.Printf("▶ ffmpeg開始: %s", ss.stationID)
	return nil
}

// readAndBroadcast reads from ffmpeg stdout and sends to broadcast channel
func (ss *StationStream) readAndBroadcast(stdout io.Reader) {
	reader := bufio.NewReaderSize(stdout, 32768)
	buf := make([]byte, 8192)
	firstData := true

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if firstData {
				log.Printf("📦 最初のデータ受信: %s", ss.stationID)
				firstData = false
			}

			// Copy data to avoid race conditions
			data := make([]byte, n)
			copy(data, buf[:n])

			// Non-blocking send to broadcast channel
			select {
			case ss.broadcast <- data:
			default:
				// Channel full, drop oldest data
				select {
				case <-ss.broadcast:
				default:
				}
				ss.broadcast <- data
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("❌ ffmpeg読み取りエラー [%s]: %v", ss.stationID, err)
			}
			break
		}
	}

	ss.mu.Lock()
	ss.running = false
	ss.mu.Unlock()

	close(ss.broadcast)
	log.Printf("⏹ ffmpeg終了: %s", ss.stationID)
}

// broadcastLoop sends data to all connected clients
func (ss *StationStream) broadcastLoop() {
	for data := range ss.broadcast {
		ss.mu.RLock()
		clients := make([]*Client, 0, len(ss.clients))
		for _, c := range ss.clients {
			clients = append(clients, c)
		}
		ss.mu.RUnlock()

		for _, client := range clients {
			select {
			case <-client.done:
				continue
			default:
				_, err := client.writer.Write(data)
				if err != nil {
					close(client.done)
					continue
				}
				if f, ok := client.writer.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	}
}

// AddClient adds a client to this stream
func (ss *StationStream) AddClient(ctx context.Context, w http.ResponseWriter, clientID string) error {
	client := &Client{
		id:     clientID,
		writer: w,
		done:   make(chan struct{}),
	}

	ss.mu.Lock()
	ss.clients[clientID] = client
	clientCount := len(ss.clients)
	ss.mu.Unlock()

	log.Printf("📊 クライアント追加 [%s]: %d 接続中", ss.stationID, clientCount)

	// Wait for client disconnect or stream end
	select {
	case <-ctx.Done():
		// Client disconnected
	case <-client.done:
		// Write error occurred
	}

	ss.removeClient(clientID)
	return nil
}

// removeClient removes a client from this stream
func (ss *StationStream) removeClient(clientID string) {
	ss.mu.Lock()
	delete(ss.clients, clientID)
	clientCount := len(ss.clients)
	ss.mu.Unlock()

	log.Printf("📊 クライアント削除 [%s]: %d 接続中", ss.stationID, clientCount)

	// If no clients left, start grace period
	if clientCount == 0 {
		ss.startGracePeriod()
	}
}

// startGracePeriod starts the grace period timer
func (ss *StationStream) startGracePeriod() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.graceTimer != nil {
		return // Already running
	}

	log.Printf("⏰ 猶予期間開始 [%s]: %d秒", ss.stationID, ss.graceSeconds)

	ss.graceTimer = time.AfterFunc(time.Duration(ss.graceSeconds)*time.Second, func() {
		ss.mu.Lock()
		clientCount := len(ss.clients)
		ss.mu.Unlock()

		if clientCount == 0 {
			log.Printf("⏰ 猶予期間終了、ffmpeg停止: %s", ss.stationID)
			ss.Stop()
		}
	})
}

// CancelGracePeriod cancels the grace period timer
func (ss *StationStream) CancelGracePeriod() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.graceTimer != nil {
		ss.graceTimer.Stop()
		ss.graceTimer = nil
		log.Printf("⏰ 猶予期間キャンセル: %s", ss.stationID)
	}
}

// Stop stops the ffmpeg process and cleans up
func (ss *StationStream) Stop() {
	ss.mu.Lock()
	if ss.cancel != nil {
		ss.cancel()
	}
	ss.running = false
	ss.mu.Unlock()

	if ss.cmd != nil {
		ss.cmd.Wait()
	}

	if ss.onClose != nil {
		ss.onClose()
	}
}
