# Architecture

## Project Structure

```
radikojp/
├── .github/
│   ├── dependabot.yml            # Dependabot configuration
│   └── workflows/
│       ├── go-version-check.yml  # Go version compatibility check
│       └── release.yml           # GitHub Actions auto-release
├── api/
│   ├── auth.go                   # Radiko authentication module
│   └── client.go                 # Radiko API client
├── config/
│   └── config.go                 # Configuration management
├── docs/                         # Documentation directory
│   ├── ARCHITECTURE.md           # Architecture (this file)
│   ├── INSTALL.md                # Installation guide
│   ├── TROUBLESHOOTING.md        # Troubleshooting
│   └── USAGE.md                  # Usage guide
├── model/
│   ├── authtoken.go              # Auth token model
│   ├── device.go                 # Device info and GPS generation
│   ├── program.go                # Program data models
│   ├── region.go                 # Region/Area definitions
│   └── station.go                # Station data models
├── player/
│   └── ffmpeg_player.go          # FFmpeg-based audio player
├── tui/
│   └── tui.go                    # Terminal UI (bubbletea)
├── main.go                       # Main program entry
├── config.example.go             # Configuration example
├── go.mod                        # Go module definition
├── go.sum                        # Go dependencies checksum
├── Makefile                      # Build script
├── README.md                     # Project description (English)
├── README.ja.md                  # Project description (Japanese)
├── README.zh.md                  # Project description (Chinese)
└── LICENSE                       # MIT License
```


## Technical Architecture

```
┌─────────────────────────────────────────────────────┐
│              Radiko JP Player                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────┐    ┌──────────────┐              │
│  │   TUI        │◄──►│   Config     │              │
│  │  (bubbletea) │    │   Manager    │              │
│  └──────────────┘    └──────────────┘              │
│         │                                           │
│         ▼                                           │
│  ┌──────────────┐    ┌──────────────┐              │
│  │ Auth Module  │───►│    Token     │              │
│  │  (auth1/2)   │    │   Manager    │              │
│  └──────────────┘    └──────────────┘              │
│         │                                           │
│         ▼                                           │
│  ┌──────────────┐    ┌──────────────┐              │
│  │  API Client  │───►│   Stream     │              │
│  │ (stations)   │    │   Selector   │              │
│  └──────────────┘    └──────────────┘              │
│         │                                           │
│         ▼                                           │
│  ┌──────────────┐    ┌──────────────┐              │
│  │   ffmpeg     │───►│  AAC→PCM    │              │
│  │  (external)  │    │   Decoder    │              │
│  └──────────────┘    └──────────────┘              │
│         │                                           │
│         ▼                                           │
│  ┌──────────────┐    ┌──────────────┐              │
│  │  oto Player  │───►│   Speaker    │              │
│  │    (Go)      │    │              │              │
│  └──────────────┘    └──────────────┘              │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## Core Modules

### 1. API Module (api/)

#### Authentication (api/auth.go)
Handles Radiko's two-step authentication:
- **auth1**: Obtains initial token, key offset, and length
- **auth2**: Validates token with partial key and GPS location
- Supports all 47 Japanese prefectures via GPS spoofing

#### API Client (api/client.go)
Communicates with Radiko services:
- `GetStations()`: Fetches station list for a region
- `GetStreamURLs()`: Gets streaming URLs for a station
- `GetCurrentProgram()`: Retrieves current program info

### 2. Player Module (player/ffmpeg_player.go)

FFmpeg-based audio player with:
- Real-time AAC to PCM decoding
- Software volume control
- Mute functionality
- Auto-reconnection on stream failure
- Reconnection status tracking

### 3. TUI Module (tui/tui.go)

Interactive terminal interface using bubbletea:
- Station list with scroll support
- Region selector (47 prefectures)
- Real-time volume display
- Current program display
- Keyboard navigation

### 4. Configuration (config/config.go)

Persistent user preferences:
- Last played station
- Volume level
- Selected region
- Auto-saved on changes

### 5. Region/Device Models (model/)

- **region.go**: All 47 Japanese prefectures with IDs
- **device.go**: Random Android device generation for auth
- **program.go**: Program schedule data structures

## Data Flow

```
User Input (TUI)
    ↓
Region Selection → Auth Module (GPS spoofing)
    ↓
API Client → Station List
    ↓
User selects station
    ↓
API Client → Stream URLs
    ↓
FFmpeg Player ← HLS Stream (with auth token)
    ↓
AAC Decoding → PCM
    ↓
oto Library → Audio Output
```

## Dependencies

### Go Dependencies
- `github.com/charmbracelet/bubbletea`: TUI framework
- `github.com/charmbracelet/lipgloss`: Terminal styling
- `github.com/charmbracelet/bubbles`: UI components
- `github.com/ebitengine/oto/v3`: Audio output

### External Dependencies
- `ffmpeg`: AAC audio decoding (required at runtime)

## Concurrency Model

- **Main goroutine**: TUI event loop
- **Audio pump goroutine**: Reads PCM from ffmpeg, writes to oto
- **Monitor goroutine**: Detects stream failures, triggers reconnect
- **ffmpeg process**: External process, communicates via stdout pipe

## Error Handling

- **Authentication failure**: Displays error in TUI, allows retry
- **Network error**: Auto-reconnects with new auth token
- **ffmpeg error**: Cleans up resources, shows error message
- **User interrupt**: Gracefully stops player and exits

## Configuration Storage

Config file location:
- **Windows**: `%APPDATA%\radikojp\config.json`
- **Linux/macOS**: `~/.config/radikojp/config.json`

## Performance

- **Memory**: ~20-30MB
- **CPU**: 5-15% (including ffmpeg)
- **Network**: ~128kbps (AAC stream)
- **Startup**: 2-3 seconds (auth + initial stream)
