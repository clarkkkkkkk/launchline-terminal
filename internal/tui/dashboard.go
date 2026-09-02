package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var dashboardItems = []string{"Start Workspace", "Applications", "Workspaces", "Settings", "Help"}

const logo = ` _        _    _   _ _   _  ____ _   _ _     ___ _   _ _____
| |      / \  | | | | \ | |/ ___| | | | |   |_ _| \ | | ____|
| |     / _ \ | | | |  \| | |   | |_| | |    | ||  \| |  _|
| |___ / ___ \| |_| | |\  | |___|  _  | |___ | || |\  | |___
|_____/_/   \_\\___/|_| \_|\____|_| |_|_____|___|_| \_|_____|`

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
	var top string
	if m.width >= 76 && !m.cfg.CompactLogo {
		top = m.theme.Accent.Render(logo)
	} else {
		top = m.theme.Accent.Render("LAUNCHLINE")
	}
	var list strings.Builder
	list.WriteString(top + "\n\n")
	list.WriteString("One command. Your entire workspace.\n\n")
	list.WriteString(m.theme.Muted.Render("Main Menu") + "\n\n")
	for i, item := range dashboardItems {
		marker := "  "
		style := func(v string) string { return v }
		if i == m.cursor {
			marker = "● "
			style = func(v string) string { return m.theme.Accent.Render(v) }
		}
		list.WriteString(style(marker+item) + "\n")
	}
	list.WriteString("\n")
	list.WriteString(fmt.Sprintf("%-22s %s\n", "Current workspace", defaultWorkspaceName(m.cfg)))
	list.WriteString(fmt.Sprintf("%-22s %d\n", "Applications", len(m.cfg.Applications)))
	list.WriteString(fmt.Sprintf("%-22s %s", "Platform", platformDetail()))
	return "", list.String(), "↑↓ Navigate   Enter Select   ? Help   Q Quit"
}
