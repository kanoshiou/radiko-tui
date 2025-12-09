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

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap 定义快捷键
type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Play      key.Binding
	VolUp     key.Binding
	VolDown   key.Binding
	Mute      key.Binding
	Reconnect key.Binding
	Quit      key.Binding
}

// ShortHelp 返回简短的帮助信息
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Play, k.VolUp, k.VolDown, k.Mute, k.Quit}
}

// FullHelp 返回详细帮助信息
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Play},
		{k.VolUp, k.VolDown, k.Mute},
		{k.Reconnect, k.Quit},
	}
}

// 默认快捷键
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "上移"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "下移"),
	),
	Play: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("Enter", "播放"),
	),
	VolUp: key.NewBinding(
		key.WithKeys("+", "=", "e"),
		key.WithHelp("+/e", "音量+"),
	),
	VolDown: key.NewBinding(
		key.WithKeys("-", "_", "q"),
		key.WithHelp("-/q", "音量-"),
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
		key.WithHelp("Esc", "退出"),
	),
}

// 样式定义
var (
	// 主题颜色
	primaryColor   = lipgloss.Color("#7C3AED") // 紫色
	secondaryColor = lipgloss.Color("#10B981") // 翠绿色
	accentColor    = lipgloss.Color("#F59E0B") // 琥珀色
	textColor      = lipgloss.Color("#CDD6F4") // 浅色文字
	dimTextColor   = lipgloss.Color("#6C7086") // 暗淡文字
	playingColor   = lipgloss.Color("#A6E3A1") // 播放中颜色

	// 标题样式
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Bold(true).
			Padding(0, 2).
			MarginBottom(1)

	// 副标题样式
	subtitleStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			Italic(true).
			MarginBottom(1)

	// 电台列表容器样式
	listContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	// 电台项目样式 - 普通
	stationItemStyle = lipgloss.NewStyle().
				Foreground(textColor).
				PaddingLeft(2)

	// 电台项目样式 - 选中
	selectedStationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(primaryColor).
				Bold(true).
				PaddingLeft(2).
				PaddingRight(2)

	// 电台项目样式 - 正在播放
	playingStationStyle = lipgloss.NewStyle().
				Foreground(playingColor).
				Bold(true).
				PaddingLeft(2)

	// 电台项目样式 - 选中且正在播放
	selectedPlayingStationStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#1E1E2E")).
					Background(secondaryColor).
					Bold(true).
					PaddingLeft(2).
					PaddingRight(2)

	// 状态栏样式
	statusBarStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#313244")).
			Padding(0, 2).
			MarginTop(1)

	// 音量条样式
	volumeBarStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// 帮助样式
	helpStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			MarginTop(1)

	// 错误样式
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Bold(true)

	// 播放指示器样式
	playingIndicatorStyle = lipgloss.NewStyle().
				Foreground(playingColor).
				Bold(true)
)

// SharedState 共享状态（使用指针在 Bubble Tea 的值传递中保持状态）
type SharedState struct {
	Player     *player.FFmpegPlayer
	AuthToken  string
	Volume     float64
	Muted      bool
	PlayingIdx int
	Stations   []model.Station // 保存电台列表的引用
}

// Model 是 TUI 的主模型
type Model struct {
	stations      []model.Station // 电台列表
	cursor        int             // 光标位置
	width         int
	height        int
	keys          KeyMap
	help          help.Model
	statusMessage string
	errorMessage  string
	shared        *SharedState // 共享状态指针
	autoPlay      bool         // 是否需要自动播放
	autoPlayIdx   int          // 自动播放的电台索引
}

// NewModel 创建新的 TUI 模型
func NewModel(stations []model.Station, authToken string, initialVolume float64, lastStationID string) Model {
	h := help.New()
	h.ShowAll = false

	// 找到上次播放的电台索引，如果找不到则使用默认电台
	defaultIdx := 0
	autoPlayIdx := -1
	for i, s := range stations {
		if s.ID == lastStationID {
			defaultIdx = i
			autoPlayIdx = i
			break
		}
	}

	// 如果没有找到上次的电台，尝试 QRR 作为默认
	if autoPlayIdx == -1 {
		for i, s := range stations {
			if s.ID == "QRR" {
				defaultIdx = i
				autoPlayIdx = i
				break
			}
		}
	}

	// 如果还是没找到，使用第一个电台
	if autoPlayIdx == -1 && len(stations) > 0 {
		autoPlayIdx = 0
	}

	// 预先创建播放器
	p := player.NewFFmpegPlayer(authToken, initialVolume)
	p.SetReconnectCallback(func() string {
		return hook.Auth()
	})

	shared := &SharedState{
		Player:     p,
		AuthToken:  authToken,
		Volume:     initialVolume,
		Muted:      false,
		PlayingIdx: -1,
		Stations:   stations,
	}

	return Model{
		stations:      stations,
		cursor:        defaultIdx,
		keys:          DefaultKeyMap,
		help:          h,
		statusMessage: "⏳ 正在自动连接...",
		shared:        shared,
		autoPlay:      true,
		autoPlayIdx:   autoPlayIdx,
	}
}

// autoPlayMsg 自动播放消息
type autoPlayMsg struct{}

// Init 初始化 - 触发自动播放
func (m Model) Init() tea.Cmd {
	// 返回一个命令来触发自动播放
	return func() tea.Msg {
		return autoPlayMsg{}
	}
}

// Update 处理消息
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case autoPlayMsg:
		// 处理自动播放
		if m.autoPlay && m.autoPlayIdx >= 0 && m.autoPlayIdx < len(m.stations) {
			m.autoPlay = false
			m.cursor = m.autoPlayIdx
			return m, m.playStation()
		}
		return m, nil

	case tea.KeyMsg:
		// 清除错误信息
		m.errorMessage = ""

		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.stations)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.Play):
			m.statusMessage = "⏳ 正在连接..."
			return m, m.playStation()

		case key.Matches(msg, m.keys.VolUp):
			if m.shared.Player != nil {
				m.shared.Player.IncreaseVolume(0.05)
				m.shared.Volume = m.shared.Player.GetVolume()
				m.shared.Muted = false
				// 保存音量
				m.saveConfig()
			}
			return m, nil

		case key.Matches(msg, m.keys.VolDown):
			if m.shared.Player != nil {
				m.shared.Player.DecreaseVolume(0.05)
				m.shared.Volume = m.shared.Player.GetVolume()
				m.shared.Muted = false
				// 保存音量
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
				m.statusMessage = "🔄 正在重连..."
				return m, m.reconnect()
			}
			return m, nil

		case key.Matches(msg, m.keys.Quit):
			// 退出前保存配置
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
				// 保存音量
				m.saveConfig()
			}
			return m, nil
		}

	case playResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("❌ 播放失败: %v", msg.err)
			m.statusMessage = ""
		} else {
			m.shared.PlayingIdx = msg.stationIdx
			m.statusMessage = "🎵 正在播放..."
			// 保存当前播放的电台
			m.saveConfig()
		}
		return m, nil

	case reconnectResultMsg:
		if msg.err != nil {
			m.errorMessage = fmt.Sprintf("❌ 重连失败: %v", msg.err)
		} else {
			m.statusMessage = "✓ 重连成功"
		}
		return m, nil
	}

	return m, nil
}

// saveConfig 保存配置
func (m *Model) saveConfig() {
	if m.shared.PlayingIdx >= 0 && m.shared.PlayingIdx < len(m.stations) {
		stationID := m.stations[m.shared.PlayingIdx].ID
		volume := m.shared.Volume
		if m.shared.Player != nil {
			volume = m.shared.Player.GetVolume()
		}
		// 异步保存，不阻塞 UI
		go config.SaveLastStation(stationID, volume)
	}
}

// playResultMsg 播放结果消息
type playResultMsg struct {
	err        error
	stationIdx int
}

// reconnectResultMsg 重连结果消息
type reconnectResultMsg struct {
	err error
}

// playStation 播放电台
func (m *Model) playStation() tea.Cmd {
	stationIdx := m.cursor
	station := m.stations[stationIdx]
	shared := m.shared

	return func() tea.Msg {
		// 获取播放列表 URL
		playlistURLs, err := api.GetStreamURLs(station.ID)
		if err != nil {
			return playResultMsg{err: err, stationIdx: stationIdx}
		}

		if len(playlistURLs) == 0 {
			return playResultMsg{err: fmt.Errorf("no stream URLs available"), stationIdx: stationIdx}
		}

		// 使用最后一个 URL
		lsid := "5e586af5ccb3b0b2498abfb19eaa8472"
		lastUrl := playlistURLs[len(playlistURLs)-1]
		finalStreamUrl := fmt.Sprintf("%s?station_id=%s&l=30&lsid=%s&type=b", lastUrl, station.ID, lsid)

		// 停止当前播放
		shared.Player.Stop()

		// 等待一小段时间确保资源释放
		time.Sleep(100 * time.Millisecond)

		// 播放新电台
		err = shared.Player.Play(finalStreamUrl)
		return playResultMsg{err: err, stationIdx: stationIdx}
	}
}

// reconnect 重连
func (m *Model) reconnect() tea.Cmd {
	shared := m.shared
	return func() tea.Msg {
		if shared.Player != nil {
			err := shared.Player.Reconnect()
			return reconnectResultMsg{err: err}
		}
		return reconnectResultMsg{err: fmt.Errorf("player not initialized")}
	}
}

// View 渲染视图
func (m Model) View() string {
	var b strings.Builder

	// 标题
	title := titleStyle.Render("📻 Radiko JP Player")
	b.WriteString(title + "\n")

	// 副标题
	subtitle := subtitleStyle.Render("日本广播电台播放器")
	b.WriteString(subtitle + "\n\n")

	// 电台列表
	var stationItems []string

	// 计算可见的电台数量（根据窗口高度）
	maxVisible := 15
	if m.height > 0 {
		maxVisible = m.height - 12 // 留出空间给其他元素
		if maxVisible < 5 {
			maxVisible = 5
		}
		if maxVisible > len(m.stations) {
			maxVisible = len(m.stations)
		}
	}
	if maxVisible > len(m.stations) {
		maxVisible = len(m.stations)
	}

	// 计算滚动偏移
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

	for i := startIdx; i < endIdx; i++ {
		station := m.stations[i]
		isSelected := i == m.cursor
		isPlaying := i == m.shared.PlayingIdx

		var itemText string
		var styledItem string

		// 构建电台文本
		if isPlaying {
			itemText = fmt.Sprintf("▶ %s (%s)", station.Name, station.ID)
		} else {
			itemText = fmt.Sprintf("  %s (%s)", station.Name, station.ID)
		}

		// 应用样式
		switch {
		case isSelected && isPlaying:
			styledItem = selectedPlayingStationStyle.Render(itemText)
		case isSelected:
			styledItem = selectedStationStyle.Render(itemText)
		case isPlaying:
			styledItem = playingStationStyle.Render(itemText)
		default:
			styledItem = stationItemStyle.Render(itemText)
		}

		stationItems = append(stationItems, styledItem)
	}

	// 列表标题
	listTitle := fmt.Sprintf("电台列表 (%d/%d)", m.cursor+1, len(m.stations))
	listContent := listTitle + "\n" + strings.Join(stationItems, "\n")

	// 添加滚动指示器
	if startIdx > 0 {
		listContent = "↑ 更多电台...\n" + listContent
	}
	if endIdx < len(m.stations) {
		listContent = listContent + "\n↓ 更多电台..."
	}

	b.WriteString(listContainerStyle.Render(listContent))
	b.WriteString("\n")

	// 状态栏
	var statusItems []string

	// 当前播放信息
	if m.shared.PlayingIdx >= 0 && m.shared.PlayingIdx < len(m.stations) {
		nowPlaying := fmt.Sprintf("🎵 %s", m.stations[m.shared.PlayingIdx].Name)
		statusItems = append(statusItems, playingIndicatorStyle.Render(nowPlaying))
	}

	// 音量条
	volumeBar := m.renderVolumeBar()
	statusItems = append(statusItems, volumeBar)

	if len(statusItems) > 0 {
		statusContent := strings.Join(statusItems, "  │  ")
		b.WriteString(statusBarStyle.Render(statusContent))
		b.WriteString("\n")
	}

	// 状态消息或错误消息
	if m.errorMessage != "" {
		b.WriteString(errorStyle.Render(m.errorMessage) + "\n")
	} else if m.statusMessage != "" {
		b.WriteString(subtitleStyle.Render(m.statusMessage) + "\n")
	}

	// 帮助
	helpView := m.help.View(m.keys)
	b.WriteString(helpStyle.Render(helpView))

	return b.String()
}

// renderVolumeBar 渲染音量条
func (m Model) renderVolumeBar() string {
	vol := int(m.shared.Volume * 100)
	if m.shared.Player != nil {
		vol = int(m.shared.Player.GetVolume() * 100)
	}

	barLength := 10
	filled := int(float64(barLength) * m.shared.Volume)
	if m.shared.Player != nil {
		filled = int(float64(barLength) * m.shared.Player.GetVolume())
	}

	bar := ""
	for i := 0; i < barLength; i++ {
		if i < filled && !m.shared.Muted {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	volText := fmt.Sprintf("%3d%%", vol)
	if m.shared.Muted {
		return fmt.Sprintf("🔇 Vol: %s [%s]", volText, bar)
	}
	return fmt.Sprintf("🔊 Vol: %s [%s]", volText, volumeBarStyle.Render(bar))
}

// Run 运行 TUI
func Run(stations []model.Station, authToken string, cfg config.Config) error {
	m := NewModel(stations, authToken, cfg.Volume, cfg.LastStationID)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()

	// 确保退出时停止播放器
	if m.shared.Player != nil {
		m.shared.Player.Stop()
	}

	return err
}
