package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"radiko-tui/api"
	"radiko-tui/model"
)

// getRealIP extracts the real client IP from the request.
// It checks headers in the following priority order:
// 1. CF-Connecting-IP (Cloudflare)
// 2. X-Real-IP (nginx)
// 3. X-Forwarded-For (standard proxy, first IP in the list)
// 4. RemoteAddr (fallback)
func getRealIP(r *http.Request) string {
	// Cloudflare: CF-Connecting-IP is the most reliable when using Cloudflare
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	// nginx: X-Real-IP is typically set by nginx
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Standard proxy: X-Forwarded-For can contain multiple IPs (client, proxy1, proxy2, ...)
	// The first IP is the original client
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fallback: use RemoteAddr (strip port if present)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Return as-is if parsing fails
	}
	return ip
}

// Server represents the HTTP streaming server
type Server struct {
	port             int
	streamManager    *StreamManager
	pcmStreamManager *PCMStreamManager
	graceSeconds     int // Grace period before killing ffmpeg after last client disconnects
}

// NewServer creates a new streaming server
func NewServer(port int, graceSeconds int) *Server {
	if graceSeconds <= 0 {
		graceSeconds = 10 // Default 10 seconds grace period
	}
	return &Server{
		port:             port,
		streamManager:    NewStreamManager(graceSeconds),
		pcmStreamManager: NewPCMStreamManager(graceSeconds),
		graceSeconds:     graceSeconds,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/play/{stationID}", s.handlePlayRequest)
	mux.HandleFunc("/api/play/{stationID}/pcm", s.handlePCMPlayRequest)
	mux.HandleFunc("/api/status", s.handleStatus)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("📡 サーバーを開始しました: http://localhost%s", addr)
	log.Printf("   AAC: vlc http://localhost%s/api/play/QRR", addr)
	log.Printf("   PCM: radiko-tui --server-url http://localhost%s", addr)
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
	clientIP := getRealIP(r)
	log.Printf("📥 リクエスト: %s %s (from %s)", r.Method, r.URL.Path, clientIP)

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

	clientIP := getRealIP(r)
	clientID := fmt.Sprintf("%s-%d", clientIP, time.Now().UnixNano())
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

// handlePCMPlayRequest handles PCM format streaming requests
func (s *Server) handlePCMPlayRequest(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("stationID")
	clientIP := getRealIP(r)
	log.Printf("📥 PCMリクエスト: %s %s (from %s)", r.Method, r.URL.Path, clientIP)

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if stationID == "" {
		http.Error(w, "stationID is required", http.StatusBadRequest)
		return
	}

	clientID := fmt.Sprintf("%s-%d", clientIP, time.Now().UnixNano())
	log.Printf("🎵 PCMクライアント接続: %s → %s", clientID, stationID)

	// Set headers for PCM streaming
	w.Header().Set("Content-Type", "audio/L16;rate=48000;channels=2")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("X-Audio-Format", "s16le")
	w.Header().Set("X-Sample-Rate", "48000")
	w.Header().Set("X-Channels", "2")

	// Subscribe to PCM stream
	err := s.pcmStreamManager.Subscribe(r.Context(), w, stationID, clientID)
	if err != nil {
		log.Printf("❌ PCMストリームエラー [%s]: %v", clientID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("👋 PCMクライアント切断: %s", clientID)
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

// ============================================================================
// PCMStreamManager - Manages PCM format ffmpeg instances per station
// ============================================================================

// PCMStreamManager manages all active PCM streams
type PCMStreamManager struct {
	mu           sync.RWMutex
	streams      map[string]*PCMStationStream
	graceSeconds int
}

// NewPCMStreamManager creates a new PCM stream manager
func NewPCMStreamManager(graceSeconds int) *PCMStreamManager {
	return &PCMStreamManager{
		streams:      make(map[string]*PCMStationStream),
		graceSeconds: graceSeconds,
	}
}

// Subscribe adds a client to a PCM station stream
func (pm *PCMStreamManager) Subscribe(ctx context.Context, w http.ResponseWriter, stationID, clientID string) error {
	stream, err := pm.getOrCreateStream(stationID)
	if err != nil {
		return err
	}

	return stream.AddClient(ctx, w, clientID)
}

// getOrCreateStream gets an existing stream or creates a new one
func (pm *PCMStreamManager) getOrCreateStream(stationID string) (*PCMStationStream, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if stream already exists
	if stream, exists := pm.streams[stationID]; exists {
		stream.CancelGracePeriod()
		if stream.running {
			log.Printf("♻️ 既存のPCM ffmpegを再利用: %s", stationID)
			return stream, nil
		}
	}

	// Create new stream
	log.Printf("🆕 新しいPCM ffmpegを開始: %s", stationID)
	stream, err := NewPCMStationStream(stationID, pm.graceSeconds, func() {
		pm.removeStream(stationID)
	})
	if err != nil {
		return nil, err
	}

	pm.streams[stationID] = stream
	return stream, nil
}

// removeStream removes a stream from the manager
func (pm *PCMStreamManager) removeStream(stationID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.streams, stationID)
	log.Printf("🗑️ PCMストリーム削除: %s", stationID)
}

// ============================================================================
// PCMStationStream - Manages a single station's PCM ffmpeg process
// ============================================================================

// PCMStationStream manages a single station's PCM stream
type PCMStationStream struct {
	stationID    string
	mu           sync.RWMutex
	clients      map[string]*Client
	running      bool
	cmd          *exec.Cmd
	cancel       context.CancelFunc
	graceTimer   *time.Timer
	graceSeconds int
	onClose      func()
	broadcast    chan []byte
}

// NewPCMStationStream creates and starts a new PCM station stream
func NewPCMStationStream(stationID string, graceSeconds int, onClose func()) (*PCMStationStream, error) {
	// Get area for this station
	areaID, err := api.GetStationArea(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get station area: %w", err)
	}
	log.Printf("📍 PCMエリア: %s", areaID)

	// Authenticate
	log.Printf("🔐 PCM認証中...")
	authToken := api.Auth(areaID)
	if authToken == "" {
		return nil, fmt.Errorf("authentication failed")
	}
	log.Printf("✓ PCM認証成功")

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
	stream := &PCMStationStream{
		stationID:    stationID,
		clients:      make(map[string]*Client),
		graceSeconds: graceSeconds,
		onClose:      onClose,
		broadcast:    make(chan []byte, 500),
	}

	// Start ffmpeg with PCM output
	if err := stream.startFFmpegPCM(streamURL, authToken); err != nil {
		return nil, err
	}

	return stream, nil
}

// startFFmpegPCM starts the ffmpeg process with PCM output
func (ps *PCMStationStream) startFFmpegPCM(streamURL, authToken string) error {
	ctx, cancel := context.WithCancel(context.Background())
	ps.cancel = cancel

	// Output PCM format: s16le, 48kHz, stereo
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "10",
		"-timeout", "30000000",
		"-headers", fmt.Sprintf("X-Radiko-AuthToken: %s\r\n", authToken),
		"-i", streamURL,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-fflags", "+nobuffer+flush_packets",
		"-flags", "low_delay",
		"-loglevel", "error",
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

	ps.cmd = cmd
	ps.running = true

	// Log ffmpeg errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("ffmpeg-pcm [%s]: %s", ps.stationID, scanner.Text())
		}
	}()

	// Read from ffmpeg and broadcast to clients
	go ps.readAndBroadcast(stdout)

	// Broadcast to clients
	go ps.broadcastLoop()

	log.Printf("▶ PCM ffmpeg開始: %s", ps.stationID)
	return nil
}

// readAndBroadcast reads from ffmpeg stdout and sends to broadcast channel
func (ps *PCMStationStream) readAndBroadcast(stdout io.Reader) {
	reader := bufio.NewReaderSize(stdout, 32768)
	// PCM frame size: 2 bytes per sample * 2 channels = 4 bytes per frame
	const frameSize = 4
	buf := make([]byte, 8192)
	residue := make([]byte, 0, frameSize) // Buffer for incomplete frames
	firstData := true

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if firstData {
				log.Printf("📦 PCM最初のデータ受信: %s", ps.stationID)
				firstData = false
			}

			// Combine residue from previous read with new data
			var dataToSend []byte
			if len(residue) > 0 {
				dataToSend = make([]byte, len(residue)+n)
				copy(dataToSend, residue)
				copy(dataToSend[len(residue):], buf[:n])
				residue = residue[:0]
			} else {
				dataToSend = buf[:n]
			}

			// Ensure we only send frame-aligned data (multiple of 4 bytes)
			alignedLen := (len(dataToSend) / frameSize) * frameSize
			if alignedLen < len(dataToSend) {
				// Save incomplete frame for next iteration
				residue = append(residue, dataToSend[alignedLen:]...)
			}

			if alignedLen > 0 {
				// Copy aligned data to avoid race conditions
				data := make([]byte, alignedLen)
				copy(data, dataToSend[:alignedLen])

				// Non-blocking send to broadcast channel
				select {
				case ps.broadcast <- data:
				default:
					// Channel full, drop oldest data
					select {
					case <-ps.broadcast:
					default:
					}
					ps.broadcast <- data
				}
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("❌ PCM ffmpeg読み取りエラー [%s]: %v", ps.stationID, err)
			}
			break
		}
	}

	ps.mu.Lock()
	ps.running = false
	ps.mu.Unlock()

	close(ps.broadcast)
	log.Printf("⏹ PCM ffmpeg終了: %s", ps.stationID)
}

// broadcastLoop sends data to all connected clients
func (ps *PCMStationStream) broadcastLoop() {
	for data := range ps.broadcast {
		ps.mu.RLock()
		clients := make([]*Client, 0, len(ps.clients))
		for _, c := range ps.clients {
			clients = append(clients, c)
		}
		ps.mu.RUnlock()

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

// AddClient adds a client to this PCM stream
func (ps *PCMStationStream) AddClient(ctx context.Context, w http.ResponseWriter, clientID string) error {
	client := &Client{
		id:     clientID,
		writer: w,
		done:   make(chan struct{}),
	}

	ps.mu.Lock()
	ps.clients[clientID] = client
	clientCount := len(ps.clients)
	ps.mu.Unlock()

	log.Printf("📊 PCMクライアント追加 [%s]: %d 接続中", ps.stationID, clientCount)

	// Wait for client disconnect or stream end
	select {
	case <-ctx.Done():
		// Client disconnected
	case <-client.done:
		// Write error occurred
	}

	ps.removeClient(clientID)
	return nil
}

// removeClient removes a client from this stream
func (ps *PCMStationStream) removeClient(clientID string) {
	ps.mu.Lock()
	delete(ps.clients, clientID)
	clientCount := len(ps.clients)
	ps.mu.Unlock()

	log.Printf("📊 PCMクライアント削除 [%s]: %d 接続中", ps.stationID, clientCount)

	// If no clients left, start grace period
	if clientCount == 0 {
		ps.startGracePeriod()
	}
}

// startGracePeriod starts the grace period timer
func (ps *PCMStationStream) startGracePeriod() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.graceTimer != nil {
		return // Already running
	}

	log.Printf("⏰ PCM猶予期間開始 [%s]: %d秒", ps.stationID, ps.graceSeconds)

	ps.graceTimer = time.AfterFunc(time.Duration(ps.graceSeconds)*time.Second, func() {
		ps.mu.Lock()
		clientCount := len(ps.clients)
		ps.mu.Unlock()

		if clientCount == 0 {
			log.Printf("⏰ PCM猶予期間終了、ffmpeg停止: %s", ps.stationID)
			ps.Stop()
		}
	})
}

// CancelGracePeriod cancels the grace period timer
func (ps *PCMStationStream) CancelGracePeriod() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.graceTimer != nil {
		ps.graceTimer.Stop()
		ps.graceTimer = nil
		log.Printf("⏰ PCM猶予期間キャンセル: %s", ps.stationID)
	}
}

// Stop stops the ffmpeg process and cleans up
func (ps *PCMStationStream) Stop() {
	ps.mu.Lock()
	if ps.cancel != nil {
		ps.cancel()
	}
	ps.running = false
	ps.mu.Unlock()

	if ps.cmd != nil {
		ps.cmd.Wait()
	}

	if ps.onClose != nil {
		ps.onClose()
	}
}
