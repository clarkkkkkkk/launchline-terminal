package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var dashboardItems = []string{"Start Workspace", "Applications", "Workspaces", "Settings", "Help"}

func (m *Model) updateDashboard(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, len(dashboardItems), -1)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, len(dashboardItems), 1)
	case "enter":
		m.errMessage = ""
		switch m.cursor {
		case 0:
			m.screen = launchSelectScreen
		case 1:
			m.screen = applicationsScreen
		case 2:
			m.screen = workspacesScreen
		case 3:
			m.screen = settingsScreen
		case 4:
			m.returnTo, m.screen = dashboardScreen, helpScreen
		}
		m.cursor = 0
	}
	return m, nil
}

func (m *Model) viewDashboard() (string, string, string) {
	var list strings.Builder
	list.WriteString(m.description("Select what you want to do:") + "\n\n")
	for i, item := range dashboardItems {
		list.WriteString(m.menuItem(i, item, true) + "\n")
	}
	return "Workspace", strings.TrimRight(list.String(), "\n"), "↑↓ Move   Enter Select   ? Help   Q Quit"
}
