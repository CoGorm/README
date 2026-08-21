package tui

import "github.com/charmbracelet/lipgloss"

// The pager chrome uses adaptive colors so it sits correctly on light and dark
// terminals without the user picking a theme.
var (
	barStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#E4E4E4", Dark: "#303030"})

	titleStyle = barStyle.
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A1A"}).
			Background(lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#A78BFA"}).
			Bold(true).
			Padding(0, 1)

	statusStyle = barStyle.
			Foreground(lipgloss.AdaptiveColor{Light: "#3A3A3A", Dark: "#DDDDDD"})

	hintStyle = barStyle.
			Foreground(lipgloss.AdaptiveColor{Light: "#6C6C6C", Dark: "#9A9A9A"})

	matchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A1A1A")).
			Background(lipgloss.Color("#FFD866"))

	currentMatchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1A1A1A")).
				Background(lipgloss.Color("#FF9F43")).
				Bold(true)
)
