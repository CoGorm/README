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
