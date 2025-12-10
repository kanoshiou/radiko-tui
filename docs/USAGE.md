# Usage Guide

## Starting the Program

```bash
# Run with default settings
./radiko

# Run with specific initial volume (0-100)
./radiko -volume 50
```

## TUI Controls

### Navigation

| Key | Action |
|-----|--------|
| ↑ / k | Move up in station list |
| ↓ / j | Move down in station list |
| ← / h | Switch to previous region |
| → / l | Switch to next region |
| Enter / Space | Play selected station |

### Playback Controls

| Key | Action |
|-----|--------|
| + / = | Increase volume |
| - / _ | Decrease volume |
| 0-9 | Set volume (0=0%, 5=50%, 9=90%) |
| m | Toggle mute |
| r | Reconnect (refresh stream) |

### General

| Key | Action |
|-----|--------|
| Esc | Exit program (or cancel region selection) |
| Ctrl+C | Force quit |

## Interface Layout

```
📻 Radiko  🔊 80%
  北海道 青森 岩手 [東京] 神奈川  [13/47]
──────────────────────────────────────────────
  TBSラジオ TBS
  文化放送 QRR
▶ ニッポン放送 LFR    ← Currently playing
  TOKYO FM FMT
  J-WAVE FMJ
  ↓ さらに表示
──────────────────────────────────────────────
▶ ニッポン放送 LFR  ♪ オールナイトニッポン
↑↓ 選択  Enter 再生  ←→ 地域切替  +- 音量  m ミュート  r 再接続  Esc 終了
```

### UI Elements

- **Header**: Title and current volume
- **Region Bar**: Shows nearby regions, current region highlighted
- **Station List**: Scrollable list of stations
  - `▶` indicates currently playing station
  - Selected station is highlighted
- **Footer**: Now playing info and keyboard shortcuts

## Region Selection

You can switch regions in two ways:

1. **Quick switch**: Press ← / → while in station list
2. **Region selector**: 
   - Press ↑ when at top of station list to enter region mode
   - Use ← / → to navigate regions
   - Press Enter to confirm
   - Press ↓ or Esc to cancel

## Precise Volume Control

For precise volume adjustments, you can enter volume control mode:

1. **Enter volume mode**: 
   - Navigate up from region selector (press ↑ when in region mode)
   - The volume bar will display with a visual indicator

2. **Volume bar display**:
   ```
   ▶ 🔊 [████████████░░░░░░░░]  60%
   ```

3. **Controls in volume mode**:
   - ← : Decrease volume by 1%
   - → : Increase volume by 1%
   - m : Toggle mute
   - ↓ : Return to region selector
   - Esc : Return to station list

## Configuration

The program automatically saves:
- Last played station
- Volume level
- Selected region

Configuration file location:
- **Windows**: `%APPDATA%\radiko-tui\config.json`
- **Linux/macOS**: `~/.config/radiko-tui/config.json`

### Config File Format

```json
{
  "last_station_id": "LFR",
  "volume": 0.8,
  "area_id": "JP13"
}
```

## Auto-Reconnect

The player automatically reconnects when:
- Stream is interrupted for more than 10 seconds
- Network connection is restored

During reconnection, you'll see status updates:
- 🔄 再接続中... (Reconnecting...)
- 🔑 認証取得中... (Getting auth...)
- ▶ 再生を再開中... (Resuming playback...)

## Tips

1. **Quick volume**: Press number keys 0-9 for instant volume levels
2. **Mute toggle**: Press `m` to quickly mute/unmute
3. **Refresh stream**: If audio stutters, press `r` to reconnect
4. **Terminal size**: Resize your terminal for better display

---

**Having issues?** See [Troubleshooting](TROUBLESHOOTING.md)
