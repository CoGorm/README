package theme

import (
	"os"
	"testing"
)

func TestColorFGBGDark(t *testing.T) {
	tests := []struct {
		env       string
		wantDark  bool
		wantFound bool
	}{
		{"15;0", true, true},      // white on black
		{"0;15", false, true},     // black on white
		{"15;8", true, true},      // bright black counts as dark
		{"7;7", false, true},      // light grey background
		{"12;13;14", false, true}, // some terminals add a third field
		{"", false, false},
		{"nonsense", false, false},
		{"15;notanumber", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("COLORFGBG", tt.env)
			if tt.env == "" {
				os.Unsetenv("COLORFGBG")
			}
			dark, ok := colorFGBGDark()
			if ok != tt.wantFound {
				t.Fatalf("ok = %v, want %v", ok, tt.wantFound)
			}
			if ok && dark != tt.wantDark {
				t.Errorf("dark = %v, want %v", dark, tt.wantDark)
			}
		})
	}
}

func TestDarkFallsBackToColorFGBGWithoutATerminal(t *testing.T) {
	// The test binary's stdout is not a terminal, so no query is attempted and
	// Dark must answer from the environment instead of hanging.
	t.Setenv("COLORFGBG", "0;15")
	dark, ok := Dark()
	if !ok {
		t.Fatal("Dark found no answer despite COLORFGBG being set")
	}
	if dark {
		t.Error("Dark said dark for a white background")
	}
}
