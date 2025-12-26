package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"

	"radiko-tui/api"
	"radiko-tui/model"
)

// Server represents the HTTP streaming server
type Server struct {
	port int
	mu   sync.Mutex
}

// NewServer creates a new streaming server
func NewServer(port int) *Server {
	return &Server{
		port: port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/play/{stationID}", s.handlePlay)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("📡 サーバーを開始しました: http://localhost%s", addr)
	log.Printf("   使用例: vlc http://localhost%s/api/play/QRR", addr)

	return http.ListenAndServe(addr, mux)
}

// handlePlay handles the /api/play/{stationID} endpoint
func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request) {
	stationID := r.PathValue("stationID")
	if stationID == "" {
		http.Error(w, "stationID is required", http.StatusBadRequest)
		return
	}

	log.Printf("🎵 再生リクエスト受信: %s", stationID)

	// Get area for this station
	areaID, err := api.GetStationArea(stationID)
	if err != nil {
		log.Printf("❌ エリア取得失敗: %v", err)
		http.Error(w, fmt.Sprintf("failed to get station area: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("📍 エリア: %s", areaID)

	// Authenticate with Radiko
	log.Printf("🔐 認証中...")
	authToken := api.Auth(areaID)
	if authToken == "" {
		log.Printf("❌ 認証失敗")
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	log.Printf("✓ 認証成功")

	// Get stream URLs
	playlistURLs, err := api.GetStreamURLs(stationID)
	if err != nil {
		log.Printf("❌ ストリームURL取得失敗: %v", err)
		http.Error(w, fmt.Sprintf("failed to get stream URL: %v", err), http.StatusInternalServerError)
		return
	}
	if len(playlistURLs) == 0 {
		log.Printf("❌ ストリームが見つかりません")
		http.Error(w, "no stream URLs found", http.StatusNotFound)
		return
	}

	// Build final stream URL
	lsid := model.GenLsid()
	lastURL := playlistURLs[len(playlistURLs)-1]
	streamURL := fmt.Sprintf("%s?station_id=%s&l=30&lsid=%s&type=b", lastURL, stationID, lsid)

	log.Printf("▶ ストリーミング開始: %s", stationID)

	// Stream audio - use request context to detect client disconnect
	err = s.streamAudio(r.Context(), w, streamURL, authToken)
	if err != nil {
		log.Printf("❌ ストリーミングエラー: %v", err)
	}

	log.Printf("⏹ クライアント切断、ストリーム終了: %s", stationID)
}

// streamAudio streams audio from Radiko to the HTTP response
func (s *Server) streamAudio(clientCtx context.Context, w http.ResponseWriter, streamURL, authToken string) error {
	// Set comprehensive headers for audio streaming compatibility
	w.Header().Set("Content-Type", "audio/aac")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Accept-Ranges", "none")
	// ICY headers for better media player compatibility
	w.Header().Set("icy-name", "Radiko Stream")
	w.Header().Set("icy-genre", "Radio")
	w.Header().Set("icy-pub", "1")

	// Create a cancellable context for ffmpeg (not tied to client request initially)
	ffmpegCtx, ffmpegCancel := context.WithCancel(context.Background())
	defer ffmpegCancel()

	// Create ffmpeg command with reconnect options for HLS streams
	cmd := exec.CommandContext(ffmpegCtx, "ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-headers", fmt.Sprintf("X-Radiko-AuthToken: %s", authToken),
		"-i", streamURL,
		"-c:a", "copy", // Copy AAC without re-encoding
		"-f", "adts",   // ADTS format for streaming AAC
		"-loglevel", "error",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Goroutine to log ffmpeg errors
	go func() {
		errOutput, _ := io.ReadAll(stderr)
		if len(errOutput) > 0 {
			log.Printf("ffmpeg stderr: %s", string(errOutput))
		}
	}()

	// Goroutine to monitor client disconnect
	go func() {
		<-clientCtx.Done()
		ffmpegCancel() // Cancel ffmpeg when client disconnects
	}()

	// Stream audio to response with larger buffer for stability
	buf := make([]byte, 16384) // Increased buffer size for better streaming
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				// Client disconnected
				ffmpegCancel()
				break
			}

			// Flush immediately to reduce latency
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			// Check if it's due to context cancellation
			select {
			case <-ffmpegCtx.Done():
				// Normal shutdown due to client disconnect
				break
			default:
				return readErr
			}
			break
		}
	}

	// Wait for ffmpeg to exit
	cmd.Wait()
	return nil
}

