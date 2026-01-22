# Radiko TUI

**[English](README.md)** | [日本語](README.ja.md) | [中文](README.zh.md)

A Terminal User Interface (TUI) for streaming Radiko Japanese internet radio, written in Go.

[![Release](https://img.shields.io/github/v/release/kanoshiou/radiko-tui)](https://github.com/kanoshiou/radiko-tui/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kanoshiou/radiko-tui)](https://go.dev/)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-blue)](https://github.com/kanoshiou/radiko-tui/pkgs/container/radiko-tui)
[![License](https://img.shields.io/github/license/kanoshiou/radiko-tui)](LICENSE)

## ✨ Features

- 🎵 Stream live Radiko radio stations
- 🗾 Support for all 47 Japanese prefectures
- 🖥️ Interactive terminal UI (TUI)
- 🌐 Server mode for HTTP streaming (AAC/PCM)
- 🔌 Client mode to connect to remote server (no local ffmpeg)
- 🔊 Volume control with mute support
- ⏺️ Record streams to AAC files
- 🔄 Auto-reconnect on stream failure
- 💾 Remembers last station and settings
- 🌏 Cross-platform (Windows/Linux/macOS)

## 📸 Screenshot

```
📻 Radiko  🔊 80%
  ◀ 埼玉 千葉 [東京] 神奈川 新潟 ▶ [13/47]
──────────────────────────────────────────────
  TBSラジオ TBS
 ▶ 文化放送 QRR 
  ニッポン放送 LFR
  ラジオNIKKEI第1 RN1
  ラジオNIKKEI第2 RN2
  ↓ さらに表示

──────────────────────────────────────────────
▶ 文化放送 QRR  ♪ 大竹まことゴールデンラジオ  ⏺ 録音中 02:15
↑↓ 選択  Enter 再生  ←→ 地域切替  +- 音量  m ミュート  s 停止  r 再接続  Esc 終了
```

## 📦 Installation

### Download Pre-built Binary (Recommended)

Download from [Releases](https://github.com/kanoshiou/radiko-tui/releases).

### Build from Source

```bash
git clone https://github.com/kanoshiou/radiko-tui.git
cd radiko-tui
go mod tidy
go build -o radiko
```

### Server-Only Build (No Audio Dependencies)

For headless Linux servers without audio support:

```bash
go build -tags noaudio -o radiko-server
```

This build excludes audio playback dependencies (oto) and only supports server mode (`-server` flag).

### Build with Default Server URL

You can specify a default server URL at build time, so clients don't need to provide `-server-url` every time:

```bash
go build -ldflags "-X main.defaultServerURL=http://your-server-ip:8080" -o radiko-tui
```

Note: If `--server-url` is provided at runtime, it takes precedence over the built-in default.

### Docker (Recommended for Server Mode)

Pull from GitHub Container Registry:

```bash
docker pull ghcr.io/kanoshiou/radiko-tui:latest
```

Run with Docker:

```bash
docker run -d --name radiko -p 8080:8080 ghcr.io/kanoshiou/radiko-tui:latest
```

Custom port and grace period:

```bash
docker run -d --name radiko -p 9000:9000 ghcr.io/kanoshiou/radiko-tui:latest \
  -server -port 9000 -grace 60
```

Docker Compose:

```yaml
version: '3.8'
services:
  radiko:
    image: ghcr.io/kanoshiou/radiko-tui:latest
    ports:
      - "8080:8080"
    environment:
      - TZ=Asia/Tokyo
    restart: unless-stopped
```

## ⚠️ Requirements

**ffmpeg is required** for audio decoding and recording.

```bash
# Windows (Chocolatey)
choco install ffmpeg

# Linux (Ubuntu/Debian)
sudo apt install ffmpeg

# macOS (Homebrew)
brew install ffmpeg
```

## 🚀 Usage

### TUI Mode (Default)

```bash
./radiko-tui
```

### Client Mode (No ffmpeg required)

Connect to a running radiko-tui server:

```bash
./radiko-tui -server-url http://192.168.1.100:8080
```

In this mode, audio decoding is handled internally. **No local ffmpeg installation is required on the client.** All TUI
features (volume, region switching) are supported.

### Server Mode

Run as an HTTP streaming server:

```bash
./radiko-tui -server -port 8080
```

Then stream in VLC or any audio player:

```bash
vlc http://localhost:8080/api/play/QRR
```

#### Server Mode Features

- **Multi-client support**: Multiple clients can listen to the same station, sharing one ffmpeg instance
- **Smart ffmpeg reuse**: When a client disconnects, ffmpeg keeps running for a grace period (default 10 seconds)
- **Automatic reconnection**: If a client reconnects within the grace period, the existing stream is reused instantly

#### Server Options

| Option | Default | Description |
|--------|---------|-------------|
| `-port` | 8080 | HTTP server port |
| `-grace` | 10 | Seconds to keep ffmpeg alive after last client disconnects |

Example with custom grace period:

```bash
./radiko-tui -server -port 8080 -grace 30
```

#### Server API Endpoints

| Endpoint                        | Description                              |
|---------------------------------|------------------------------------------|
| `GET /api/play/{stationID}`     | Stream audio (AAC) for VLC/Browser       |
| `GET /api/play/{stationID}/pcm` | Stream audio (PCM) for radiko-tui client |
| `GET /api/status`               | Get JSON status of active streams        |

### Controls

| Key | Action |
|-----|--------|
| ↑/↓ or k/j | Navigate stations |
| ←/→ or h/l | Switch regions |
| Enter/Space | Play station |
| +/- | Volume up/down |
| 0-9 | Set volume level |
| m | Toggle mute |
| s | Start/Stop recording |
| r | Reconnect |
| Esc | Exit |

### Recording

Press `s` to start/stop recording the current stream. Recordings are saved to your Downloads folder as AAC files with the format: `radiko_StationName_YYYYMMDD_HHMMSS.aac`

When recording a different station than currently playing, the station name will be shown in brackets: `⏺ 録音中[StationName] MM:SS`

## 📖 Documentation

- [Installation Guide](docs/INSTALL.md)
- [Usage Guide](docs/USAGE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Architecture](docs/ARCHITECTURE.md)

## 🏗️ Tech Stack

- **TUI**: [bubbletea](https://github.com/charmbracelet/bubbletea)
- **Audio**: [oto](https://github.com/ebitengine/oto) + ffmpeg
- **Styling**: [lipgloss](https://github.com/charmbracelet/lipgloss)

## 🙏 Special Thanks

Special thanks to [rajiko](https://github.com/jackyzy823/rajiko) for inspiration and reference.

## 📋 System Requirements

- ffmpeg (runtime)
- Go 1.18+ (build only)
- Terminal with UTF-8 support

## 🤝 Contributing

Issues and Pull Requests are welcome!

## 📄 License

MIT License - See [LICENSE](LICENSE)

---

**Note**: This project is for learning and personal use. Please comply with Radiko's terms of service.
