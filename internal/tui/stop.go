package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/launchline/launchline/internal/app"
)

type stopState struct {
	generation int
	summary    app.StopSummary
	done       bool
}

type stopFinishedMsg struct {
	generation int
	summary    app.StopSummary
	err        error
}

func (m *Model) beginStop(reference string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	generation := m.stop.generation + 1
	m.stop = stopState{generation: generation}
	m.screen = stoppingScreen
	return func() tea.Msg {
		summary, err := m.launcher.Stop(ctx, reference)
		return stopFinishedMsg{generation: generation, summary: summary, err: err}
	}
}

func (m *Model) updateStopping(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case stopFinishedMsg:
		if msg.generation != m.stop.generation {
			return m, nil
		}
		m.stop.summary, m.stop.done = msg.summary, true
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if msg.err != nil {
			m.errMessage = msg.err.Error()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			if msg.String() == "enter" && !m.stop.done {
				return m, nil
			}
			if m.cancel != nil {
				m.cancel()
				m.cancel = nil
			}
			m.screen, m.cursor = dashboardScreen, 0
			m.prompt.Focus()
		case "up":
			m.cursor = moveCursor(m.cursor, len(m.stop.summary.Results), -1)
		case "down":
			m.cursor = moveCursor(m.cursor, len(m.stop.summary.Results), 1)
		}
	}
	return m, nil
}

func (m *Model) viewStopping() (string, string, string) {
	if !m.stop.done {
		return "Stop Workspace", "Requesting application closure…", "Esc Cancel"
	}
	summary := m.stop.summary
	var lines []string
	count := max(1, m.height-17)
	start, end := visibleRangeRows(len(summary.Results), m.cursor, count)
	for i := start; i < end; i++ {
		result := summary.Results[i]
		line := "· " + result.Application.Name + " — no tracked process running"
		if result.Err != nil {
			line = "× " + result.Application.Name + " — " + result.Err.Error()
		} else if result.Requested > 0 {
			line = "✓ " + result.Application.Name + " — close requested"
		}
		lines = append(lines, truncate(line, m.contentWidth()))
	}
	if len(summary.Results) == 0 && m.errMessage == "" {
		lines = append(lines, "This workspace has no applications.")
	}
	lines = append(lines, "", fmt.Sprintf("%d process close requests sent.", summary.Requested()), truncate("Check apps for save prompts; requests do not confirm exit.", m.contentWidth()))
	return truncate("Stop Workspace — "+summary.Workspace.Name, m.contentWidth()), strings.Join(lines, "\n"), "↑↓ Results   Enter/Esc Back"
}
