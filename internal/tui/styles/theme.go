package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Title    lipgloss.Style
	Accent   lipgloss.Style
	Muted    lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Disabled lipgloss.Style
	Footer   lipgloss.Style
}

func New() Theme {
	accent := lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#A78BFA"}
	muted := lipgloss.AdaptiveColor{Light: "#5F6368", Dark: "#8B949E"}
	return Theme{
		Title:    lipgloss.NewStyle().Bold(true),
		Accent:   lipgloss.NewStyle().Foreground(accent).Bold(true),
		Muted:    lipgloss.NewStyle().Foreground(muted),
		Success:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#18794E", Dark: "#3FB950"}),
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}),
		Disabled: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8C959F", Dark: "#6E7681"}),
		Footer:   lipgloss.NewStyle().Foreground(muted),
	}
}
