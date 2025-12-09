package tui

import (
	"fmt"
	"strings"
	"time"

	"radikojp/api"
	"radikojp/config"
	"radikojp/hook"
	"radikojp/model"
	"radikojp/player"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FocusMode 焦点模式
type FocusMode int

const (
	FocusStations FocusMode = iota // 焦点在电台列表
	FocusRegion                    // 焦点在地区选择
)

// KeyMap 定义快捷键
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
	Quit      key.Binding
}

// ShortHelp 返回简短的帮助信息
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.VolUp, k.VolDown, k.Quit}
}

// FullHelp 返回详细帮助信息
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Select},
		{k.VolUp, k.VolDown, k.Mute, k.Reconnect, k.Quit},
	}
}

// 默认快捷键
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "上移"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "下移"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←", "左"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→", "右"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("Enter", "选择"),
	),
	VolUp: key.NewBinding(
		key.WithKeys("+", "="),
		key.WithHelp("+", "音量+"),
	),
	VolDown: key.NewBinding(
		key.WithKeys("-", "_"),
		key.WithHelp("-", "音量-"),
	),
	Mute: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "静音"),
	),
	Reconnect: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "重连"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("Esc", "退出/返回"),
	),
}

// 样式定义 - 简化版
var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	textColor      = lipgloss.Color("#CDD6F4")
	dimTextColor   = lipgloss.Color("#6C7086")
	playingColor   = lipgloss.Color("#A6E3A1")
	regionColor    = lipgloss.Color("#89B4FA")

	// 标题
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// 地区项 - 普通
	regionItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// 地区项 - 选中
	regionSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(regionColor).
				Bold(true).
				Padding(0, 1)

	// 地区项 - 当前（已确认）
	regionCurrentStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	// 电台项 - 普通
	stationItemStyle = lipgloss.NewStyle().
				Foreground(textColor)

	// 电台项 - 选中
	stationSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(primaryColor).
				Bold(true).
				Padding(0, 1)

	// 电台项 - 播放中
	stationPlayingStyle = lipgloss.NewStyle().
				Foreground(playingColor).
				Bold(true)

	// 电台项 - 选中且播放中
	stationSelectedPlayingStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#1E1E2E")).
					Background(secondaryColor).
					Bold(true).
					Padding(0, 1)

	// 状态行
	statusStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)

	// 错误
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8"))

	// 音量条
	volumeStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// 焦点指示
	focusIndicatorStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)
)

// SharedState 共享状态
type SharedState struct {
	Player        *player.FFmpegPlayer
	AuthToken     string
	Volume        float64
	Muted         bool
	PlayingIdx    int
	Stations      []model.Station
	CurrentAreaID string
}

// Model TUI 模型
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

	// 地区
	areas          []model.Area
	currentArea    int // 已确认的地区索引
	selectedArea   int // 选择中的地区索引（在地区模式下）
	isLoading      bool
	focus          FocusMode
}

// NewModel 创建模型
func NewModel(stations []model.Station, authToken string, initialVolume float64, lastStationID string, areaID string) Model {
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

	p := player.NewFFmpegPlayer(authToken, initialVolume)

	shared := &SharedState{
		Player:        p,
		AuthToken:     authToken,
		Volume:        initialVolume,
		Muted:         false,
		PlayingIdx:    -1,
		Stations:      stations,
		CurrentAreaID: areaID,
	}

	p.SetReconnectCallback(func() string {
		return hook.Auth(shared.CurrentAreaID)
	})

	return Model{
		stations:      stations,
		cursor:        defaultIdx,
		keys:          DefaultKeyMap,
		statusMessage: "自动连接中...",
		shared:        shared,
		autoPlay:      true,
		autoPlayIdx:   autoPlayIdx,
		areas:         areas,
		currentArea:   currentAreaIdx,
		selectedArea:  currentAreaIdx,
		focus:         FocusStations,
	}
}

// 消息类型
type autoPlayMsg struct{}
type stationsLoadedMsg struct {
	stations []model.Station
	err      error
}
type playResultMsg struct {
	err        error
	stationIdx int
}
type reconnectResultMsg struct {
	err error
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		return autoPlayMsg{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
			m.errorMessage = fmt.Sprintf("加载失败: %v", msg.err)
		} else {
			m.stations = msg.stations
			m.shared.Stations = msg.stations
			m.shared.CurrentAreaID = m.getCurrentAreaID()
			m.cursor = 0
			m.shared.PlayingIdx = -1
			m.statusMessage = fmt.Sprintf("已切换到 %s (%d个电台)", m.getCurrentAreaName(), len(m.stations))
			m.saveAreaConfig()
		}
		return m, nil

	case playResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("播放失败: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.shared.PlayingIdx = msg.stationIdx
			m.statusMessage = "播放中"
			m.saveConfig()
		}
		return m, nil

	case reconnectResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("重连失败: %v", msg.err)
		} else {
			m.statusMessage = "重连成功"
		}
		return m, nil

	case tea.KeyMsg:
		if m.isLoading {
			return m, nil
		}

		m.errorMessage = ""

		// 根据焦点模式处理按键
		if m.focus == FocusRegion {
			return m.handleRegionKeys(msg)
		}
		return m.handleStationKeys(msg)
	}

	return m, nil
}

// handleStationKeys 处理电台模式下的按键
func (m Model) handleStationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		} else {
			// 在顶部按上，跳到地区选择
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
		// 快速切换上一个地区
		if m.currentArea > 0 {
			m.currentArea--
			m.selectedArea = m.currentArea
			return m, m.loadStationsForCurrentArea()
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		// 快速切换下一个地区
		if m.currentArea < len(m.areas)-1 {
			m.currentArea++
			m.selectedArea = m.currentArea
			return m, m.loadStationsForCurrentArea()
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		m.statusMessage = "连接中..."
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
		if m.shared.Player != nil && m.shared.PlayingIdx >= 0 {
			m.statusMessage = "重连中..."
			return m, m.reconnect()
		}
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.saveConfig()
		if m.shared.Player != nil {
			m.shared.Player.Stop()
		}
		return m, tea.Quit

	// 数字键设置音量
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

// handleRegionKeys 处理地区模式下的按键
func (m Model) handleRegionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
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
		// 按下或Esc返回电台列表，不切换地区
		m.focus = FocusStations
		m.selectedArea = m.currentArea // 重置选择
		return m, nil

	case key.Matches(msg, m.keys.Select):
		// 确认切换地区
		if m.selectedArea != m.currentArea {
			m.currentArea = m.selectedArea
			m.focus = FocusStations
			return m, m.loadStationsForCurrentArea()
		}
		// 如果选择的是当前地区，直接返回
		m.focus = FocusStations
		return m, nil
	}

	return m, nil
}

// 辅助方法
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
	m.statusMessage = fmt.Sprintf("加载 %s ...", m.getCurrentAreaName())
	areaID := m.getCurrentAreaID()

	return func() tea.Msg {
		stations, err := api.GetStations(areaID)
		return stationsLoadedMsg{stations: stations, err: err}
	}
}

func (m *Model) saveConfig() {
	if m.shared.PlayingIdx >= 0 && m.shared.PlayingIdx < len(m.stations) {
		stationID := m.stations[m.shared.PlayingIdx].ID
		volume := m.shared.Volume
		if m.shared.Player != nil {
			volume = m.shared.Player.GetVolume()
		}
		areaID := m.getCurrentAreaID()
		go config.SaveConfig(stationID, volume, areaID)
	}
}

func (m *Model) saveAreaConfig() {
	areaID := m.getCurrentAreaID()
	volume := m.shared.Volume
	if m.shared.Player != nil {
		volume = m.shared.Player.GetVolume()
	}
	stationID := ""
	if m.shared.PlayingIdx >= 0 && m.shared.PlayingIdx < len(m.stations) {
		stationID = m.stations[m.shared.PlayingIdx].ID
	}
	go config.SaveConfig(stationID, volume, areaID)
}

func (m *Model) playStation() tea.Cmd {
	stationIdx := m.cursor
	station := m.stations[stationIdx]
	shared := m.shared

	return func() tea.Msg {
		playlistURLs, err := api.GetStreamURLs(station.ID)
		if err != nil {
			return playResultMsg{err: err, stationIdx: stationIdx}
		}

		if len(playlistURLs) == 0 {
			return playResultMsg{err: fmt.Errorf("无可用流"), stationIdx: stationIdx}
		}

		lsid := "5e586af5ccb3b0b2498abfb19eaa8472"
		lastUrl := playlistURLs[len(playlistURLs)-1]
		finalStreamUrl := fmt.Sprintf("%s?station_id=%s&l=30&lsid=%s&type=b", lastUrl, station.ID, lsid)

		shared.Player.Stop()
		time.Sleep(100 * time.Millisecond)

		err = shared.Player.Play(finalStreamUrl)
		return playResultMsg{err: err, stationIdx: stationIdx}
	}
}

func (m *Model) reconnect() tea.Cmd {
	shared := m.shared
	return func() tea.Msg {
		if shared.Player != nil {
			err := shared.Player.Reconnect()
			return reconnectResultMsg{err: err}
		}
		return reconnectResultMsg{err: fmt.Errorf("播放器未初始化")}
	}
}

// View 渲染视图
func (m Model) View() string {
	var b strings.Builder

	// 标题行：📻 Radiko + 音量
	title := titleStyle.Render("📻 Radiko")
	volBar := m.renderVolume()
	b.WriteString(fmt.Sprintf("%s  %s\n", title, volBar))

	// 地区选择行
	regionLine := m.renderRegionLine()
	b.WriteString(regionLine + "\n")

	// 分隔
	b.WriteString(strings.Repeat("─", 40) + "\n")

	// 加载中
	if m.isLoading {
		b.WriteString(fmt.Sprintf("⏳ %s\n", m.statusMessage))
		return b.String()
	}

	// 电台列表
	b.WriteString(m.renderStationList())

	// 状态行
	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render("✗ "+m.errorMessage) + "\n")
	} else if m.shared.PlayingIdx >= 0 && m.shared.PlayingIdx < len(m.stations) {
		nowPlaying := m.stations[m.shared.PlayingIdx].Name
		b.WriteString(statusStyle.Render(fmt.Sprintf("▶ %s", nowPlaying)) + "\n")
	}

	// 帮助提示
	if m.focus == FocusRegion {
		b.WriteString(statusStyle.Render("← → 选择地区  Enter 确认  ↓/Esc 返回"))
	} else {
		b.WriteString(statusStyle.Render("↑↓ 选择  Enter 播放  ← → 切地区  +- 音量  Esc 退出"))
	}

	return b.String()
}

// renderVolume 渲染音量
func (m Model) renderVolume() string {
	vol := int(m.shared.Volume * 100)
	if m.shared.Player != nil {
		vol = int(m.shared.Player.GetVolume() * 100)
	}

	if m.shared.Muted {
		return statusStyle.Render(fmt.Sprintf("🔇 %d%%", vol))
	}
	return volumeStyle.Render(fmt.Sprintf("🔊 %d%%", vol))
}

// renderRegionLine 渲染地区选择行
func (m Model) renderRegionLine() string {
	var parts []string

	// 焦点指示
	if m.focus == FocusRegion {
		parts = append(parts, focusIndicatorStyle.Render("▶ "))
	} else {
		parts = append(parts, "  ")
	}

	// 显示当前地区附近的几个地区
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
			// 在地区模式下选中的
			styled = regionSelectedStyle.Render(area.Name)
		} else if i == m.currentArea {
			// 当前确认的地区
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

	// 显示地区计数
	parts = append(parts, statusStyle.Render(fmt.Sprintf(" [%d/%d]", m.selectedArea+1, len(m.areas))))

	return strings.Join(parts, "")
}

// renderStationList 渲染电台列表
func (m Model) renderStationList() string {
	var lines []string

	maxVisible := 12
	if m.height > 0 {
		maxVisible = m.height - 8
		if maxVisible < 5 {
			maxVisible = 5
		}
	}
	if maxVisible > len(m.stations) {
		maxVisible = len(m.stations)
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
		lines = append(lines, statusStyle.Render("  ↑ 更多"))
	}

	for i := startIdx; i < endIdx; i++ {
		station := m.stations[i]
		isSelected := i == m.cursor && m.focus == FocusStations
		isPlaying := i == m.shared.PlayingIdx

		prefix := "  "
		if isPlaying {
			prefix = "▶ "
		}

		text := fmt.Sprintf("%s%s", prefix, station.Name)

		var styled string
		switch {
		case isSelected && isPlaying:
			styled = stationSelectedPlayingStyle.Render(text)
		case isSelected:
			styled = stationSelectedStyle.Render(text)
		case isPlaying:
			styled = stationPlayingStyle.Render(text)
		default:
			styled = stationItemStyle.Render(text)
		}

		lines = append(lines, styled)
	}

	if endIdx < len(m.stations) {
		lines = append(lines, statusStyle.Render("  ↓ 更多"))
	}

	return strings.Join(lines, "\n") + "\n"
}

// Run 运行 TUI
func Run(stations []model.Station, authToken string, cfg config.Config) error {
	m := NewModel(stations, authToken, cfg.Volume, cfg.LastStationID, cfg.AreaID)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()

	if m.shared.Player != nil {
		m.shared.Player.Stop()
	}

	return err
}
