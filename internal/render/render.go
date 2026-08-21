// Package render turns markdown into styled terminal output.
package render

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
)

// Styles are the built-in glamour styles this tool exposes by name.
var Styles = []string{"auto", "dark", "light", "dracula", "tokyo-night", "pink", "ascii", "notty"}

// MinWidth keeps rendering readable on very narrow terminals.
const MinWidth = 40

// Markdown renders src at the given width. style is a built-in style name or a
// path to a glamour style JSON file; resolve "auto" with AutoStyle first.
func Markdown(src []byte, width int, style string) (string, error) {
	if width < MinWidth {
		width = MinWidth
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
		glamour.WithStylePath(style),
	)
	if err != nil {
		return "", fmt.Errorf("unknown style %q (try one of: %s)", style, strings.Join(Styles, ", "))
	}
	defer r.Close() //nolint:errcheck

	out, err := r.RenderBytes(src)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(out), "\n"), nil
}

// AutoStyle names the built-in style for a terminal, given whether output is
// going to one at all and whether its background is dark.
func AutoStyle(isTerminal, isDark bool) string {
	switch {
	case !isTerminal:
		return "notty"
	case isDark:
		return "dark"
	default:
		return "light"
	}
}

// IsDarkStyle reports whether a style name describes a dark background, so the
// pager's own chrome can match a style the user asked for by name instead of
// asking the terminal a question we already know the answer to.
func IsDarkStyle(style string) bool {
	switch style {
	case "light", "ascii":
		return false
	default:
		return true
	}
}
