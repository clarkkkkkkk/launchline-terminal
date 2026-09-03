package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
	launchcommand "github.com/launchline/launchline/internal/command"
	"github.com/launchline/launchline/internal/discovery"
)

type discoveryRefreshedMsg struct {
	result discovery.RefreshResult
	err    error
}

func (m *Model) refreshDiscovery(ctx context.Context) tea.Cmd {
	if m.discovery == nil {
		return nil
	}
	return func() tea.Msg {
		result, err := m.discovery.Refresh(ctx)
		return discoveryRefreshedMsg{result: result, err: err}
	}
}

func (m *Model) startDiscoveryRefresh() tea.Cmd {
	if m.refreshCancel != nil {
		m.refreshCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.refreshCancel = cancel
	return m.refreshDiscovery(ctx)
}

func (m *Model) applyDiscoveryRefresh(msg discoveryRefreshedMsg) (tea.Model, tea.Cmd) {
	m.refreshing = false
	m.refreshCancel = nil
	if len(msg.result.Catalog.Applications) > 0 || msg.err == nil {
		m.catalog = msg.result.Catalog
	}
	available := map[string]app.Application{}
	for _, item := range m.catalog.Applications {
		available[item.ID] = app.Application{Name: item.Name, Path: item.Target, Arguments: item.Arguments, Kind: item.Kind, DiscoveryID: item.ID, Source: item.Source}
	}
	if err := m.config.ReconcileDiscoveredCatalog(available); err != nil {
		m.errMessage = err.Error()
	} else if err := m.refresh(); err != nil {
		m.errMessage = err.Error()
	}
	if msg.err != nil {
		m.notice = "Application refresh completed with warnings. Cached applications were retained."
		return m, nil
	}
	switch {
	case msg.result.Added > 0:
		m.notice = fmt.Sprintf("Application catalog refreshed. %d new applications found.", msg.result.Added)
	case msg.result.Removed > 0:
		m.notice = fmt.Sprintf("Application catalog refreshed. %d applications are no longer detected.", msg.result.Removed)
	default:
		m.notice = "Application catalog is up to date."
	}
	return m, nil
}

func (m *Model) updateDashboard(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		return m.executePrompt()
	case "esc":
		m.prompt.SetValue("")
		m.suggestions = nil
		m.errMessage, m.notice = "", ""
		return m, nil
	case "up":
		m.prompt.SetValue(m.history.Previous(m.prompt.Value()))
		m.prompt.CursorEnd()
		return m, nil
	case "down":
		m.prompt.SetValue(m.history.Next())
		m.prompt.CursorEnd()
		return m, nil
	case "tab":
		completed, suggestions := m.commands.Complete(m.prompt.Value(), workspaceNames(m.cfg))
		m.prompt.SetValue(completed)
		m.prompt.CursorEnd()
		m.suggestions = suggestions
		return m, nil
	}
	var command tea.Cmd
	m.prompt, command = m.prompt.Update(key)
	_, m.suggestions = m.commands.Complete(m.prompt.Value(), workspaceNames(m.cfg))
	return m, command
}

func (m *Model) executePrompt() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.prompt.Value())
	if value == "" {
		return m, nil
	}
	m.history.Add(value)
	m.prompt.SetValue("")
	m.suggestions = nil
	m.errMessage, m.notice = "", ""
	invocation, err := m.commands.Parse(value)
	if err != nil {
		m.errMessage = err.Error()
		return m, nil
	}
	switch invocation.Definition.Action {
	case launchcommand.ActionHelp:
		m.returnTo, m.screen = dashboardScreen, helpScreen
	case launchcommand.ActionExit:
		if m.refreshCancel != nil {
			m.refreshCancel()
		}
		return m, tea.Quit
	case launchcommand.ActionApplications:
		m.screen, m.cursor = applicationsScreen, 0
		m.search.SetValue("")
		return m, m.search.Focus()
	case launchcommand.ActionWorkspaces:
		m.screen, m.cursor = workspacesScreen, 0
	case launchcommand.ActionWorkspace:
		if len(invocation.Arguments) != 1 {
			m.errMessage = "Usage: /workspace <name>"
			return m, nil
		}
		workspace, ok := resolveWorkspace(m.cfg, invocation.Arguments[0])
		if !ok {
			m.errMessage = fmt.Sprintf("Workspace %q was not found.", invocation.Arguments[0])
			return m, nil
		}
		m.openWorkspaceForm(&workspace)
	case launchcommand.ActionStart:
		if len(invocation.Arguments) > 1 {
			m.errMessage = "Usage: /start [workspace]"
			return m, nil
		}
		reference := ""
		if len(invocation.Arguments) == 1 {
			reference = invocation.Arguments[0]
		}
		return m, m.beginLaunch(reference)
	case launchcommand.ActionAdd:
		m.openApplicationForm(nil)
	case launchcommand.ActionRefresh:
		if m.discovery == nil {
			m.errMessage = "Application discovery is unavailable in this session."
			return m, nil
		}
		m.refreshing = true
		m.notice = "Refreshing applications..."
		return m, m.startDiscoveryRefresh()
	case launchcommand.ActionSettings:
		m.screen, m.cursor = settingsScreen, 0
	case launchcommand.ActionVersion:
		m.notice = "Launchline " + m.version
	case launchcommand.ActionClear:
		m.errMessage, m.notice = "", ""
	}
	return m, nil
}

func (m *Model) viewDashboard() (string, string, string) {
	var body strings.Builder
	workspace, ok := resolveWorkspace(m.cfg, m.cfg.DefaultWorkspaceID)
	if !ok {
		body.WriteString(m.description("No default workspace configured. Use /workspaces to create or choose one."))
	} else {
		body.WriteString(m.theme.Title.Render(workspace.Name) + "\n")
		body.WriteString(m.theme.Muted.Render(fmt.Sprintf("%d applications · Default workspace", len(workspace.Applications))) + "\n\n")
		byID := map[string]app.Application{}
		for _, item := range m.cfg.Applications {
			byID[item.ID] = item
		}
		limit := len(workspace.Applications)
		if m.shouldRenderFullLogo() {
			limit = min(limit, max(0, m.height-23))
		} else if m.layoutMode() == narrowLayout {
			limit = min(limit, 4)
		} else {
			limit = min(limit, 6)
		}
		for _, id := range workspace.Applications[:limit] {
			item, exists := byID[id]
			if !exists {
				continue
			}
			if item.Unavailable {
				body.WriteString(m.theme.Warning.Render("! "+item.Name+" — not currently available") + "\n")
			} else {
				body.WriteString(m.theme.Success.Render("✓ "+item.Name) + "\n")
			}
		}
		if remaining := len(workspace.Applications) - limit; remaining > 0 {
			body.WriteString(m.theme.Muted.Render(fmt.Sprintf("… and %d more", remaining)) + "\n")
		}
	}
	rule := m.theme.Divider.Render(strings.Repeat("─", max(1, m.contentWidth())))
	body.WriteString("\n" + rule + "\n" + m.prompt.View() + "\n" + rule)
	if len(m.suggestions) > 0 {
		suggestions := m.suggestions
		if len(suggestions) > 3 {
			suggestions = suggestions[:3]
		}
		body.WriteString("\n" + m.theme.Muted.Render("Suggestion: "+strings.Join(suggestions, "  ")+" · Tab to complete"))
	}
	footer := "/help for commands   ↑↓ History   Tab Complete   Esc Clear"
	if m.refreshing {
		footer = "Refreshing applications...   " + footer
	}
	return "", strings.TrimRight(body.String(), "\n"), footer
}

func workspaceNames(cfg app.Config) []string {
	names := make([]string, 0, len(cfg.Workspaces))
	for _, item := range cfg.Workspaces {
		names = append(names, item.Name)
	}
	return names
}

func resolveWorkspace(cfg app.Config, reference string) (app.Workspace, bool) {
	for _, item := range cfg.Workspaces {
		if item.ID == reference || strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(reference)) {
			return item, true
		}
	}
	return app.Workspace{}, false
}
