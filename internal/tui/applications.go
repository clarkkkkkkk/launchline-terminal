package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/discovery"
)

type applicationChoice struct {
	key         string
	name        string
	configured  *app.Application
	discovered  *discovery.Application
	unavailable bool
}

func (m *Model) applicationChoices(query string) []applicationChoice {
	linked := map[string]app.Application{}
	for _, item := range m.cfg.Applications {
		if item.DiscoveryID != "" {
			linked[item.DiscoveryID] = item
		}
	}
	choices := make([]applicationChoice, 0, len(m.catalog.Applications)+len(m.cfg.Applications))
	seen := map[string]bool{}
	for i := range m.catalog.Applications {
		item := m.catalog.Applications[i]
		choice := applicationChoice{key: item.ID, name: item.Name, discovered: &item}
		if configured, ok := linked[item.ID]; ok {
			copy := configured
			choice.key, choice.configured = configured.ID, &copy
		}
		choices = append(choices, choice)
		seen[item.ID] = true
	}
	for i := range m.cfg.Applications {
		item := m.cfg.Applications[i]
		if item.DiscoveryID != "" && seen[item.DiscoveryID] {
			continue
		}
		copy := item
		choices = append(choices, applicationChoice{key: item.ID, name: item.Name, configured: &copy, unavailable: item.Unavailable})
	}
	query = strings.ToLower(strings.TrimSpace(query))
	filtered := choices[:0]
	for _, choice := range choices {
		if query == "" || strings.Contains(strings.ToLower(choice.name), query) {
			filtered = append(filtered, choice)
		}
	}
	choices = filtered
	sortChoices(choices)
	return choices
}

func sortChoices(items []applicationChoice) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && strings.ToLower(items[j].name) < strings.ToLower(items[j-1].name); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func (m *Model) updateApplications(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.applicationChoices(m.search.Value())
	if m.search.Focused() {
		switch key.String() {
		case "up":
			m.search.Blur()
			m.cursor = max(0, len(items)-1)
			return m, nil
		case "down":
			m.search.Blur()
			m.cursor = 0
			return m, nil
		case "enter":
			if len(items) > 0 {
				m.openApplicationDetails(items[m.cursor])
			}
			return m, nil
		case "esc":
			m.search.SetValue("")
			m.search.Blur()
			m.cursor = 0
			return m, nil
		}
		var command tea.Cmd
		previousQuery := m.search.Value()
		m.search, command = m.search.Update(key)
		items = m.applicationChoices(m.search.Value())
		if m.search.Value() != previousQuery {
			m.cursor = 0
		} else if m.cursor >= len(items) {
			m.cursor = max(0, len(items)-1)
		}
		return m, command
	}

	switch strings.ToLower(key.String()) {
	case "up":
		m.cursor = moveCursor(m.cursor, len(items), -1)
		return m, nil
	case "down":
		m.cursor = moveCursor(m.cursor, len(items), 1)
		return m, nil
	case "/":
		return m, m.search.Focus()
	case "a":
		m.openApplicationForm(nil)
		return m, nil
	case "d":
		if len(items) > 0 && items[m.cursor].configured != nil {
			item := items[m.cursor].configured
			m.confirm = confirmState{kind: "application", id: item.ID, name: item.Name, returnTo: applicationsScreen}
			m.screen = confirmScreen
		} else if len(items) > 0 {
			m.notice = "Discovered catalog entries cannot be deleted. They disappear when no longer installed."
		}
		return m, nil
	case "enter":
		if len(items) > 0 {
			m.openApplicationDetails(items[m.cursor])
		}
		return m, nil
	case "e":
		if len(items) > 0 && items[m.cursor].configured != nil && items[m.cursor].configured.Manual() {
			m.openApplicationForm(items[m.cursor].configured)
		} else if len(items) > 0 {
			m.openApplicationDetails(items[m.cursor])
			m.notice = "Discovered applications are read-only. Use /add to register an editable manual target."
		}
		return m, nil
	case " ", "space":
		return m, nil
	case "?":
		m.returnTo, m.screen = applicationsScreen, helpScreen
		return m, nil
	case "esc":
		if m.search.Value() != "" {
			m.search.SetValue("")
			m.cursor = 0
			return m, nil
		}
		m.search.Blur()
		m.prompt.Focus()
		m.screen, m.cursor = dashboardScreen, 0
		return m, nil
	}
	return m, nil
}

func (m *Model) viewApplications() (string, string, string) {
	items := m.applicationChoices(m.search.Value())
	var body strings.Builder
	body.WriteString(m.description(fmt.Sprintf("Applications — %d discovered · %d linked/manual", len(m.catalog.Applications), len(m.cfg.Applications))) + "\n\n")
	body.WriteString(m.theme.Muted.Render("Search applications") + "\n" + m.search.View() + "\n\n")
	footer := "↑↓ Navigate   / Search   Enter Details   E Edit   D Delete   A Add   Esc Back"
	if m.search.Focused() {
		footer = "Type Search   ↑↓ Results   Enter Details   Esc Done"
	}
	if len(items) == 0 {
		if len(m.catalog.Applications) == 0 && len(m.cfg.Applications) == 0 {
			body.WriteString(m.theme.Title.Render("No applications found.") + "\n" + m.description("Use /refresh to scan this device, or /add to register a custom binary manually."))
		} else {
			body.WriteString(m.theme.Title.Render("No matching applications.") + "\n" + m.description("Clear the search to see the complete cached catalog."))
		}
		return "Application Catalog", body.String(), footer
	}
	start, end := visibleRangeRows(len(items), m.cursor, m.height-18)
	for i := start; i < end; i++ {
		meta := "discovered"
		if items[i].configured != nil {
			if items[i].configured.Manual() {
				meta = "manual"
			} else {
				meta = "discovered · linked"
			}
		}
		if items[i].unavailable {
			meta = "not currently available"
		}
		line := items[i].name + "  " + m.theme.Muted.Render(meta)
		prefix := "  "
		if i == m.cursor {
			prefix = m.theme.Accent.Render("● ")
			line = m.theme.Focus.Render(line)
		}
		body.WriteString(prefix + line + "\n")
	}
	selected := items[m.cursor]
	target := ""
	if selected.discovered != nil {
		target = selected.discovered.Target
	} else if selected.configured != nil {
		target = selected.configured.Path
	}
	if target != "" {
		body.WriteString("\n" + m.theme.Muted.Render("Target") + "  " + truncate(target, max(12, m.contentWidth()-8)))
	}
	return "Application Catalog", body.String(), footer
}

func (m *Model) openApplicationDetails(item applicationChoice) {
	m.appDetail = item
	m.search.Blur()
	m.screen = applicationDetailsScreen
}

func (m *Model) updateApplicationDetails(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(key.String()) {
	case "e":
		if m.appDetail.configured != nil && m.appDetail.configured.Manual() {
			m.openApplicationForm(m.appDetail.configured)
		}
	case "esc":
		m.screen = applicationsScreen
	case "?":
		m.returnTo, m.screen = applicationDetailsScreen, helpScreen
	}
	return m, nil
}

func (m *Model) viewApplicationDetails() (string, string, string) {
	item := m.appDetail
	target, kind, source, platformName, arguments := "", "executable", "Manual registration", runtime.GOOS, []string(nil)
	status, registration := "Available", "Discovered"
	editable := item.configured != nil && item.configured.Manual()
	if item.discovered != nil {
		target = item.discovered.Target
		kind = item.discovered.Kind
		source = item.discovered.Source
		platformName = item.discovered.Platform
		arguments = item.discovered.Arguments
	}
	if item.configured != nil {
		if target == "" {
			target = item.configured.Path
			kind = item.configured.Kind
			arguments = item.configured.Arguments
			if item.configured.Source != "" {
				source = item.configured.Source
			}
		}
		if item.configured.Manual() {
			registration = "Manual · editable"
		} else {
			registration = "Discovered · linked to Launchline"
		}
	}
	if kind == "" {
		kind = "executable"
	}
	if item.unavailable || (item.configured != nil && item.configured.Unavailable) {
		status = "Not currently available"
	}
	args := app.FormatArguments(arguments)
	if args == "" {
		args = "None"
	}
	rows := []string{
		m.theme.Muted.Render("Registration") + "  " + registration,
		m.theme.Muted.Render("Status") + "        " + status,
		m.theme.Muted.Render("Target") + "        " + truncate(target, max(12, m.contentWidth()-14)),
		m.theme.Muted.Render("Arguments") + "     " + truncate(args, max(12, m.contentWidth()-14)),
		m.theme.Muted.Render("Kind") + "          " + kind,
		m.theme.Muted.Render("Source") + "        " + source,
		m.theme.Muted.Render("Platform") + "      " + platformName,
	}
	footer := "Esc Back"
	if editable {
		footer = "E Edit   Esc Back"
	} else {
		rows = append(rows, "", m.description("This application is managed by local discovery and cannot be edited here. Use /add to register an editable manual target."))
	}
	return "Application Details — " + item.name, strings.Join(rows, "\n"), footer
}

func (m *Model) openApplicationForm(item *app.Application) {
	labels := []string{"Name", "Path", "Arguments"}
	fields := make([]textinput.Model, len(labels))
	for i, label := range labels {
		field := textinput.New()
		field.Prompt = ""
		field.Placeholder = label
		field.CharLimit = 2048
		field.Width = max(12, min(70, m.contentWidth()-2))
		field.Cursor.Style = m.theme.Accent
		field.TextStyle = m.theme.Command
		field.PlaceholderStyle = m.theme.Muted
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
		m.search.Blur()
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
		m.search.Blur()
		return m, nil
	}
	var command tea.Cmd
	form.fields[form.focus], command = form.fields[form.focus].Update(key)
	return m, command
}

func (m *Model) viewApplicationForm() (string, string, string) {
	title := "Application Editor — New Application"
	description := "Register a custom application that discovery did not find."
	if m.appForm.id != "" {
		title = "Application Editor — " + m.appForm.fields[0].Value()
		description = "Update this manual application without changing its workspace identity."
	}
	labels := []string{"Name", "Executable / application path", "Arguments (optional; quotes supported, no shell evaluation)"}
	var body strings.Builder
	body.WriteString(m.description(description) + "\n\n")
	for i, field := range m.appForm.fields {
		label := labels[i]
		if i == m.appForm.focus {
			label = m.theme.Accent.Render("● ") + m.theme.Focus.Render(label)
		} else {
			label = "  " + label
		}
		body.WriteString(label + "\n" + field.View() + "\n\n")
	}
	return title, strings.TrimSpace(body.String()), "Tab/↑↓ Next Field   Enter Continue/Save   Esc Cancel"
}
