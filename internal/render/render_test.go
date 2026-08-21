package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMarkdownWrapsToWidth(t *testing.T) {
	src := []byte(strings.Repeat("word ", 200))
	for _, width := range []int{40, 60, 100} {
		out, err := Markdown(src, width, "notty")
		if err != nil {
			t.Fatalf("Markdown: %v", err)
		}
		for _, line := range strings.Split(out, "\n") {
			if w := ansi.StringWidth(line); w > width {
				t.Errorf("width %d: line is %d columns: %q", width, w, line)
			}
		}
	}
}

func TestMarkdownClampsTinyWidths(t *testing.T) {
	if _, err := Markdown([]byte("# hi"), 1, "notty"); err != nil {
		t.Errorf("Markdown at width 1: %v", err)
	}
}

func TestMarkdownRejectsUnknownStyle(t *testing.T) {
	_, err := Markdown([]byte("# hi"), 80, "not-a-style")
	if err == nil {
		t.Fatal("Markdown accepted an unknown style")
	}
	// The message should point the user at the styles that do work.
	for _, want := range []string{"not-a-style", "dracula"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestMarkdownAcceptsEveryAdvertisedStyle(t *testing.T) {
	for _, style := range Styles {
		if style == "auto" {
			continue // depends on the terminal, not testable here
		}
		if _, err := Markdown([]byte("# hi"), 80, style); err != nil {
			t.Errorf("style %s: %v", style, err)
		}
	}
}

func TestMarkdownRendersEmoji(t *testing.T) {
	out, err := Markdown([]byte("hello :tada:"), 80, "notty")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "🎉") {
		t.Errorf("emoji shortcode was not expanded: %q", out)
	}
}

func TestAutoStyle(t *testing.T) {
	tests := []struct {
		terminal, dark bool
		want           string
	}{
		{true, true, "dark"},
		{true, false, "light"},
		{false, true, "notty"},
		{false, false, "notty"},
	}
	for _, tt := range tests {
		if got := AutoStyle(tt.terminal, tt.dark); got != tt.want {
			t.Errorf("AutoStyle(terminal=%v, dark=%v) = %s, want %s", tt.terminal, tt.dark, got, tt.want)
		}
	}
}

func TestIsDarkStyle(t *testing.T) {
	for _, style := range []string{"dark", "dracula", "tokyo-night", "notty", "my-theme.json"} {
		if !IsDarkStyle(style) {
			t.Errorf("IsDarkStyle(%q) = false, want true", style)
		}
	}
	for _, style := range []string{"light", "ascii"} {
		if IsDarkStyle(style) {
			t.Errorf("IsDarkStyle(%q) = true, want false", style)
		}
	}
}

func TestEveryAutoStyleResultRenders(t *testing.T) {
	// AutoStyle must only ever name styles Markdown accepts.
	for _, terminal := range []bool{true, false} {
		for _, dark := range []bool{true, false} {
			style := AutoStyle(terminal, dark)
			if _, err := Markdown([]byte("# hi"), 80, style); err != nil {
				t.Errorf("style %q from AutoStyle: %v", style, err)
			}
		}
	}
}

func TestMarkdownWrapsLinksToWidth(t *testing.T) {
	// Glamour v2 emits OSC 8 hyperlinks, whose escape bytes must not count
	// towards the visible width or the pager's layout drifts.
	src := []byte("A [very long link label indeed](https://example.com/a/rather/long/path) " +
		strings.Repeat("and more words ", 20))
	out, err := Markdown(src, 60, "dark")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\x1b]8;") {
		t.Skip("this glamour build does not emit hyperlinks")
	}
	for _, line := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(line); w > 60 {
			t.Errorf("line is %d columns wide: %q", w, line)
		}
	}
	if strings.Contains(ansi.Strip(out), "\x1b") {
		t.Error("ansi.Strip left escape bytes behind, so search would see them")
	}
}
