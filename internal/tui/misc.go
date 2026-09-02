package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateSettings(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "left", "right", " ", "enter":
		value := !m.cfg.CompactLogo
		if err := m.config.SetCompactLogo(value); err != nil {
			m.errMessage = err.Error()
		} else if err := m.refresh(); err != nil {
			m.errMessage = err.Error()
		} else {
			m.notice = "Display preference saved."
		}
	case "esc":
		m.screen, m.cursor = dashboardScreen, 0
	}
	return m, nil
}

func (m *Model) viewSettings() (string, string, string) {
	value := "Automatic full logo"
	if m.cfg.CompactLogo {
		value = "Always compact"
	}
	body := m.theme.Accent.Render("● Logo style") + "\n" + value + "\n\n" + m.theme.Muted.Render("The full logo is always replaced by compact branding on narrow terminals.") + "\n\n" + m.theme.Muted.Render("Default workspace") + "\n" + defaultWorkspaceName(m.cfg) + "\n\n" + m.theme.Muted.Render("Configuration") + "\n" + truncate(m.config.ConfigPath(), max(15, m.width-6))
	return "Settings", body, "←→/Space Toggle Logo   Esc Back   ? Help   Q Quit"
}

func (m *Model) viewHelp() string {
	return strings.Join([]string{
		"Launchline is a local, keyboard-first workspace launcher.",
		"",
		"↑ / ↓       Navigate lists",
		"← / →       Change options or return to a form field",
		"Enter       Open, confirm, or save",
		"Space       Toggle application selection",
		"Esc         Cancel or go back",
		"Q           Quit outside text inputs",
		"?           Open this contextual help",
		"",
		"Applications",
		"A add · E edit · D delete",
		"",
		"Workspaces",
		"C create · E edit · F make default · D delete",
		"",
		m.theme.Muted.Render("Paths and arguments are passed directly to the operating system. Launchline never evaluates them as shell code."),
	}, "\n")
}

func (m *Model) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y":
		var err error
		if m.confirm.kind == "application" {
			err = m.config.DeleteApplication(m.confirm.id)
		} else {
			err = m.config.DeleteWorkspace(m.confirm.id)
		}
		if err == nil {
			err = m.refresh()
		}
		m.screen, m.cursor = m.confirm.returnTo, 0
		if err != nil {
			m.errMessage = err.Error()
		} else if m.confirm.kind == "application" {
			m.notice = "Application removed from Launchline. The installed program was not changed."
		} else {
			m.notice = "Workspace removed. Registered applications were not changed."
		}
	case "n", "esc":
		m.screen = m.confirm.returnTo
	}
	return m, nil
}

func (m *Model) viewConfirm() (string, string, string) {
	if m.confirm.kind == "application" {
		body := fmt.Sprintf("Delete application %q?\n\nThis removes only the Launchline application entry and its workspace references.\nIt does not uninstall, delete, or modify the actual program.", m.confirm.name)
		return "Confirm removal", body, "Y Remove from Launchline   N Cancel"
	}
	body := fmt.Sprintf("Delete workspace %q?\n\nThis removes only the Launchline workspace configuration.\nIt does not uninstall or delete applications.", m.confirm.name)
	return "Confirm deletion", body, "Y Delete Workspace   N Cancel"
}
