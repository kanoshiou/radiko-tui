# Troubleshooting

## Common Issues

### ffmpeg not found

**Error**: `ffmpeg not found in PATH`

**Solution**:
1. Install ffmpeg (see [Installation Guide](INSTALL.md))
2. Verify installation: `ffmpeg -version`
3. Restart your terminal/computer
4. On Windows, ensure ffmpeg is in your PATH environment variable

### No sound

**Possible causes**:
1. System volume is muted
2. Wrong audio output device selected
3. ffmpeg decoding issue

**Solutions**:
- Check system volume settings
- Press `m` to toggle mute (might be muted in app)
- Press `+` to increase volume
- Try reconnecting with `r`
- Restart the program

### Stream keeps disconnecting

**Possible causes**:
1. Unstable network connection
2. VPN issues

**Solutions**:
- Check your internet connection
- The program auto-reconnects after 5 seconds
- Press `r` to manually reconnect
- Try a different network

### TUI display issues

**Symptoms**: Garbled text, wrong colors, misaligned UI

**Solutions**:
1. Ensure terminal supports UTF-8:
   ```bash
   # Linux/macOS
   export LANG=en_US.UTF-8
   ```
2. Use a modern terminal (Windows Terminal, iTerm2, etc.)
3. Try resizing the terminal window
4. Minimum recommended size: 80x24 characters

### Authentication failed

**Possible causes**:
1. Network blocking Radiko API
2. Radiko service issue

**Solutions**:
- Check if you can access https://radiko.jp in a browser
- Wait a few minutes and try again
- Check your network/firewall settings

### Build errors

**Error**: Go module issues

**Solution**:
```bash
# Clean and reinstall dependencies
go clean -modcache
rm go.sum
go mod tidy
go build
```

**Error**: CGo compilation errors (oto library)

**Solution**:
- **Windows**: Install MinGW-w64 or TDM-GCC
- **Linux**: Install `libasound2-dev` (Debian/Ubuntu) or `alsa-lib-devel` (Fedora)
- **macOS**: Install Xcode command line tools: `xcode-select --install`

### Program crashes on startup

**Possible causes**:
1. Audio device not available
2. Config file corrupted

**Solutions**:
1. Check audio device is connected
2. Delete config file and restart:
   - Windows: `del %APPDATA%\radikojp\config.json`
   - Linux/macOS: `rm ~/.config/radikojp/config.json`

### High CPU usage

**Normal**: 5-15% CPU usage (ffmpeg decoding)

**If higher**:
- Check if multiple ffmpeg processes are running
- Restart the program
- Report issue on GitHub

## Debug Information

To get more information for bug reports:

```bash
# Check ffmpeg version
ffmpeg -version

# Check Go version (if building from source)
go version

# Check system info
# Windows
systeminfo
# Linux
uname -a
# macOS
sw_vers
```

## Getting Help

If you can't solve your issue:

1. Check existing [GitHub Issues](https://github.com/kanoshiou/radikojp/issues)
2. Create a new issue with:
   - OS and version
   - ffmpeg version
   - Error message or screenshot
   - Steps to reproduce
