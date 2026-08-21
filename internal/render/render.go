// Package render turns markdown into styled terminal output.
package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// Styles are the built-in glamour styles this tool exposes by name.
var Styles = []string{"auto", "dark", "light", "dracula", "tokyo-night", "pink", "ascii", "notty"}

// MinWidth keeps rendering readable on very narrow terminals.
const MinWidth = 40

// Markdown renders src at the given width. style is either "auto", one of the
// built-in style names, or a path to a glamour style JSON file.
func Markdown(src []byte, width int, style string) (string, error) {
	if width < MinWidth {
		width = MinWidth
	}

	opts := []glamour.TermRendererOption{
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	}
	if style == "" || style == "auto" {
		opts = append(opts, glamour.WithAutoStyle())
	} else {
		opts = append(opts, glamour.WithStylePath(style))
	}

	r, err := glamour.NewTermRenderer(opts...)
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
