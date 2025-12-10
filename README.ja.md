# Radiko JP Player

[English](README.md) | **[日本語](README.ja.md)** | [中文](README.zh.md)

インタラクティブなTUIを備えた、Goで書かれたRadiko日本インターネットラジオプレーヤーです。

[![Release](https://img.shields.io/github/v/release/kanoshiou/radiko-tui)](https://github.com/kanoshiou/radiko-tui/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/kanoshiou/radiko-tui)](https://go.dev/)
[![License](https://img.shields.io/github/license/kanoshiou/radiko-tui)](LICENSE)

## ✨ 機能

- 🎵 Radikoラジオ局のライブストリーミング
- 🗾 日本全国47都道府県対応
- 🖥️ インタラクティブなターミナルUI（TUI）
- 🔊 ミュート機能付き音量調整
- 🔄 ストリーム障害時の自動再接続
- 💾 前回の放送局と設定を記憶
- 🌏 クロスプラットフォーム（Windows/Linux/macOS）

## 📸 スクリーンショット

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

## 📦 インストール

### ビルド済みバイナリのダウンロード（推奨）

[Releases](https://github.com/kanoshiou/radiko-tui/releases) からダウンロードしてください。

### ソースからビルド

```bash
git clone https://github.com/kanoshiou/radiko-tui.git
cd radiko-tui
go mod tidy
go build -o radiko
```

## ⚠️ 必要条件

音声デコードには **ffmpeg が必要** です。

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

### 操作方法

| キー | 操作 |
|-----|--------|
| ↑/↓ または k/j | 放送局を選択 |
| ←/→ または h/l | 地域を切り替え |
| Enter/Space | 放送局を再生 |
| +/- | 音量調整 |
| 0-9 | 音量レベルを設定 |
| m | ミュート切り替え |
| r | 再接続 |
| Esc | 終了 |

## 📖 ドキュメント

- [インストールガイド](docs/INSTALL.md)
- [使用ガイド](docs/USAGE.md)
- [トラブルシューティング](docs/TROUBLESHOOTING.md)
- [アーキテクチャ](docs/ARCHITECTURE.md)

## 🏗️ 技術スタック

- **TUI**: [bubbletea](https://github.com/charmbracelet/bubbletea)
- **オーディオ**: [oto](https://github.com/ebitengine/oto) + ffmpeg
- **スタイリング**: [lipgloss](https://github.com/charmbracelet/lipgloss)

## 🙏 謝辞

インスピレーションと参考のために [rajiko](https://github.com/jackyzy823/rajiko) に感謝します。

## 📋 システム要件

- ffmpeg（実行時）
- Go 1.18+（ビルド時のみ）
- UTF-8対応ターミナル

## 🤝 貢献

IssueおよびPull Requestを歓迎します！

## 📄 ライセンス

MITライセンス - [LICENSE](LICENSE) を参照

---

**注意**: このプロジェクトは学習および個人使用のみを目的としています。Radikoの利用規約を遵守してください。
