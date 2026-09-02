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
	if m.layoutMode() == narrowLayout {
		body := m.theme.Accent.Render("● ") + m.theme.Focus.Render("Logo style") + "\n  " + value + "\n\n" + m.theme.Muted.Render("Default workspace") + "\n" + defaultWorkspaceName(m.cfg) + "\n\n" + m.theme.Muted.Render("Configuration") + "\n" + truncate(m.config.ConfigPath(), max(15, m.contentWidth()))
		return "Launchline Settings", body, "←→/Space Change   Esc Back   ? Help"
	}
	body := m.description("Adjust the small set of local presentation preferences.") + "\n\n" + m.theme.Accent.Render("● ") + m.theme.Focus.Render("Logo style") + "\n  " + value + "\n\n" + m.description("The block logo always collapses on narrow or short terminals.") + "\n\n" + m.theme.Muted.Render("Default workspace") + "\n" + defaultWorkspaceName(m.cfg) + "\n\n" + m.theme.Muted.Render("Configuration") + "\n" + truncate(m.config.ConfigPath(), max(15, m.contentWidth()))
	return "Launchline Settings", body, "←→/Space Change   Esc Back   ? Help   Q Quit"
}

func (m *Model) viewHelp() string {
	if m.layoutMode() == narrowLayout {
		return strings.Join([]string{
			m.description("Launchline is a local, keyboard-first workspace launcher."),
			"",
			"↑↓ Move · Enter Open/Save",
			"←→ Change · Space Toggle",
			"Esc Back · Q Quit · ? Help",
			"",
			"Apps  A Add · E Edit · D Delete",
			"Workspaces  C Create · F Default",
		}, "\n")
	}
	return strings.Join([]string{
		m.description("Launchline is a local, keyboard-first workspace launcher."),
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
		m.description("Paths and arguments are passed directly to the operating system. Launchline never evaluates them as shell code."),
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
		body := m.description(fmt.Sprintf("Delete application %q?", m.confirm.name)) + "\n\nThis removes only the Launchline application entry and its workspace references.\n" + m.theme.Warning.Render("The installed program will not be changed.")
		return "Confirm Removal — " + m.confirm.name, body, "Y Remove from Launchline   N Cancel"
	}
	body := m.description(fmt.Sprintf("Delete workspace %q?", m.confirm.name)) + "\n\nThis removes only the Launchline workspace configuration.\n" + m.theme.Warning.Render("Registered and installed applications will not be changed.")
	return "Confirm Deletion — " + m.confirm.name, body, "Y Delete Workspace   N Cancel"
}
