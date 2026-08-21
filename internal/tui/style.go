package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// styles is the pager's chrome, resolved for one terminal background. Lip Gloss
// v2 has no adaptive colour that decides at render time, so the light and dark
// choices are made once, up front.
type styles struct {
	bar          lipgloss.Style
	title        lipgloss.Style
	status       lipgloss.Style
	hint         lipgloss.Style
	match        lipgloss.Style
	currentMatch lipgloss.Style
}

func newStyles(isDark bool) styles {
	pick := lipgloss.LightDark(isDark)
	c := func(light, dark string) color.Color {
		return pick(lipgloss.Color(light), lipgloss.Color(dark))
	}

	bar := lipgloss.NewStyle().Background(c("#E4E4E4", "#303030"))
	return styles{
		bar: bar,
		title: bar.
			Foreground(c("#FFFFFF", "#1A1A1A")).
			Background(c("#7D56F4", "#A78BFA")).
			Bold(true).
			Padding(0, 1),
		status:       bar.Foreground(c("#3A3A3A", "#DDDDDD")),
		hint:         bar.Foreground(c("#6C6C6C", "#9A9A9A")),
		match:        lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(lipgloss.Color("#FFD866")),
		currentMatch: lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(lipgloss.Color("#FF9F43")).Bold(true),
	}
}
