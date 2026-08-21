// Package tui provides the full-screen pager used when stdout is a terminal.
package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// RenderFunc produces the styled document for a given wrap width. The pager
// calls it again whenever the terminal is resized so the text reflows.
type RenderFunc func(width int) (string, error)

// Run displays a document in an alternate-screen pager until the user quits.
// isDark selects the chrome that suits the terminal background.
func Run(title string, isDark bool, render RenderFunc) error {
	m := &model{
		title:  title,
		render: render,
		input:  newInput(),
		style:  newStyles(isDark),
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return m.err
}

func newInput() textinput.Model {
	in := textinput.New()
	in.Prompt = "/"
	in.Placeholder = "search"
	in.CharLimit = 128
	return in
}

type keyMap struct {
	Top    key.Binding
	Bottom key.Binding
	Search key.Binding
	Next   key.Binding
	Prev   key.Binding
	Help   key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Top:    key.NewBinding(key.WithKeys("g", "home")),
	Bottom: key.NewBinding(key.WithKeys("G", "end")),
	Search: key.NewBinding(key.WithKeys("/")),
	Next:   key.NewBinding(key.WithKeys("n")),
	Prev:   key.NewBinding(key.WithKeys("N")),
	Help:   key.NewBinding(key.WithKeys("?")),
	Quit:   key.NewBinding(key.WithKeys("q", "esc", "ctrl+c")),
}

type model struct {
	title  string
	render RenderFunc
	style  styles

	viewport viewport.Model
	input    textinput.Model
	ready    bool
	width    int
	height   int

	lines []string // rendered document, one entry per line
	plain []string // the same lines with ANSI stripped, lowercased

	searching bool   // the search prompt has focus
	query     string // the committed query
	matches   []int  // indices into lines that contain query
	current   int    // index into matches
	showHelp  bool
	err       error
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.resize(msg.Width, msg.Height)

	case tea.KeyPressMsg:
		if m.searching {
			return m, m.updateSearch(msg)
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Top):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, keys.Bottom):
			m.viewport.GotoBottom()
			return m, nil
		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			m.layout()
			return m, nil
		case key.Matches(msg, keys.Search):
			m.searching = true
			m.input.SetValue("")
			return m, m.input.Focus()
		case key.Matches(msg, keys.Next):
			m.jump(1)
			return m, nil
		case key.Matches(msg, keys.Prev):
			m.jump(-1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// updateSearch handles keys while the search prompt is open.
func (m *model) updateSearch(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.searching = false
		m.input.Blur()
		m.setQuery("")
		return nil
	case "enter":
		m.searching = false
		m.input.Blur()
		m.setQuery(strings.TrimSpace(m.input.Value()))
		m.jump(0)
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.setQuery(strings.TrimSpace(m.input.Value()))
	return cmd
}

// resize re-renders the document for the new terminal size and lays out the
// viewport between the header and footer.
func (m *model) resize(width, height int) tea.Cmd {
	m.width, m.height = width, height

	doc, err := m.render(width)
	if err != nil {
		m.err = err
		return tea.Quit
	}
	m.lines = strings.Split(doc, "\n")
	m.plain = make([]string, len(m.lines))
	for i, line := range m.lines {
		m.plain[i] = strings.ToLower(ansi.Strip(line))
	}

	m.layout()
	m.findMatches()
	m.viewport.SetContent(m.content())
	return nil
}

// layout sizes the viewport to whatever space the header and footer leave.
func (m *model) layout() {
	m.input.SetWidth(m.width - 4)

	body := m.height - lipgloss.Height(m.header()) - lipgloss.Height(m.footer())
	if body < 1 {
		body = 1
	}
	if !m.ready {
		m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(body))
		m.ready = true
		return
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(body)
}

// setQuery recomputes the match set for a new query, keeping the viewport
// content in sync so highlights follow the user's typing.
func (m *model) setQuery(q string) {
	if q == m.query {
		return
	}
	m.query = q
	m.findMatches()
	m.viewport.SetContent(m.content())
}

func (m *model) findMatches() {
	m.matches, m.current = nil, 0
	if m.query == "" {
		return
	}
	needle := strings.ToLower(m.query)
	for i, line := range m.plain {
		if strings.Contains(line, needle) {
			m.matches = append(m.matches, i)
		}
	}
}

// jump moves to the match delta positions away, wrapping at both ends. A delta
// of zero selects the first match at or below the current scroll position.
func (m *model) jump(delta int) {
	if len(m.matches) == 0 {
		return
	}
	if delta == 0 {
		m.current = 0
		for i, line := range m.matches {
			if line >= m.viewport.YOffset() {
				m.current = i
				break
			}
		}
	} else {
		m.current = (m.current + delta + len(m.matches)) % len(m.matches)
	}
	m.viewport.SetContent(m.content())
	m.scrollTo(m.matches[m.current])
}

// scrollTo brings line into view, resting it a third of the way down the
// viewport so surrounding context stays visible.
func (m *model) scrollTo(line int) {
	offset := line - m.viewport.Height()/3
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
}

// content returns the document with search matches highlighted. Highlighted
// lines lose their original colors, which is the price of styling a substring
// inside text that already carries ANSI escapes.
func (m *model) content() string {
	if m.query == "" || len(m.matches) == 0 {
		return strings.Join(m.lines, "\n")
	}
	out := make([]string, len(m.lines))
	copy(out, m.lines)
	for i, line := range m.matches {
		out[line] = m.highlight(ansi.Strip(m.lines[line]), i == m.current)
	}
	return strings.Join(out, "\n")
}

// highlight styles every occurrence of the query within a plain-text line.
func (m *model) highlight(line string, current bool) string {
	style := m.style.match
	if current {
		style = m.style.currentMatch
	}
	var b strings.Builder
	rest := line
	for {
		i := strings.Index(strings.ToLower(rest), strings.ToLower(m.query))
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:i])
		b.WriteString(style.Render(rest[i : i+len(m.query)]))
		rest = rest[i+len(m.query):]
	}
}

func (m *model) View() tea.View {
	view := tea.NewView("")
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	if !m.ready {
		view.SetContent("\n  loading…")
		return view
	}
	view.SetContent(m.header() + "\n" + m.viewport.View() + "\n" + m.footer())
	return view
}

func (m *model) header() string {
	// Keep room for at least a sliver of bar so the title never wraps.
	title := m.style.title.Render(ansi.Truncate(m.title, max(m.width-6, 1), "…"))
	gap := m.width - lipgloss.Width(title)
	if gap < 0 {
		gap = 0
	}
	return title + m.style.bar.Render(strings.Repeat(" ", gap))
}

func (m *model) footer() string {
	bottom := m.statusBar()
	if m.searching {
		bottom = m.input.View()
	}
	if !m.showHelp {
		return bottom
	}
	// The help gets its own line; squeezing it beside the status truncates it
	// on anything narrower than a very wide terminal.
	help := "j/k scroll · d/u half page · g/G top/bottom · / search · n/N next/prev · q quit"
	return m.style.hint.Width(m.width).Render(" "+ansi.Truncate(help, max(m.width-2, 1), "…")) + "\n" + bottom
}

// statusBar is the bottom line: a hint on the left, position and search state
// on the right.
func (m *model) statusBar() string {
	status := fmt.Sprintf("%3.0f%%", m.viewport.ScrollPercent()*100)
	if m.query != "" {
		if len(m.matches) == 0 {
			status = "no matches  " + status
		} else {
			status = fmt.Sprintf("%d/%d for %q  %s", m.current+1, len(m.matches), m.query, status)
		}
	}

	right := m.style.status.Render(" " + status + " ")
	left := m.style.hint.Render(" ? help ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		left, gap = "", max(m.width-lipgloss.Width(right), 0)
	}
	return left + m.style.bar.Render(strings.Repeat(" ", gap)) + right
}
