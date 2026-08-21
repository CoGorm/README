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
