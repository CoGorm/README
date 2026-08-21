package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// Tests run without a terminal, where lipgloss would otherwise strip every
// escape sequence and make the highlighting assertions vacuous.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

// newTestModel builds a pager over doc, sized as if the terminal were 80x24.
func newTestModel(t *testing.T, doc string) *model {
	t.Helper()
	m := &model{
		title:  "test.md",
		render: func(int) (string, error) { return doc, nil },
		input:  newInput(),
	}
	if cmd := m.resize(80, 24); cmd != nil {
		t.Fatalf("resize returned %v, want nil", cmd)
	}
	return m
}

// press feeds a key to the model the way bubbletea would.
func press(m *model, s string) {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	m.Update(msg)
}

func numberedDoc(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "line " + strings.Repeat("x", i%3) + " number " + itoa(i)
	}
	return strings.Join(lines, "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestSearchFindsAndCyclesMatches(t *testing.T) {
	m := newTestModel(t, "alpha\nbeta\nalpha again\ngamma\nALPHA shouting\n")

	press(m, "/")
	if !m.searching {
		t.Fatal("/ did not open the search prompt")
	}
	press(m, "alpha")
	press(m, "enter")

	if m.searching {
		t.Error("enter did not close the search prompt")
	}
	if len(m.matches) != 3 {
		t.Fatalf("found %d matches, want 3 (search should ignore case)", len(m.matches))
	}
	if m.current != 0 {
		t.Errorf("current = %d, want the first match", m.current)
	}

	press(m, "n")
	if m.current != 1 {
		t.Errorf("after n current = %d, want 1", m.current)
	}
	press(m, "N")
	press(m, "N")
	if m.current != 2 {
		t.Errorf("N should wrap to the last match, got %d", m.current)
	}
}

func TestSearchEscapeClearsTheQuery(t *testing.T) {
	m := newTestModel(t, "alpha\nbeta\n")
	press(m, "/")
	press(m, "alpha")
	press(m, "esc")

	if m.searching {
		t.Error("esc left the prompt open")
	}
	if m.query != "" || len(m.matches) != 0 {
		t.Errorf("esc left query %q with %d matches", m.query, len(m.matches))
	}
}

func TestSearchScrollsTheMatchIntoView(t *testing.T) {
	m := newTestModel(t, numberedDoc(200))

	press(m, "/")
	press(m, "number 150")
	press(m, "enter")

	if len(m.matches) != 1 {
		t.Fatalf("found %d matches, want 1", len(m.matches))
	}
	line := m.matches[0]
	top, bottom := m.viewport.YOffset, m.viewport.YOffset+m.viewport.Height
	if line < top || line >= bottom {
		t.Errorf("match on line %d is outside the visible range [%d,%d)", line, top, bottom)
	}
}

func TestContentHighlightsMatchesWithoutChangingLineCount(t *testing.T) {
	doc := numberedDoc(50)
	m := newTestModel(t, doc)

	press(m, "/")
	press(m, "number 4")
	press(m, "enter")

	got := m.content()
	if want := strings.Count(doc, "\n"); strings.Count(got, "\n") != want {
		t.Errorf("highlighting changed the line count: got %d, want %d", strings.Count(got, "\n"), want)
	}
	if ansi.Strip(got) != doc {
		t.Error("highlighting changed the visible text")
	}
	if !strings.Contains(got, "\x1b[") {
		t.Error("no highlight was applied")
	}
}

func TestNavigationKeys(t *testing.T) {
	m := newTestModel(t, numberedDoc(200))

	press(m, "j")
	if m.viewport.YOffset == 0 {
		t.Error("j did not scroll down")
	}
	press(m, "k")
	if m.viewport.YOffset != 0 {
		t.Error("k did not scroll back up")
	}
	press(m, "G")
	if !m.viewport.AtBottom() {
		t.Error("G did not go to the bottom")
	}
	press(m, "g")
	if !m.viewport.AtTop() {
		t.Error("g did not go to the top")
	}
}

func TestHelpTogglesWithoutLosingViewportLines(t *testing.T) {
	m := newTestModel(t, numberedDoc(200))
	full := m.viewport.Height

	press(m, "?")
	if !m.showHelp {
		t.Fatal("? did not turn help on")
	}
	if m.viewport.Height != full-1 {
		t.Errorf("help line should cost exactly one row: %d -> %d", full, m.viewport.Height)
	}
	press(m, "?")
	if m.viewport.Height != full {
		t.Errorf("hiding help did not restore the viewport: %d, want %d", m.viewport.Height, full)
	}
}

func TestViewFitsTheTerminal(t *testing.T) {
	for _, keys := range [][]string{{}, {"?"}, {"/"}, {"?", "/"}} {
		m := newTestModel(t, numberedDoc(200))
		for _, k := range keys {
			press(m, k)
		}
		view := m.View()
		if lines := strings.Count(view, "\n") + 1; lines != 24 {
			t.Errorf("after %v the view is %d lines, want 24", keys, lines)
		}
		for _, line := range strings.Split(view, "\n") {
			if w := ansi.StringWidth(line); w > 80 {
				t.Errorf("after %v a line is %d columns wide, want <= 80", keys, w)
			}
		}
	}
}
