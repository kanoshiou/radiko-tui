# Radiko TUI

[English](README.md) | [日本語](README.ja.md) | **[中文](README.zh.md)**

一个用 Go 语言编写的 Radiko 日本网络电台终端用户界面（TUI）播放器。

[![Release](https://img.shields.io/github/v/release/kanoshiou/radiko-tui)](https://github.com/kanoshiou/radiko-tui/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kanoshiou/radiko-tui)](https://go.dev/)
[![Docker](https://img.shields.io/badge/docker-ghcr.io-blue)](https://github.com/kanoshiou/radiko-tui/pkgs/container/radiko-tui)
[![License](https://img.shields.io/github/license/kanoshiou/radiko-tui)](LICENSE)

## ✨ 功能特性

- 🎵 实时播放 Radiko 电台
- 🗾 支持日本全部 47 个都道府县
- 🖥️ 交互式终端界面 (TUI)
- 🌐 服务器模式支持 HTTP 流媒体（AAC/PCM）
- 🔌 客户端模式，连接远程服务器（无需本地 ffmpeg）
- 🔊 音量控制，支持静音
- ⏺️ 录制流媒体为 AAC 文件
- 🔄 流媒体中断时自动重连
- 💾 记住上次播放的电台和设置
- 🌏 跨平台支持 (Windows/Linux/macOS)

## 📸 界面预览

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

## 📦 安装

### 下载预编译版本（推荐）

从 [Releases](https://github.com/kanoshiou/radiko-tui/releases) 页面下载。

### 从源码编译

```bash
git clone https://github.com/kanoshiou/radiko-tui.git
cd radiko-tui
go mod tidy
go build -o radiko
```

### 纯服务器构建（无音频依赖）

对于无音频支持的 Linux 服务器：

```bash
go build -tags noaudio -o radiko-server
```

此构建排除音频播放依赖（oto），仅支持服务器模式（`-server` 参数）。

### 预设服务器地址构建

可以在编译时指定默认服务器地址，这样客户端运行时无需每次都输入 `-server-url`：

```bash
go build -ldflags "-X main.defaultServerURL=http://your-server-ip:8080" -o radiko-tui
```

注意：运行时如果指定了 `--server-url` 参数，其优先级高于编译时预设的地址。

### Docker（推荐用于服务器模式）

从 GitHub Container Registry 拉取：

```bash
docker pull ghcr.io/kanoshiou/radiko-tui:latest
```

使用 Docker 运行：

```bash
docker run -d --name radiko -p 8080:8080 ghcr.io/kanoshiou/radiko-tui:latest
```

自定义端口和保留时间：

```bash
docker run -d --name radiko -p 9000:9000 ghcr.io/kanoshiou/radiko-tui:latest \
  -server -port 9000 -grace 60
```

Docker Compose：

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

## ⚠️ 依赖要求

音频解码和录音需要 **ffmpeg**。

```bash
# Windows (Chocolatey)
choco install ffmpeg

# Linux (Ubuntu/Debian)
sudo apt install ffmpeg

# macOS (Homebrew)
brew install ffmpeg
```

## 🚀 使用方法

### TUI 模式（默认）

```bash
./radiko-tui
```

### 客户端模式（无需 ffmpeg）

连接到运行中的 radiko-tui 服务器：

```bash
./radiko-tui -server-url http://192.168.1.100:8080
```

此模式下，音频解码在 TUI 内部处理，**客户端无需安装 ffmpeg**。所有 TUI 功能（音量、地域切换等）均可使用。

### 服务器模式

作为 HTTP 流媒体服务器运行：

```bash
./radiko-tui -server -port 8080
```

然后在 VLC 或其他播放器中播放：

```bash
vlc http://localhost:8080/api/play/QRR
```

#### 服务器模式特性

- **多客户端支持**：多个客户端可以收听同一电台，共享一个 ffmpeg 实例
- **智能 ffmpeg 复用**：客户端断开后，ffmpeg 会保持运行一段时间（默认 10 秒）
- **自动重连**：如果客户端在保留期内重连，可立即复用现有流

#### 服务器选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `-port` | 8080 | HTTP 服务器端口 |
| `-grace` | 10 | 最后一个客户端断开后保持 ffmpeg 运行的秒数 |

自定义保留时间示例：

```bash
./radiko-tui -server -port 8080 -grace 30
```

#### 服务器 API 端点

| 端点                              | 说明                                |
|---------------------------------|-----------------------------------|
| `GET /api/play/{stationID}`     | 流式传输指定电台 (AAC)，适用于 VLC/浏览器        |
| `GET /api/play/{stationID}/pcm` | 流式传输指定电台 (PCM)，适用于 radiko-tui 客户端 |
| `GET /api/status`               | 获取活动流的 JSON 状态                    |

### 快捷键

| 按键 | 功能 |
|-----|--------|
| ↑/↓ 或 k/j | 选择电台 |
| ←/→ 或 h/l | 切换地区 |
| Enter/空格 | 播放电台 |
| +/- | 调节音量 |
| 0-9 | 设置音量级别 |
| m | 静音切换 |
| s | 开始/停止录音 |
| r | 重新连接 |
| Esc | 退出 |

### 录音功能

按 `s` 键可以开始/停止录制当前播放的流媒体。录音文件会保存到下载文件夹，文件名格式为：`radiko_电台名_YYYYMMDD_HHMMSS.aac`

当录制的电台与当前播放的电台不同时，电台名会显示在括号中：`⏺ 録音中[电台名] MM:SS`

## 📖 文档

- [安装指南](docs/INSTALL.md)
- [使用说明](docs/USAGE.md)
- [故障排除](docs/TROUBLESHOOTING.md)
- [架构说明](docs/ARCHITECTURE.md)

## 🏗️ 技术栈

- **TUI**: [bubbletea](https://github.com/charmbracelet/bubbletea)
- **音频**: [oto](https://github.com/ebitengine/oto) + ffmpeg
- **样式**: [lipgloss](https://github.com/charmbracelet/lipgloss)

## 🙏 特别感谢

特别感谢 [rajiko](https://github.com/jackyzy823/rajiko) 提供的灵感和参考。

## 📋 系统要求

- ffmpeg（运行时必需）
- Go 1.18+（仅编译时需要）
- 支持 UTF-8 的终端

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE)

---

**注意**: 本项目仅供学习和个人使用。请遵守 Radiko 的使用条款。
