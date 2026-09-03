package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
)

func (m *Model) updateWorkspaces(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := sortedWorkspaces(m.cfg.Workspaces)
	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, len(items), -1)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, len(items), 1)
	case "c", "a":
		m.openWorkspaceForm(nil)
	case "e", "enter":
		if len(items) > 0 {
			m.openWorkspaceForm(&items[m.cursor])
		}
	case "d":
		if len(items) > 0 {
			item := items[m.cursor]
			m.confirm = confirmState{kind: "workspace", id: item.ID, name: item.Name, returnTo: workspacesScreen}
			m.screen = confirmScreen
		}
	case "f":
		if len(items) > 0 {
			if err := m.config.SetDefaultWorkspace(items[m.cursor].ID); err != nil {
				m.errMessage = err.Error()
			} else if err := m.refresh(); err != nil {
				m.errMessage = err.Error()
			} else {
				m.notice = items[m.cursor].Name + " is now the default workspace."
			}
		}
	case "esc":
		m.screen, m.cursor = dashboardScreen, 0
	}
	return m, nil
}

func (m *Model) viewWorkspaces() (string, string, string) {
	items := sortedWorkspaces(m.cfg.Workspaces)
	if len(items) == 0 {
		body := m.description("Create and manage the groups of applications you launch together.") + "\n\n" + m.theme.Title.Render("No workspace configured.") + "\n" + m.description("Create your first workspace and group the applications you normally start together.") + "\n\n" + m.theme.Accent.Render(">") + " Create Workspace"
		return "Workspace Management", body, "C Create   Esc Back   ? Help   Q Quit"
	}
	start, end := visibleRange(len(items), m.cursor, m.height)
	var body strings.Builder
	body.WriteString(m.description("Create and manage the groups of applications you launch together.") + "\n\n")
	for i := start; i < end; i++ {
		defaultMark := ""
		if items[i].ID == m.cfg.DefaultWorkspaceID {
			defaultMark = "  " + m.theme.Success.Render("✓ Default")
		}
		line := m.menuItem(i, items[i].Name, true) + "  " + m.theme.Muted.Render(fmt.Sprintf("%d apps", len(items[i].Applications))) + defaultMark
		body.WriteString(line + "\n")
	}
	return "Workspace Management", body.String(), "↑↓ Move   C Create   E/Enter Edit   F Default   D Delete   Esc Back"
}

func (m *Model) openWorkspaceForm(item *app.Workspace) {
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "Workspace name"
	name.CharLimit = 200
	name.Width = max(12, min(70, m.contentWidth()-2))
	name.Cursor.Style = m.theme.Accent
	name.TextStyle = m.theme.Command
	name.PlaceholderStyle = m.theme.Muted
	selected := map[string]bool{}
	id := ""
	stage := 0
	if item != nil {
		id = item.ID
		stage = 1
		name.SetValue(item.Name)
		for _, appID := range item.Applications {
			selected[appID] = true
		}
	}
	search := textinput.New()
	search.Prompt = "> "
	search.Placeholder = "Search applications"
	search.CharLimit = 200
	search.Width = max(12, min(70, m.contentWidth()-2))
	search.Cursor.Style = m.theme.Accent
	search.PromptStyle = m.theme.Accent
	search.TextStyle = m.theme.Command
	if stage == 0 {
		name.Focus()
	} else {
		search.Focus()
	}
	m.wsForm = workspaceForm{id: id, name: name, stage: stage, selected: selected, search: search}
	m.screen = workspaceFormScreen
}

func (m *Model) updateWorkspaceForm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	form := &m.wsForm
	if key.String() == "esc" {
		m.screen, m.cursor = workspacesScreen, 0
		return m, nil
	}
	if form.stage == 0 {
		if key.String() == "enter" || key.String() == "tab" || key.String() == "down" {
			if strings.TrimSpace(form.name.Value()) == "" {
				m.errMessage = "Workspace name is required."
				return m, nil
			}
			form.name.Blur()
			form.stage = 1
			return m, form.search.Focus()
		}
		var command tea.Cmd
		form.name, command = form.name.Update(key)
		return m, command
	}
	items := m.applicationChoices(form.search.Value())
	switch key.String() {
	case "?":
		m.returnTo, m.screen = workspaceFormScreen, helpScreen
	case "up":
		form.cursor = moveCursor(form.cursor, len(items), -1)
	case "down":
		form.cursor = moveCursor(form.cursor, len(items), 1)
	case "shift+tab":
		form.search.Blur()
		form.stage = 0
		return m, form.name.Focus()
	case " ", "space":
		if len(items) > 0 {
			id := items[form.cursor].key
			form.selected[id] = !form.selected[id]
		}
	case "right":
		if len(items) > 0 {
			form.selected[items[form.cursor].key] = true
		}
	case "left":
		if len(items) > 0 {
			form.selected[items[form.cursor].key] = false
		}
	case "enter":
		selected := make([]string, 0, len(form.selected))
		for _, item := range m.applicationChoices("") {
			if !form.selected[item.key] {
				continue
			}
			if item.configured != nil {
				selected = append(selected, item.configured.ID)
			} else if item.discovered != nil {
				linked, err := m.config.LinkDiscoveredApplication(app.Application{Name: item.discovered.Name, Path: item.discovered.Target, Arguments: item.discovered.Arguments, Kind: item.discovered.Kind, DiscoveryID: item.discovered.ID, Source: item.discovered.Source})
				if err != nil {
					m.errMessage = err.Error()
					return m, nil
				}
				selected = append(selected, linked.ID)
			}
		}
		input := app.Workspace{Name: form.name.Value(), Applications: selected}
		var err error
		if form.id == "" {
			_, err = m.config.CreateWorkspace(input)
		} else {
			_, err = m.config.UpdateWorkspace(form.id, input)
		}
		if err != nil {
			m.errMessage = err.Error()
			return m, nil
		}
		if err := m.refresh(); err != nil {
			m.errMessage = err.Error()
			return m, nil
		}
		m.screen, m.cursor = workspacesScreen, 0
		m.notice = "Workspace saved."
		return m, nil
	default:
		var command tea.Cmd
		form.search, command = form.search.Update(key)
		filtered := m.applicationChoices(form.search.Value())
		if form.cursor >= len(filtered) {
			form.cursor = max(0, len(filtered)-1)
		}
		return m, command
	}
	return m, nil
}

func (m *Model) viewWorkspaceForm() (string, string, string) {
	title := "Workspace Editor — New Workspace"
	if strings.TrimSpace(m.wsForm.name.Value()) != "" {
		title = "Workspace Editor — " + m.wsForm.name.Value()
	}
	if m.wsForm.stage == 0 {
		body := m.description("Name this workspace, then choose the applications it should launch.") + "\n\n" + m.theme.Accent.Render("● ") + m.theme.Focus.Render("Workspace name") + "\n" + m.wsForm.name.View()
		return title, body, "Enter/Tab Choose Applications   Esc Cancel"
	}
	items := m.applicationChoices(m.wsForm.search.Value())
	var body strings.Builder
	body.WriteString(m.description("Select discovered or manual applications for this workspace.") + "\n\n")
	body.WriteString(m.theme.Muted.Render(fmt.Sprintf("%d selected", selectedApplicationCount(m.wsForm.selected))) + "\n")
	body.WriteString(m.theme.Muted.Render("Search applications") + "\n" + m.wsForm.search.View() + "\n\n")
	if len(items) == 0 {
		body.WriteString(m.theme.Title.Render("No applications configured.") + "\n" + m.description("You can save an empty workspace and add applications later.") + "\n")
	} else {
		start, end := visibleRangeRows(len(items), m.wsForm.cursor, m.height-18)
		for i := start; i < end; i++ {
			check := "[ ]"
			if m.wsForm.selected[items[i].key] {
				check = "[✓]"
			}
			if items[i].unavailable {
				check = "[!]"
			}
			prefix := "  "
			label := check + " " + items[i].name
			if i == m.wsForm.cursor {
				prefix = m.theme.Accent.Render("● ")
				label = m.theme.Focus.Render(label)
			}
			body.WriteString(prefix + label + "\n")
		}
	}
	return title, body.String(), "↑↓ Move   ← Unselect   → Select   Space Toggle   Enter Save   Shift+Tab Name"
}

func selectedApplicationCount(selected map[string]bool) int {
	count := 0
	for _, isSelected := range selected {
		if isSelected {
			count++
		}
	}
	return count
}
