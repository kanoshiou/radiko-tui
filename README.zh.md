# Radiko JP Player

[English](README.md) | [日本語](README.ja.md) | **[中文](README.zh.md)**

一个用 Go 语言编写的 Radiko 日本网络电台播放器，带有交互式 TUI 界面。

[![Release](https://img.shields.io/github/v/release/kanoshiou/radiko-tui)](https://github.com/kanoshiou/radiko-tui/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kanoshiou/radiko-tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/kanoshiou/radiko-tui)](LICENSE)

## ✨ 功能特性

- 🎵 实时播放 Radiko 电台
- 🗾 支持日本全部 47 个都道府县
- 🖥️ 交互式终端界面 (TUI)
- 🔊 音量控制，支持静音
- 🔄 流媒体中断时自动重连
- 💾 记住上次播放的电台和设置
- 🌏 跨平台支持 (Windows/Linux/macOS)

## 📸 界面预览

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

## ⚠️ 依赖要求

音频解码需要 **ffmpeg**。

```bash
# Windows (Chocolatey)
choco install ffmpeg

# Linux (Ubuntu/Debian)
sudo apt install ffmpeg

# macOS (Homebrew)
brew install ffmpeg
```

## 🚀 使用方法

```bash
./radiko
```

### 快捷键

| 按键 | 功能 |
|-----|--------|
| ↑/↓ 或 k/j | 选择电台 |
| ←/→ 或 h/l | 切换地区 |
| Enter/空格 | 播放电台 |
| +/- | 调节音量 |
| 0-9 | 设置音量级别 |
| m | 静音切换 |
| r | 重新连接 |
| Esc | 退出 |

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
