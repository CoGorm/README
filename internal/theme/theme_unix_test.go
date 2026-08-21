//go:build unix

package theme

import "testing"

func TestSplitOSC(t *testing.T) {
	prefix := []byte("\x1b]11;")
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"bel terminated", "\x1b]11;rgb:1c1c/1c1c/1c1c\x07", "rgb:1c1c/1c1c/1c1c", true},
		{"st terminated", "\x1b]11;rgb:ffff/ffff/ffff\x1b\\", "rgb:ffff/ffff/ffff", true},
		{"leading noise", "junk\x1b]11;rgb:0/0/0\x07", "rgb:0/0/0", true},
		{"still arriving", "\x1b]11;rgb:1c1c/1c", "", false},
		{"unrelated reply", "\x1b]10;rgb:0/0/0\x07", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := splitOSC([]byte(tt.in), prefix)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("payload = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestQueryDarkGivesUpWithoutATerminal(t *testing.T) {
	// Nothing to talk to means an immediate no, not a wait.
	if _, ok := queryDark(Timeout); ok {
		t.Error("queryDark claimed an answer with no terminal attached")
	}
}

func TestHasCPR(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"plain report", "\x1b[1;1R", true},
		{"two digit coords", "\x1b[24;90R", true},
		{"after an osc reply", "\x1b]11;rgb:0/0/0\x07\x1b[3;7R", true},
		{"nothing yet", "", false},
		{"partial", "\x1b[24;", false},
		{"osc reply only", "\x1b]11;rgb:0/0/0\x07", false},
		{"a different final byte", "\x1b[24;90H", false},
		{"escape with no bracket", "\x1bR", false},
		{"stray letters between", "\x1b[24;9xR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasCPR([]byte(tt.in)); got != tt.want {
				t.Errorf("hasCPR(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
