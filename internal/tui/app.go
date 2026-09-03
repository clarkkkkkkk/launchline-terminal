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
	"github.com/charmbracelet/x/ansi"
	launchassets "github.com/launchline/launchline/assets"
	"github.com/launchline/launchline/internal/app"
	launchcommand "github.com/launchline/launchline/internal/command"
	"github.com/launchline/launchline/internal/discovery"
	"github.com/launchline/launchline/internal/tui/styles"
)

type screen int

const (
	dashboardScreen screen = iota
	applicationsScreen
	applicationDetailsScreen
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
	search   textinput.Model
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
	config        *app.Service
	launcher      *app.LaunchService
	cfg           app.Config
	screen        screen
	returnTo      screen
	cursor        int
	width         int
	height        int
	version       string
	errMessage    string
	notice        string
	appForm       applicationForm
	appDetail     applicationChoice
	wsForm        workspaceForm
	confirm       confirmState
	launch        launchState
	spinner       spinner.Model
	cancel        context.CancelFunc
	refreshCancel context.CancelFunc
	theme         styles.Theme
	discovery     *discovery.Service
	catalog       discovery.Catalog
	commands      launchcommand.Registry
	history       launchcommand.History
	prompt        textinput.Model
	search        textinput.Model
	suggestions   []string
	refreshing    bool
}

func New(config *app.Service, launcher *app.LaunchService) (*Model, error) {
	return newModel(config, launcher, "dev")
}

func NewWithDiscovery(config *app.Service, launcher *app.LaunchService, discoveryService *discovery.Service, version string) (*Model, error) {
	return newModelWithDiscovery(config, launcher, discoveryService, version)
}

func newModel(config *app.Service, launcher *app.LaunchService, version string) (*Model, error) {
	return newModelWithDiscovery(config, launcher, nil, version)
}

func newModelWithDiscovery(config *app.Service, launcher *app.LaunchService, discoveryService *discovery.Service, version string) (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = styles.New().Accent
	prompt := textinput.New()
	prompt.Prompt = "> "
	prompt.CharLimit = 512
	prompt.Width = 74
	prompt.Cursor.Style = styles.New().Accent
	prompt.PromptStyle = styles.New().Accent
	prompt.TextStyle = styles.New().Command
	prompt.Focus()
	search := textinput.New()
	search.Prompt = "> "
	search.Placeholder = "Search applications"
	search.CharLimit = 200
	search.Width = 70
	search.Cursor.Style = styles.New().Accent
	search.PromptStyle = styles.New().Accent
	search.TextStyle = styles.New().Command
	model := &Model{config: config, launcher: launcher, discovery: discoveryService, cfg: cfg, catalog: discovery.EmptyCatalog(), width: 80, height: 24, version: version, theme: styles.New(), spinner: spin, commands: launchcommand.NewRegistry(), prompt: prompt, search: search}
	if discoveryService != nil {
		if catalog, loadErr := discoveryService.Load(); loadErr == nil {
			model.catalog = catalog
		} else {
			model.notice = "Cached application catalog could not be read; /refresh will rebuild it."
		}
	}
	return model, nil
}

func Run(config *app.Service, launcher *app.LaunchService, version string) error {
	model, err := newModel(config, launcher, version)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func RunWithDiscovery(config *app.Service, launcher *app.LaunchService, discoveryService *discovery.Service, version string) error {
	model, err := newModelWithDiscovery(config, launcher, discoveryService, version)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	if m.discovery != nil {
		m.refreshing = true
		return m.startDiscoveryRefresh()
	}
	return nil
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		fieldWidth := max(8, m.contentWidth()-3)
		m.prompt.Width = fieldWidth
		m.search.Width = fieldWidth
		m.wsForm.search.Width = fieldWidth
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.cancel != nil {
				m.cancel()
			}
			if m.refreshCancel != nil {
				m.refreshCancel()
			}
			return m, tea.Quit
		}
	}
	if msg, ok := message.(discoveryRefreshedMsg); ok {
		return m.applyDiscoveryRefresh(msg)
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
		if m.screen == dashboardScreen {
			return m, m.prompt.Focus()
		}
		return m, nil
	}
	if m.screen != dashboardScreen && m.screen != applicationsScreen && m.screen != applicationFormScreen && m.screen != workspaceFormScreen && m.screen != confirmScreen {
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
	case applicationDetailsScreen:
		return m.updateApplicationDetails(key)
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
	case applicationDetailsScreen:
		title, body, footer = m.viewApplicationDetails()
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
		title, body, footer = "Keyboard & Commands", m.viewHelp(), "Enter/Esc Back"
	case confirmScreen:
		title, body, footer = m.viewConfirm()
	}
	return m.frame(title, body, footer)
}

func (m *Model) frame(title, body, footer string) string {
	if m.width < 28 || m.height < 14 {
		return m.smallTerminalView()
	}
	contentWidth, leftPadding := m.contentWidth(), m.leftPadding()
	sections := make([]string, 0, 8)
	if m.screen == dashboardScreen {
		sections = append(sections, m.brandHeader(contentWidth))
	} else {
		sections = append(sections, m.brandWordmark())
	}
	sections = append(sections, m.commandContext())
	if title != "" {
		sections = append(sections, m.theme.Title.Render(title))
	}
	if body != "" {
		sections = append(sections, body)
	}

	messages := make([]string, 0, 2)
	if m.errMessage != "" {
		messages = append(messages, m.theme.Error.Render("× "+truncate(m.errMessage, contentWidth)))
	}
	if m.notice != "" {
		messages = append(messages, m.theme.Success.Render("✓ "+truncate(m.notice, contentWidth)))
	}
	sections = append(sections, messages...)
	if footer != "" {
		sections = append(sections, m.theme.Hints.Render(truncate(footer, contentWidth)))
	}
	separator := "\n\n"
	if m.layoutMode() == narrowLayout {
		separator = "\n"
	}
	upper := strings.Join(sections, separator)
	upper = ansi.Hardwrap(upper, contentWidth, true)
	status := m.statusLine(contentWidth)
	gap := 1
	if remaining := m.height - lipgloss.Height(upper) - 2; remaining > gap {
		gap = remaining
	}
	content := upper + strings.Repeat("\n", gap) + status
	return lipgloss.NewStyle().PaddingLeft(leftPadding).MaxWidth(max(1, m.width)).Render(content)
}

func (m *Model) smallTerminalView() string {
	leftPadding := 1
	width := max(4, m.width-leftPadding)
	lines := []string{
		m.theme.Logo.Render(truncate("LAUNCHLINE", width)),
		truncate(m.commandContext(), width),
		"",
		m.theme.Warning.Render(truncate("Terminal too small.", width)),
		m.theme.Muted.Render(truncate("Resize to continue.", width)),
	}
	upper := strings.Join(lines, "\n")
	status := m.statusLine(width)
	gap := max(1, m.height-lipgloss.Height(upper)-1)
	content := upper + strings.Repeat("\n", gap) + status
	return lipgloss.NewStyle().PaddingLeft(leftPadding).MaxWidth(max(1, m.width)).Render(content)
}

func truncate(value string, width int) string {
	if width < 1 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

type layoutMode int

const (
	narrowLayout layoutMode = iota
	normalLayout
	wideLayout
)

func (m *Model) layoutMode() layoutMode {
	if m.width >= 110 && m.height >= 28 {
		return wideLayout
	}
	if m.width >= 84 && m.height >= 25 {
		return normalLayout
	}
	return narrowLayout
}

func (m *Model) contentWidth() int {
	padding := m.leftPadding()
	available := max(12, m.width-padding)
	if m.layoutMode() == wideLayout {
		return min(96, available)
	}
	return min(84, available)
}

func (m *Model) leftPadding() int {
	if m.shouldRenderFullLogo() {
		return 0
	}
	if m.width < 48 {
		return 1
	}
	return 2
}

func (m *Model) brandHeader(width int) string {
	return m.brandWordmark() + "\n\n" + m.theme.Muted.Render(truncate("Tips to get started: /help", width))
}

func (m *Model) brandWordmark() string {
	wordmark := "LAUNCHLINE"
	if m.shouldRenderFullLogo() {
		wordmark = launchassets.LaunchlineLogo()
	}
	return m.theme.Logo.Render(wordmark)
}

func (m *Model) shouldRenderFullLogo() bool {
	minimumHeight := 20
	if m.screen != dashboardScreen {
		minimumHeight = 28
	}
	if m.cfg.CompactLogo || m.height < minimumHeight {
		return false
	}
	return m.width >= lipgloss.Width(launchassets.LaunchlineLogo())
}

func (m *Model) commandContext() string {
	command := "launchline"
	switch m.screen {
	case applicationsScreen, applicationDetailsScreen:
		command = "launchline apps"
	case applicationFormScreen:
		if m.appForm.id == "" {
			command = "launchline add"
		} else {
			command = "launchline apps edit"
		}
	case workspacesScreen:
		command = "launchline workspace"
	case workspaceFormScreen:
		command = "launchline workspace edit"
		if m.wsForm.id == "" {
			command = "launchline workspace create"
		}
	case launchSelectScreen:
		command = "launchline start"
	case launchingScreen:
		command = "launchline start"
		if m.launch.workspace.Name != "" {
			command += " " + m.launch.workspace.Name
		}
	case settingsScreen:
		command = "launchline config"
	case helpScreen:
		command = "launchline help"
	case confirmScreen:
		if m.confirm.kind == "workspace" {
			command = "launchline workspace delete"
		} else {
			command = "launchline apps delete"
		}
	}
	return m.theme.Accent.Render(">") + " " + m.theme.Command.Render(command)
}

func (m *Model) statusLine(width int) string {
	workspace := defaultWorkspaceName(m.cfg)
	if workspace == "Not configured" {
		workspace = "No workspace"
	}
	if m.screen == workspaceFormScreen && strings.TrimSpace(m.wsForm.name.Value()) != "" {
		workspace = m.wsForm.name.Value()
	}
	if m.screen == launchingScreen && m.launch.workspace.Name != "" {
		workspace = m.launch.workspace.Name
	}
	left := "~ " + workspace
	if m.refreshing {
		left = "~ Applications · Refreshing..."
	}
	right := "Launchline " + m.version
	if m.layoutMode() != narrowLayout {
		right += " · " + runtime.GOOS + "/" + runtime.GOARCH
	}
	if lipgloss.Width(left)+lipgloss.Width(right)+3 > width {
		if width >= 34 {
			left = truncate(left, width-lipgloss.Width(right)-2)
		} else {
			return m.theme.Status.Render(truncate(left, width))
		}
	}
	space := max(2, width-lipgloss.Width(left)-lipgloss.Width(right))
	return m.theme.Status.Render(left + strings.Repeat(" ", space) + right)
}

func (m *Model) description(value string) string {
	return m.theme.Muted.Render(ansi.Wordwrap(value, m.contentWidth(), ""))
}

func (m *Model) menuItem(index int, label string, numbered bool) string {
	prefix := "  "
	if index == m.cursor {
		prefix = m.theme.Accent.Render("● ")
	}
	if numbered {
		label = fmt.Sprintf("%d. %s", index+1, label)
	}
	if index == m.cursor {
		label = m.theme.Focus.Render(label)
	}
	return prefix + label
}

func visibleRange(total, cursor, height int) (int, int) {
	return visibleRangeRows(total, cursor, max(3, height-13))
}

func visibleRangeRows(total, cursor, rows int) (int, int) {
	rows = max(1, rows)
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
