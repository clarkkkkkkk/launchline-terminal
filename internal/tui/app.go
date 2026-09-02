package tui

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/platform"
	"github.com/launchline/launchline/internal/tui/styles"
)

type screen int

const (
	dashboardScreen screen = iota
	applicationsScreen
	applicationFormScreen
	workspacesScreen
	workspaceFormScreen
	launchSelectScreen
	launchingScreen
	settingsScreen
	helpScreen
	confirmScreen
)

type applicationForm struct {
	id     string
	fields []textinput.Model
	focus  int
}

type workspaceForm struct {
	id       string
	name     textinput.Model
	stage    int
	cursor   int
	selected map[string]bool
}

type confirmState struct {
	kind     string
	id       string
	name     string
	returnTo screen
}

type launchState struct {
	workspace app.Workspace
	apps      []app.Application
	results   map[string]app.LaunchResult
	stream    <-chan app.LaunchResult
	done      bool
}

type Model struct {
	config     *app.Service
	launcher   *app.LaunchService
	cfg        app.Config
	screen     screen
	returnTo   screen
	cursor     int
	width      int
	height     int
	errMessage string
	notice     string
	appForm    applicationForm
	wsForm     workspaceForm
	confirm    confirmState
	launch     launchState
	spinner    spinner.Model
	cancel     context.CancelFunc
	theme      styles.Theme
}

func New(config *app.Service, launcher *app.LaunchService) (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = styles.New().Accent
	return &Model{config: config, launcher: launcher, cfg: cfg, width: 80, height: 24, theme: styles.New(), spinner: spin}, nil
}

func Run(config *app.Service, launcher *app.LaunchService) error {
	model, err := New(config, launcher)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
	}

	if m.screen == launchingScreen {
		return m.updateLaunching(message)
	}
	key, isKey := message.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if m.screen == helpScreen && (key.String() == "esc" || key.String() == "enter" || key.String() == "?") {
		m.screen, m.cursor = m.returnTo, 0
		return m, nil
	}
	if m.screen != applicationFormScreen && m.screen != workspaceFormScreen && m.screen != confirmScreen {
		switch key.String() {
		case "?":
			m.returnTo, m.screen = m.screen, helpScreen
			m.cursor, m.errMessage = 0, ""
			return m, nil
		case "q":
			return m, tea.Quit
		}
	}

	m.errMessage, m.notice = "", ""
	switch m.screen {
	case dashboardScreen:
		return m.updateDashboard(key)
	case applicationsScreen:
		return m.updateApplications(key)
	case applicationFormScreen:
		return m.updateApplicationForm(key)
	case workspacesScreen:
		return m.updateWorkspaces(key)
	case workspaceFormScreen:
		return m.updateWorkspaceForm(key)
	case launchSelectScreen:
		return m.updateLaunchSelect(key)
	case settingsScreen:
		return m.updateSettings(key)
	case confirmScreen:
		return m.updateConfirm(key)
	}
	return m, nil
}

func (m *Model) refresh() error {
	cfg, err := m.config.Load()
	if err != nil {
		return err
	}
	m.cfg = cfg
	return nil
}

func (m *Model) View() string {
	var title, body, footer string
	switch m.screen {
	case dashboardScreen:
		title, body, footer = m.viewDashboard()
	case applicationsScreen:
		title, body, footer = m.viewApplications()
	case applicationFormScreen:
		title, body, footer = m.viewApplicationForm()
	case workspacesScreen:
		title, body, footer = m.viewWorkspaces()
	case workspaceFormScreen:
		title, body, footer = m.viewWorkspaceForm()
	case launchSelectScreen:
		title, body, footer = m.viewLaunchSelect()
	case launchingScreen:
		title, body, footer = m.viewLaunching()
	case settingsScreen:
		title, body, footer = m.viewSettings()
	case helpScreen:
		title, body, footer = "Help", m.viewHelp(), "Esc Back"
	case confirmScreen:
		title, body, footer = m.viewConfirm()
	}
	return m.frame(title, body, footer)
}

func (m *Model) frame(title, body, footer string) string {
	available := m.width - 4
	if available < 24 {
		available = max(10, m.width)
	}
	if title != "" {
		title = m.theme.Title.Render(title) + "\n\n"
	}
	messages := ""
	if m.errMessage != "" {
		messages += "\n\n" + m.theme.Error.Render("× "+truncate(m.errMessage, available))
	}
	if m.notice != "" {
		messages += "\n\n" + m.theme.Success.Render("✓ "+truncate(m.notice, available))
	}
	content := title + body + messages
	if footer != "" {
		content += "\n\n" + m.theme.Footer.Render(truncate(footer, available))
	}
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(max(1, m.width)).Render(content)
}

func truncate(value string, width int) string {
	if width < 1 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func visibleRange(total, cursor, height int) (int, int) {
	rows := max(3, height-13)
	if total <= rows {
		return 0, total
	}
	start := cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > total {
		start = total - rows
	}
	return start, start + rows
}

func moveCursor(cursor, count, delta int) int {
	if count == 0 {
		return 0
	}
	cursor = (cursor + delta) % count
	if cursor < 0 {
		cursor += count
	}
	return cursor
}

func defaultWorkspaceName(cfg app.Config) string {
	for _, item := range cfg.Workspaces {
		if item.ID == cfg.DefaultWorkspaceID {
			return item.Name
		}
	}
	return "Not configured"
}

func sortedApplications(items []app.Application) []app.Application {
	out := append([]app.Application(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func sortedWorkspaces(items []app.Workspace) []app.Workspace {
	out := append([]app.Workspace(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func platformDetail() string {
	return fmt.Sprintf("%s · %s/%s", platform.Name(), runtime.GOOS, runtime.GOARCH)
}
