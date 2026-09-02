package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
)

func (m *Model) updateApplications(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := sortedApplications(m.cfg.Applications)
	switch key.String() {
	case "up", "k":
		m.cursor = moveCursor(m.cursor, len(items), -1)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, len(items), 1)
	case "a":
		m.openApplicationForm(nil)
	case "e", "enter":
		if len(items) > 0 {
			m.openApplicationForm(&items[m.cursor])
		}
	case "d":
		if len(items) > 0 {
			item := items[m.cursor]
			m.confirm = confirmState{kind: "application", id: item.ID, name: item.Name, returnTo: applicationsScreen}
			m.screen = confirmScreen
		}
	case "esc":
		m.screen, m.cursor = dashboardScreen, 0
	}
	return m, nil
}

func (m *Model) viewApplications() (string, string, string) {
	items := sortedApplications(m.cfg.Applications)
	if len(items) == 0 {
		return "Applications", "No applications yet.\n\nAdd the applications you regularly use, then include them in a workspace.\n\n" + m.theme.Accent.Render("[A] Add Application"), "A Add   Esc Back   ? Help   Q Quit"
	}
	start, end := visibleRange(len(items), m.cursor, m.height)
	var body strings.Builder
	for i := start; i < end; i++ {
		marker := "  "
		line := items[i].Name
		if i == m.cursor {
			marker = "● "
			line = m.theme.Accent.Render(marker + line)
		} else {
			line = marker + line
		}
		body.WriteString(line + "\n")
	}
	selected := items[m.cursor]
	body.WriteString("\n" + m.theme.Muted.Render(fmt.Sprintf("%d applications configured.", len(items))) + "\n\n")
	body.WriteString("Path\n" + truncate(selected.Path, max(12, m.width-8)))
	if len(selected.Arguments) > 0 {
		body.WriteString("\nArguments\n" + truncate(app.FormatArguments(selected.Arguments), max(12, m.width-8)))
	}
	return "Applications", body.String(), "↑↓ Navigate   A Add   E/Enter Edit   D Delete   Esc Back"
}

func (m *Model) openApplicationForm(item *app.Application) {
	labels := []string{"Name", "Path", "Arguments"}
	fields := make([]textinput.Model, len(labels))
	for i, label := range labels {
		field := textinput.New()
		field.Prompt = ""
		field.Placeholder = label
		field.CharLimit = 2048
		field.Width = max(20, min(70, m.width-8))
		fields[i] = field
	}
	id := ""
	if item != nil {
		id = item.ID
		fields[0].SetValue(item.Name)
		fields[1].SetValue(item.Path)
		fields[2].SetValue(app.FormatArguments(item.Arguments))
	}
	fields[0].Focus()
	m.appForm = applicationForm{id: id, fields: fields}
	m.screen = applicationFormScreen
}

func (m *Model) updateApplicationForm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	form := &m.appForm
	switch key.String() {
	case "esc":
		m.screen, m.cursor = applicationsScreen, 0
		return m, nil
	case "tab", "down":
		form.fields[form.focus].Blur()
		form.focus = moveCursor(form.focus, len(form.fields), 1)
		return m, form.fields[form.focus].Focus()
	case "shift+tab", "up":
		form.fields[form.focus].Blur()
		form.focus = moveCursor(form.focus, len(form.fields), -1)
		return m, form.fields[form.focus].Focus()
	case "enter":
		if form.focus < len(form.fields)-1 {
			form.fields[form.focus].Blur()
			form.focus++
			return m, form.fields[form.focus].Focus()
		}
		arguments, err := app.ParseArguments(form.fields[2].Value())
		if err != nil {
			m.errMessage = err.Error()
			return m, nil
		}
		input := app.Application{Name: form.fields[0].Value(), Path: form.fields[1].Value(), Arguments: arguments}
		if form.id == "" {
			_, err = m.config.AddApplication(input)
		} else {
			_, err = m.config.UpdateApplication(form.id, input)
		}
		if err != nil {
			m.errMessage = err.Error()
			return m, nil
		}
		if err := m.refresh(); err != nil {
			m.errMessage = err.Error()
			return m, nil
		}
		m.screen, m.cursor = applicationsScreen, 0
		m.notice = "Application saved."
		return m, nil
	}
	var command tea.Cmd
	form.fields[form.focus], command = form.fields[form.focus].Update(key)
	return m, command
}

func (m *Model) viewApplicationForm() (string, string, string) {
	title := "Add Application"
	if m.appForm.id != "" {
		title = "Edit Application"
	}
	labels := []string{"Name", "Executable / application path", "Arguments (optional; quotes supported, no shell evaluation)"}
	var body strings.Builder
	for i, field := range m.appForm.fields {
		label := labels[i]
		if i == m.appForm.focus {
			label = m.theme.Accent.Render("● " + label)
		}
		body.WriteString(label + "\n" + field.View() + "\n\n")
	}
	return title, strings.TrimSpace(body.String()), "Tab/↑↓ Next field   Enter Continue/Save   Esc Cancel"
}
