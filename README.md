# Radiko JP Player

**[English](README.md)** | [日本語](README.ja.md) | [中文](README.zh.md)

A Radiko Japanese internet radio player written in Go with an interactive TUI.

[![Release](https://img.shields.io/github/v/release/kanoshiou/radiko-tui)](https://github.com/kanoshiou/radiko-tui/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kanoshiou/radiko-tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/kanoshiou/radiko-tui)](LICENSE)

## ✨ Features

- 🎵 Stream live Radiko radio stations
- 🗾 Support for all 47 Japanese prefectures
- 🖥️ Interactive terminal UI (TUI)
- 🔊 Volume control with mute support
- 🔄 Auto-reconnect on stream failure
- 💾 Remembers last station and settings
- 🌏 Cross-platform (Windows/Linux/macOS)

## 📸 Screenshot

```
📻 Radiko  🔊 80%
  北海道 青森 岩手 [東京] 神奈川  [13/47]
──────────────────────────────────────────────
  TBSラジオ TBS
▶ 文化放送 QRR
  ニッポン放送 LFR
──────────────────────────────────────────────
▶ 文化放送 QRR  ♪ 大竹まことゴールデンラジオ
↑↓ 選択  Enter 再生  ←→ 地域切替  Esc 終了
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

## ⚠️ Requirements

**ffmpeg is required** for audio decoding.

```bash
# Windows (Chocolatey)
choco install ffmpeg

# Linux (Ubuntu/Debian)
sudo apt install ffmpeg

# macOS (Homebrew)
brew install ffmpeg
```

## 🚀 Usage

```bash
./radiko
```

### Controls

| Key | Action |
|-----|--------|
| ↑/↓ or k/j | Navigate stations |
| ←/→ or h/l | Switch regions |
| Enter/Space | Play station |
| +/- | Volume up/down |
| 0-9 | Set volume level |
| m | Toggle mute |
| r | Reconnect |
| Esc | Exit |

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
