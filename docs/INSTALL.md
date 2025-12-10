# Installation Guide

## System Requirements

- **Operating System**: Windows 10+, Linux, macOS
- **ffmpeg**: Required (for audio decoding)
- **Go 1.18+**: Only needed if building from source

## Quick Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Go to [Releases](https://github.com/kanoshiou/radiko-tui/releases)
2. Download the appropriate file for your OS:
   - Windows: `radiko-tui-windows-amd64.exe`
   - Linux: `radiko-tui-linux-amd64`
   - macOS: `radiko-tui-darwin-amd64`
3. Make it executable (Linux/macOS): `chmod +x radiko-tui-*`
4. Run the program

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/kanoshiou/radiko-tui.git
cd radiko-tui

# Install dependencies
go mod tidy

# Build
go build -o radiko

# Run
./radiko
```

## Installing ffmpeg (Required)

### Windows

**Using Chocolatey:**
```powershell
choco install ffmpeg
```

**Using Scoop:**
```powershell
scoop install ffmpeg
```

**Manual installation:**
1. Download from https://ffmpeg.org/download.html
2. Extract to a folder (e.g., `C:\ffmpeg`)
3. Add to PATH: `C:\ffmpeg\bin`

### Linux

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install ffmpeg
```

**Fedora:**
```bash
sudo dnf install ffmpeg
```

**Arch Linux:**
```bash
sudo pacman -S ffmpeg
```

### macOS

**Using Homebrew:**
```bash
brew install ffmpeg
```

### Verify Installation

```bash
ffmpeg -version
```

You should see version information if installed correctly.

## Running the Program

```bash
# Simple run
./radiko

# With custom initial volume (0-100)
./radiko -volume 50
```

## Troubleshooting Installation

### "ffmpeg not found" error
- Ensure ffmpeg is installed and in your system PATH
- Restart your terminal after installing ffmpeg
- On Windows, you may need to restart your computer

### Build errors
```bash
# Clean module cache and retry
go clean -modcache
go mod tidy
go build
```

### Permission denied (Linux/macOS)
```bash
chmod +x ./radiko
```

---

**Need more help?** See [Troubleshooting](TROUBLESHOOTING.md)
