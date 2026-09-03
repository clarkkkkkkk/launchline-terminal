package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Logo     lipgloss.Style
	Eyebrow  lipgloss.Style
	Title    lipgloss.Style
	Focus    lipgloss.Style
	Accent   lipgloss.Style
	Command  lipgloss.Style
	Muted    lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style
	Disabled lipgloss.Style
	Hints    lipgloss.Style
	Status   lipgloss.Style
	Divider  lipgloss.Style
}

func New() Theme {
	bright := lipgloss.AdaptiveColor{Light: "#111318", Dark: "#F4F4F5"}
	accent := lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	muted := lipgloss.AdaptiveColor{Light: "#60646C", Dark: "#8B8D98"}
	return Theme{
		Logo:     lipgloss.NewStyle().Foreground(bright).Bold(true),
		Eyebrow:  lipgloss.NewStyle().Foreground(bright).Bold(true),
		Title:    lipgloss.NewStyle().Foreground(bright).Bold(true),
		Focus:    lipgloss.NewStyle().Foreground(bright).Bold(true),
		Accent:   lipgloss.NewStyle().Foreground(accent).Bold(true),
		Command:  lipgloss.NewStyle().Foreground(bright),
		Muted:    lipgloss.NewStyle().Foreground(muted),
		Success:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#18794E", Dark: "#3FB950"}),
		Warning:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}),
		Error:    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}),
		Disabled: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8C959F", Dark: "#6E7681"}),
		Hints:    lipgloss.NewStyle().Foreground(muted),
		Status:   lipgloss.NewStyle().Foreground(muted),
		Divider:  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D0D1D4", Dark: "#45464D"}),
	}
}
