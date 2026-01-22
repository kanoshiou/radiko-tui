//go:build !noaudio

package tui

import (
	"fmt"
	"strings"
	"time"

	"radiko-tui/api"
	"radiko-tui/config"
	"radiko-tui/model"
	"radiko-tui/player"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FocusMode represents the current focus area
type FocusMode int

const (
	FocusStations FocusMode = iota
	FocusRegion
	FocusVolume
)

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Select    key.Binding
	VolUp     key.Binding
	VolDown   key.Binding
	Mute      key.Binding
	Reconnect key.Binding
	Record    key.Binding // Defines record key, used as 'Stop' when recording
	Quit      key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.VolUp, k.VolDown, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Select},
		{k.VolUp, k.VolDown, k.Mute, k.Reconnect, k.Quit},
	}
}

var DefaultKeyMap = KeyMap{
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "上へ")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "下へ")),
	Left:      key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←", "左")),
	Right:     key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→", "右")),
	Select:    key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("Enter", "選択")),
	VolUp:     key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "音量+")),
	VolDown:   key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "音量-")),
	Mute:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "ミュート")),
	Reconnect: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "再接続")),
	Record:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "録音/停止")),
	Quit:      key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("Esc", "終了/戻る")),
}

// Styles
var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	textColor      = lipgloss.Color("#CDD6F4")
	dimTextColor   = lipgloss.Color("#6C7086")
	playingColor   = lipgloss.Color("#A6E3A1")
	regionColor    = lipgloss.Color("#89B4FA")
	warningColor   = lipgloss.Color("#FAB387")
	recordingColor = lipgloss.Color("#F38BA8")

	titleStyle                  = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	regionItemStyle             = lipgloss.NewStyle().Foreground(textColor)
	regionSelectedStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#1E1E2E")).Background(regionColor).Bold(true).Padding(0, 1)
	regionCurrentStyle          = lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	stationNameStyle            = lipgloss.NewStyle().Foreground(textColor)
	stationIDStyle              = lipgloss.NewStyle().Foreground(dimTextColor)
	stationSelectedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#1E1E2E")).Background(primaryColor).Bold(true).Padding(0, 1)
	stationPlayingStyle         = lipgloss.NewStyle().Foreground(playingColor).Bold(true)
	stationSelectedPlayingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#1E1E2E")).Background(secondaryColor).Bold(true).Padding(0, 1)
	statusStyle                 = lipgloss.NewStyle().Foreground(dimTextColor)
	errorStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))
	volumeStyle                 = lipgloss.NewStyle().Foreground(accentColor)
	focusIndicatorStyle         = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	programStyle                = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7"))
	nowPlayingStyle             = lipgloss.NewStyle().Foreground(playingColor).Bold(true)
	reconnectStyle              = lipgloss.NewStyle().Foreground(warningColor)
	recordingStyle              = lipgloss.NewStyle().Foreground(recordingColor).Bold(true)
)

// PlayingInfo holds information about the currently playing station
type PlayingInfo struct {
	StationID      string
	StationName    string
	CurrentProgram string
}

// SharedState holds shared state between components
type SharedState struct {
	Player        player.Player
	AuthToken     string
	Volume        float64
	Muted         bool
	CurrentAreaID string
	Playing       *PlayingInfo
	ServerURL     string // If set, we are in client mode
}

// Model is the TUI model
type Model struct {
	stations      []model.Station
	cursor        int
	width         int
	height        int
	keys          KeyMap
	statusMessage string
	errorMessage  string
	shared        *SharedState
	autoPlay      bool
	autoPlayIdx   int

	areas        []model.Area
	currentArea  int
	selectedArea int
	isLoading    bool
	focus        FocusMode
}

// Message types
type autoPlayMsg struct{}
type stationsLoadedMsg struct {
	stations []model.Station
	err      error
}
type playResultMsg struct {
	err         error
	stationIdx  int
	stationID   string
	stationName string
}
type reconnectResultMsg struct{ err error }
type programUpdateMsg struct{ program string }
type tickMsg struct{}

func NewModel(stations []model.Station, authToken string, initialVolume float64, lastStationID string, areaID string, serverURL string) Model {
	areas := model.AllAreas()

	currentAreaIdx := 0
	for i, area := range areas {
		if area.ID == areaID {
			currentAreaIdx = i
			break
		}
	}

	defaultIdx := 0
	autoPlayIdx := -1
	for i, s := range stations {
		if s.ID == lastStationID {
			defaultIdx = i
			autoPlayIdx = i
			break
		}
	}

	if autoPlayIdx == -1 {
		for i, s := range stations {
			if s.ID == "QRR" {
				defaultIdx = i
				autoPlayIdx = i
				break
			}
		}
	}

	if autoPlayIdx == -1 && len(stations) > 0 {
		autoPlayIdx = 0
	}

	var p player.Player
	if serverURL != "" {
		p = player.NewHTTPPlayer(serverURL, initialVolume)
	} else {
		fp := player.NewFFmpegPlayer(authToken, initialVolume)
		// Set reconnect callback to re-authenticate
		// We use a closure that captures the CURRENT shared state area ID
		// Note: shared isn't created yet, so we'll set it later or access via a wrapper
		p = fp
	}

	shared := &SharedState{
		Player:        p,
		AuthToken:     authToken,
		Volume:        initialVolume,
		Muted:         false,
		CurrentAreaID: areaID,
		Playing:       nil,
		ServerURL:     serverURL,
	}

	// Set callback for FFmpegPlayer
	if fp, ok := p.(*player.FFmpegPlayer); ok {
		fp.SetReconnectCallback(func() string {
			return api.Auth(shared.CurrentAreaID)
		})
	}

	return Model{
		stations:      stations,
		cursor:        defaultIdx,
		keys:          DefaultKeyMap,
		statusMessage: "",
		shared:        shared,
		autoPlay:      true,
		autoPlayIdx:   autoPlayIdx,
		areas:         areas,
		currentArea:   currentAreaIdx,
		selectedArea:  currentAreaIdx,
		focus:         FocusStations,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return autoPlayMsg{} },
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func fetchProgramCmd(stationID string) tea.Cmd {
	return func() tea.Msg {
		prog, err := api.GetCurrentProgram(stationID)
		if err != nil || prog == nil {
			return programUpdateMsg{program: ""}
		}
		return programUpdateMsg{program: prog.Title}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Check reconnection status if Player supports it (FFmpegPlayer only currently returns status via interface? No, interface doesn't expose GetReconnectStatus yet, need to check type or add to interface)
		// Wait, I didn't add GetReconnectStatus to Player interface.
		// I should rely on generic behavior or cast.
		// For now, let's just refresh program info.
		// FFmpegPlayer's status monitoring was specific.

		// Refresh program info every 30 seconds
		var cmd tea.Cmd
		if m.shared.Playing != nil && time.Now().Second()%30 == 0 {
			cmd = fetchProgramCmd(m.shared.Playing.StationID)
		}
		return m, tea.Batch(cmd, tickCmd())

	case programUpdateMsg:
		if m.shared.Playing != nil {
			m.shared.Playing.CurrentProgram = msg.program
		}
		return m, nil

	case autoPlayMsg:
		if m.autoPlay && m.autoPlayIdx >= 0 && m.autoPlayIdx < len(m.stations) {
			m.autoPlay = false
			m.cursor = m.autoPlayIdx
			return m, m.playStation()
		}
		return m, nil

	case stationsLoadedMsg:
		m.isLoading = false
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("読み込み失敗: %v", msg.err)
		} else {
			m.stations = msg.stations
			m.shared.CurrentAreaID = m.getCurrentAreaID()
			m.cursor = 0
			m.statusMessage = fmt.Sprintf("%s に切り替えました", m.getCurrentAreaName())
			m.saveAreaConfig()
		}
		return m, nil

	case playResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("再生失敗: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.shared.Playing = &PlayingInfo{
				StationID:   msg.stationID,
				StationName: msg.stationName,
			}
			m.statusMessage = ""
			m.errorMessage = ""
			m.saveConfig()
			return m, fetchProgramCmd(msg.stationID)
		}
		return m, nil

	case reconnectResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("再接続失敗: %v", msg.err)
		} else {
			m.statusMessage = "再接続成功"
		}
		return m, nil

	case tea.KeyMsg:
		if m.isLoading {
			return m, nil
		}
		m.errorMessage = ""
		m.statusMessage = ""

		if m.focus == FocusVolume {
			return m.handleVolumeKeys(msg)
		}
		if m.focus == FocusRegion {
			return m.handleRegionKeys(msg)
		}
		return m.handleStationKeys(msg)
	}

	return m, nil
}

func (m Model) handleStationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.focus = FocusRegion
			m.selectedArea = m.currentArea
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.stations)-1 {
			m.cursor++
		}
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.currentArea > 0 {
			m.currentArea--
			m.selectedArea = m.currentArea
			return m, m.loadStationsForCurrentArea()
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.currentArea < len(m.areas)-1 {
			m.currentArea++
			m.selectedArea = m.currentArea
			return m, m.loadStationsForCurrentArea()
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		return m, m.playStation()

	case key.Matches(msg, m.keys.VolUp):
		if m.shared.Player != nil {
			m.shared.Player.IncreaseVolume(0.05)
			m.shared.Volume = m.shared.Player.GetVolume()
			m.shared.Muted = false
			m.saveConfig()
		}
		return m, nil

	case key.Matches(msg, m.keys.VolDown):
		if m.shared.Player != nil {
			m.shared.Player.DecreaseVolume(0.05)
			m.shared.Volume = m.shared.Player.GetVolume()
			m.shared.Muted = false
			m.saveConfig()
		}
		return m, nil

	case key.Matches(msg, m.keys.Mute):
		if m.shared.Player != nil {
			m.shared.Player.ToggleMute()
			m.shared.Muted = m.shared.Player.IsMuted()
		}
		return m, nil

	case key.Matches(msg, m.keys.Reconnect):
		if m.shared.Player != nil && m.shared.Playing != nil {
			return m, m.reconnect()
		}
		return m, nil

	case key.Matches(msg, m.keys.Record):
		if m.shared.Player != nil && m.shared.Playing != nil {
			started, filePath, err := m.shared.Player.ToggleRecording(m.shared.Playing.StationName)
			if err != nil {
				m.errorMessage = err.Error()
			} else if started {
				m.statusMessage = "録音開始"
			} else {
				m.statusMessage = fmt.Sprintf("録音保存: %s", filePath)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.saveConfig()
		if m.shared.Player != nil {
			// Stop recording if active
			if m.shared.Player.IsRecording() {
				m.shared.Player.StopRecording()
			}
			m.shared.Player.Stop()
		}
		return m, tea.Quit

	case msg.String() >= "0" && msg.String() <= "9":
		if m.shared.Player != nil {
			vol := float64(msg.String()[0]-'0') / 10.0
			m.shared.Player.SetVolume(vol)
			m.shared.Volume = vol
			m.shared.Muted = false
			m.saveConfig()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleRegionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		// Move to volume control when pressing up from region
		m.focus = FocusVolume
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.selectedArea > 0 {
			m.selectedArea--
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.selectedArea < len(m.areas)-1 {
			m.selectedArea++
		}
		return m, nil

	case key.Matches(msg, m.keys.Down), key.Matches(msg, m.keys.Quit):
		m.focus = FocusStations
		m.selectedArea = m.currentArea
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if m.selectedArea != m.currentArea {
			m.currentArea = m.selectedArea
			m.focus = FocusStations
			return m, m.loadStationsForCurrentArea()
		}
		m.focus = FocusStations
		return m, nil
	}
	return m, nil
}

// handleVolumeKeys handles keyboard input when volume control is focused
func (m Model) handleVolumeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Left):
		// Decrease volume by 1%
		if m.shared.Player != nil {
			m.shared.Player.DecreaseVolume(0.01)
			m.shared.Volume = m.shared.Player.GetVolume()
			m.shared.Muted = false
			m.saveConfig()
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		// Increase volume by 1%
		if m.shared.Player != nil {
			m.shared.Player.IncreaseVolume(0.01)
			m.shared.Volume = m.shared.Player.GetVolume()
			m.shared.Muted = false
			m.saveConfig()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		// Move to region selector
		m.focus = FocusRegion
		m.selectedArea = m.currentArea
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		// Exit volume mode to station list
		m.focus = FocusStations
		m.selectedArea = m.currentArea
		return m, nil

	case key.Matches(msg, m.keys.Mute):
		if m.shared.Player != nil {
			m.shared.Player.ToggleMute()
			m.shared.Muted = m.shared.Player.IsMuted()
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		// Confirm and move to station list
		m.focus = FocusStations
		return m, nil
	}
	return m, nil
}

func (m *Model) getCurrentAreaID() string {
	if m.currentArea >= 0 && m.currentArea < len(m.areas) {
		return m.areas[m.currentArea].ID
	}
	return "JP13"
}

func (m *Model) getCurrentAreaName() string {
	if m.currentArea >= 0 && m.currentArea < len(m.areas) {
		return m.areas[m.currentArea].Name
	}
	return "東京"
}

func (m *Model) loadStationsForCurrentArea() tea.Cmd {
	m.isLoading = true
	m.statusMessage = fmt.Sprintf("%s を読み込み中...", m.getCurrentAreaName())
	areaID := m.getCurrentAreaID()
	return func() tea.Msg {
		stations, err := api.GetStations(areaID)
		return stationsLoadedMsg{stations: stations, err: err}
	}
}

func (m *Model) saveConfig() {
	if m.shared.Playing != nil {
		volume := m.shared.Volume
		if m.shared.Player != nil {
			volume = m.shared.Player.GetVolume()
		}
		go config.SaveConfig(m.shared.Playing.StationID, volume, m.getCurrentAreaID())
	}
}

func (m *Model) saveAreaConfig() {
	volume := m.shared.Volume
	if m.shared.Player != nil {
		volume = m.shared.Player.GetVolume()
	}
	stationID := ""
	if m.shared.Playing != nil {
		stationID = m.shared.Playing.StationID
	}
	go config.SaveConfig(stationID, volume, m.getCurrentAreaID())
}

func (m *Model) playStation() tea.Cmd {
	stationIdx := m.cursor
	station := m.stations[stationIdx]
	shared := m.shared
	currentAreaID := m.getCurrentAreaID()

	return func() tea.Msg {
		var playTarget string
		if shared.ServerURL != "" {
			// Client mode: play by station ID directly
			playTarget = station.ID
			shared.Player.Stop()
			time.Sleep(100 * time.Millisecond)
		} else {
			// Local mode: resolve stream URL
			playlistURLs, err := api.GetStreamURLs(station.ID)
			if err != nil {
				return playResultMsg{err: err, stationIdx: stationIdx}
			}
			if len(playlistURLs) == 0 {
				return playResultMsg{err: fmt.Errorf("利用可能なストリームがありません"), stationIdx: stationIdx}
			}

			lsid := model.GenLsid()
			lastUrl := playlistURLs[len(playlistURLs)-1]
			playTarget = fmt.Sprintf("%s?station_id=%s&l=30&lsid=%s&type=b", lastUrl, station.ID, lsid)

			shared.Player.Stop()
			time.Sleep(100 * time.Millisecond)

			// Re-authenticate for the current area to ensure token matches the region
			newToken := api.Auth(currentAreaID)
			if newToken != "" {
				shared.AuthToken = newToken
				// Update auth token for FFmpegPlayer
				if fp, ok := shared.Player.(*player.FFmpegPlayer); ok {
					fp.UpdateAuthToken(newToken)
				}
			}
		}

		err := shared.Player.Play(playTarget)
		return playResultMsg{
			err:         err,
			stationIdx:  stationIdx,
			stationID:   station.ID,
			stationName: station.Name,
		}
	}
}

func (m *Model) reconnect() tea.Cmd {
	shared := m.shared
	return func() tea.Msg {
		if shared.Player != nil {
			return reconnectResultMsg{err: shared.Player.Reconnect()}
		}
		return reconnectResultMsg{err: fmt.Errorf("プレーヤーが初期化されていません")}
	}
}

// View renders the UI - fixed bottom layout
func (m Model) View() string {
	// Calculate available height
	totalHeight := m.height
	if totalHeight == 0 {
		totalHeight = 24 // Default height
	}

	// Fixed region heights
	headerHeight := 3 // Title + Region + Separator
	footerHeight := 3 // Separator + Playing info + Help

	// Content area height
	contentHeight := totalHeight - headerHeight - footerHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content strings.Builder

	// === Header ===
	// Title + Volume
	title := titleStyle.Render("📻 Radiko")
	if m.shared.ServerURL != "" {
		title += statusStyle.Render(" [サーバー接続]")
	}

	volBar := m.renderVolume()
	content.WriteString(fmt.Sprintf("%s  %s\n", title, volBar))

	// Region line
	content.WriteString(m.renderRegionLine() + "\n")
	content.WriteString(strings.Repeat("─", 50) + "\n")

	// === Content area ===
	contentLines := m.renderContent(contentHeight)
	content.WriteString(contentLines)

	// Pad with empty lines to fix bottom position
	currentLines := strings.Count(content.String(), "\n")
	targetLines := totalHeight - footerHeight
	for i := currentLines; i < targetLines; i++ {
		content.WriteString("\n")
	}

	// === Fixed bottom area ===
	content.WriteString(m.renderFooter())

	return content.String()
}

// renderContent renders the content area
func (m Model) renderContent(maxHeight int) string {
	var lines []string

	if m.isLoading {
		lines = append(lines, fmt.Sprintf("⏳ %s", m.statusMessage))
		return strings.Join(lines, "\n") + "\n"
	}

	// Station list
	maxVisible := maxHeight - 2 // Leave space for status messages
	if maxVisible > len(m.stations) {
		maxVisible = len(m.stations)
	}
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.stations) {
		endIdx = len(m.stations)
		startIdx = endIdx - maxVisible
		if startIdx < 0 {
			startIdx = 0
		}
	}

	if startIdx > 0 {
		lines = append(lines, statusStyle.Render("  ↑ さらに表示"))
	}

	for i := startIdx; i < endIdx; i++ {
		station := m.stations[i]
		isSelected := i == m.cursor && m.focus == FocusStations
		isPlaying := m.shared.Playing != nil && m.shared.Playing.StationID == station.ID

		prefix := "  "
		if isPlaying {
			prefix = "▶ "
		}

		var styled string
		switch {
		case isSelected && isPlaying:
			text := fmt.Sprintf("%s%s %s", prefix, station.Name, station.ID)
			styled = stationSelectedPlayingStyle.Render(text)
		case isSelected:
			text := fmt.Sprintf("%s%s %s", prefix, station.Name, station.ID)
			styled = stationSelectedStyle.Render(text)
		case isPlaying:
			styled = stationPlayingStyle.Render(prefix+station.Name) + " " + stationIDStyle.Render(station.ID)
		default:
			styled = stationNameStyle.Render(prefix+station.Name) + " " + stationIDStyle.Render(station.ID)
		}
		lines = append(lines, styled)
	}

	if endIdx < len(m.stations) {
		lines = append(lines, statusStyle.Render("  ↓ さらに表示"))
	}

	// Status/Error messages
	if m.errorMessage != "" {
		lines = append(lines, errorStyle.Render("✗ "+m.errorMessage))
	} else if m.statusMessage != "" {
		lines = append(lines, statusStyle.Render(m.statusMessage))
	}

	return strings.Join(lines, "\n") + "\n"
}

// renderFooter renders the fixed bottom area
func (m Model) renderFooter() string {
	var lines []string

	lines = append(lines, strings.Repeat("─", 50))

	// Playing info + Reconnection status
	var playLine string
	if m.shared.Playing != nil {
		playLine = nowPlayingStyle.Render("▶ ") + m.shared.Playing.StationName + " " + stationIDStyle.Render(m.shared.Playing.StationID)
		if m.shared.Playing.CurrentProgram != "" {
			playLine += "  " + programStyle.Render("♪ "+m.shared.Playing.CurrentProgram)
		}

		// Check status using type assertion for specific details if needed
		// For general status, we trust tickMsg to update m.statusMessage if it was supported
		// But here we want inline status in footer
		if m.shared.Player != nil {
			// Try to get status from FFmpegPlayer
			if fp, ok := m.shared.Player.(*player.FFmpegPlayer); ok {
				switch fp.GetReconnectStatus() {
				case player.ReconnectStarted:
					playLine += "  " + reconnectStyle.Render("🔄 再接続中...")
				case player.ReconnectAuth:
					playLine += "  " + reconnectStyle.Render("🔑 認証取得中...")
				case player.ReconnectPlaying:
					playLine += "  " + reconnectStyle.Render("▶ 再生を再開中...")
				}
			}

			// Check recording status
			if m.shared.Player.IsRecording() {
				_, duration, recordingStation := m.shared.Player.GetRecordingInfo()
				mins := int(duration.Minutes())
				secs := int(duration.Seconds()) % 60
				// Check if recording station is different from playing station
				if m.shared.Playing != nil && recordingStation != m.shared.Playing.StationName {
					playLine += "  " + recordingStyle.Render(fmt.Sprintf("⏺ 録音中[%s] %02d:%02d", recordingStation, mins, secs))
				} else {
					playLine += "  " + recordingStyle.Render(fmt.Sprintf("⏺ 録音中 %02d:%02d", mins, secs))
				}
			}
		}
	} else {
		playLine = statusStyle.Render("再生していません")
	}
	lines = append(lines, playLine)

	// Help - change "s 録音" to "s 停止" when recording
	isRecording := m.shared.Player != nil && m.shared.Player.IsRecording()
	switch m.focus {
	case FocusVolume:
		lines = append(lines, statusStyle.Render("← → 音量調整  m ミュート  ↓ 地域へ  Esc 戻る"))
	case FocusRegion:
		lines = append(lines, statusStyle.Render("← → 選択  Enter 確定  ↑ 音量へ  ↓/Esc 戻る"))
	default:
		if isRecording {
			lines = append(lines, statusStyle.Render("↑↓ 選択  Enter 再生  ←→ 地域切替  +- 音量  m ミュート  ")+recordingStyle.Render("s 停止")+statusStyle.Render("  r 再接続  Esc 終了"))
		} else {
			lines = append(lines, statusStyle.Render("↑↓ 選択  Enter 再生  ←→ 地域切替  +- 音量  m ミュート  s 録音  r 再接続  Esc 終了"))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderVolume() string {
	vol := int(m.shared.Volume * 100)
	if m.shared.Player != nil {
		vol = int(m.shared.Player.GetVolume() * 100)
	}

	// In volume focus mode, show a detailed volume bar
	if m.focus == FocusVolume {
		return m.renderVolumeBar(vol)
	}

	// Normal display
	if m.shared.Muted {
		return statusStyle.Render(fmt.Sprintf("🔇 %d%%", vol))
	}
	return volumeStyle.Render(fmt.Sprintf("🔊 %d%%", vol))
}

// renderVolumeBar renders a detailed volume bar for precise control
func (m Model) renderVolumeBar(vol int) string {
	barWidth := 20
	filled := vol * barWidth / 100

	var bar strings.Builder

	// Focus indicator
	bar.WriteString(focusIndicatorStyle.Render("▶ "))

	// Volume icon
	if m.shared.Muted {
		bar.WriteString(statusStyle.Render("🔇 "))
	} else if vol == 0 {
		bar.WriteString(volumeStyle.Render("🔈 "))
	} else if vol < 50 {
		bar.WriteString(volumeStyle.Render("🔉 "))
	} else {
		bar.WriteString(volumeStyle.Render("🔊 "))
	}

	// Volume bar
	bar.WriteString("[")
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString(volumeStyle.Render("█"))
		} else {
			bar.WriteString(statusStyle.Render("░"))
		}
	}
	bar.WriteString("] ")

	// Percentage
	bar.WriteString(volumeStyle.Render(fmt.Sprintf("%3d%%", vol)))

	return bar.String()
}

func (m Model) renderRegionLine() string {
	var parts []string

	if m.focus == FocusRegion {
		parts = append(parts, focusIndicatorStyle.Render("▶ "))
	} else {
		parts = append(parts, "  ")
	}

	visibleCount := 5
	startIdx := m.selectedArea - visibleCount/2
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleCount
	if endIdx > len(m.areas) {
		endIdx = len(m.areas)
		startIdx = endIdx - visibleCount
		if startIdx < 0 {
			startIdx = 0
		}
	}

	if startIdx > 0 {
		parts = append(parts, statusStyle.Render("◀ "))
	}

	for i := startIdx; i < endIdx; i++ {
		area := m.areas[i]
		var styled string
		if m.focus == FocusRegion && i == m.selectedArea {
			styled = regionSelectedStyle.Render(area.Name)
		} else if i == m.currentArea {
			styled = regionCurrentStyle.Render(area.Name)
		} else {
			styled = regionItemStyle.Render(area.Name)
		}
		parts = append(parts, styled)
		if i < endIdx-1 {
			parts = append(parts, " ")
		}
	}

	if endIdx < len(m.areas) {
		parts = append(parts, statusStyle.Render(" ▶"))
	}

	parts = append(parts, statusStyle.Render(fmt.Sprintf(" [%d/%d]", m.selectedArea+1, len(m.areas))))
	return strings.Join(parts, "")
}

// Run starts the TUI
func Run(stations []model.Station, authToken string, cfg config.Config, serverURL string) error {
	m := NewModel(stations, authToken, cfg.Volume, cfg.LastStationID, cfg.AreaID, serverURL)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()

	if m.shared.Player != nil {
		m.shared.Player.Stop()
	}
	return err
}
