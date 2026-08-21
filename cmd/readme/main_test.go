package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		argv     []string
		wantOpts options
		wantArgs []string
	}{
		{"no arguments", nil, options{style: "auto"}, nil},
		{"a file", []string{"a.md"}, options{style: "auto"}, []string{"a.md"}},
		{"long flags", []string{"--style", "dark", "--width", "70"}, options{style: "dark", width: 70}, nil},
		{"short flags", []string{"-s", "light", "-w", "70"}, options{style: "light", width: 70}, nil},
		{"inline values", []string{"--style=dark", "-w=70"}, options{style: "dark", width: 70}, nil},
		{"flags after the file", []string{"a.md", "-n"}, options{style: "auto", noPager: true}, []string{"a.md"}},
		{"stdin", []string{"-"}, options{style: "auto"}, []string{"-"}},
		{"end of flags", []string{"--", "-weird.md"}, options{style: "auto"}, []string{"-weird.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, args, err := parseArgs(tt.argv)
			if err != nil {
				t.Fatalf("parseArgs: %v", err)
			}
			if opts != tt.wantOpts {
				t.Errorf("opts = %+v, want %+v", opts, tt.wantOpts)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	for _, argv := range [][]string{
		{"--nope"},
		{"--width"},
		{"--width", "zero"},
		{"--width", "-3"},
	} {
		if _, _, err := parseArgs(argv); err == nil {
			t.Errorf("parseArgs(%v) accepted bad input", argv)
		}
	}
}

func TestParseArgsHelpIsQuietAndSuccessful(t *testing.T) {
	// --help prints usage itself, so run() must exit without a second message.
	_, _, err := parseArgs([]string{"--help"})
	if !errors.Is(err, errQuiet) {
		t.Errorf("parseArgs(--help) = %v, want errQuiet", err)
	}
}

func TestReadSourceRejectsExtraArguments(t *testing.T) {
	if _, _, err := readSource([]string{"a.md", "b.md"}); err == nil {
		t.Error("readSource accepted two files")
	}
}

func TestDisplayNameIsRelativeToTheWorkingDirectory(t *testing.T) {
	chdir(t, t.TempDir())
	// Read the directory back so the comparison uses the resolved path.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(dir, "docs", "README.md")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := displayName(nested), filepath.Join("docs", "README.md"); got != want {
		t.Errorf("displayName = %q, want %q", got, want)
	}
	// A path outside the working directory stays absolute rather than turning
	// into a pile of "..".
	outside := filepath.Join(filepath.Dir(dir), "elsewhere", "README.md")
	if got := displayName(outside); got != outside {
		t.Errorf("displayName = %q, want %q", got, outside)
	}
}

// chdir moves into dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) }) //nolint:errcheck
}

func TestClampWidth(t *testing.T) {
	if got := clampWidth(1000); got != maxWidth {
		t.Errorf("clampWidth(1000) = %d, want %d", got, maxWidth)
	}
	if got := clampWidth(5); got != 40 {
		t.Errorf("clampWidth(5) = %d, want the minimum", got)
	}
	if got := clampWidth(72); got != 72 {
		t.Errorf("clampWidth(72) = %d, want 72", got)
	}
}

func TestRendererCachesByWidth(t *testing.T) {
	source := []byte("# hi\n\nsome text\n")
	draw := newRenderer(source, "notty", 0)

	first, err := draw(80)
	if err != nil {
		t.Fatal(err)
	}

	same, err := draw(80)
	if err != nil {
		t.Fatal(err)
	}
	if same != first {
		t.Error("the same width produced a different document")
	}

	wider, err := draw(60)
	if err != nil {
		t.Fatal(err)
	}
	if wider == first {
		t.Error("a new width returned the cached document")
	}
	if back, _ := draw(80); back != first {
		t.Error("returning to a previous width did not reproduce its document")
	}
}

func TestRendererHonoursExplicitWidth(t *testing.T) {
	source := []byte(strings.Repeat("word ", 100))
	draw := newRenderer(source, "notty", 50)

	// The window size must not override --width.
	fixed, err := draw(120)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(fixed, "\n") {
		if len(line) > 50 {
			t.Fatalf("line is %d columns, want <= 50: %q", len(line), line)
		}
	}
}
