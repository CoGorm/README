// Command readme renders a README markdown file in the terminal.
//
//	readme              # find ./README.md (or readme.md, .github/README.md, …)
//	readme docs/api.md  # render a specific file
//	readme ../project   # find the readme in another directory
//	cat notes.md | readme
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CoGorm/README/internal/find"
	"github.com/CoGorm/README/internal/render"
	"github.com/CoGorm/README/internal/tui"
	"golang.org/x/term"
)

const version = "0.1.0"

// maxWidth stops the text from stretching into an unreadable line length on
// very wide terminals.
const maxWidth = 100

type options struct {
	style   string
	width   int
	noPager bool
	version bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		if !errors.Is(err, errQuiet) {
			fmt.Fprintln(os.Stderr, "readme: "+err.Error())
		}
		os.Exit(1)
	}
}

// errQuiet aborts with a non-zero status without printing anything further.
var errQuiet = errors.New("quiet")

func run(argv []string) error {
	opts, args, err := parseArgs(argv)
	if err != nil {
		return err
	}
	if opts.version {
		fmt.Println("readme " + version)
		return nil
	}

	title, source, err := readSource(args)
	if err != nil {
		return err
	}

	width := opts.width
	if width == 0 {
		width = detectWidth()
	}

	doc, err := render.Markdown(source, width, opts.style)
	if err != nil {
		return err
	}

	if opts.noPager || !isTerminal(os.Stdout) || fits(doc) {
		fmt.Println(doc)
		return nil
	}

	// Let the pager re-render on resize so the text reflows with the window.
	return tui.Run(title, func(w int) (string, error) {
		if opts.width != 0 {
			w = opts.width
		}
		return render.Markdown(source, clampWidth(w), opts.style)
	})
}

// readSource resolves the arguments into a display title and markdown bytes,
// reading stdin when it is piped in or "-" is given explicitly.
func readSource(args []string) (string, []byte, error) {
	if len(args) > 1 {
		return "", nil, fmt.Errorf("expected at most one file, got %d", len(args))
	}

	if (len(args) == 1 && args[0] == "-") || (len(args) == 0 && hasPipedStdin()) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", nil, fmt.Errorf("reading stdin: %w", err)
		}
		if len(data) > 0 || len(args) == 1 {
			return "stdin", data, nil
		}
		// Empty pipe: fall through and look for a readme on disk instead.
	}

	target := "."
	if len(args) == 1 {
		target = args[0]
	}
	path, err := find.Locate(target)
	if err != nil {
		var notFound *find.NotFoundError
		if errors.As(err, &notFound) {
			return "", nil, fmt.Errorf("%w\ntry: readme <file>, or readme - to read from a pipe", err)
		}
		return "", nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return displayName(path), data, nil
}

// displayName shortens a path for the pager title, preferring a path relative
// to the working directory when the file lives underneath it.
func displayName(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || strings.HasPrefix(rel, "..") || len(rel) >= len(path) {
		return path
	}
	return rel
}

// parseArgs reads the flags by hand so that short and long forms stay together
// and flags may follow the file name.
func parseArgs(argv []string) (options, []string, error) {
	opts := options{style: "auto"}
	var args []string

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		// A lone "-" means stdin, and everything after "--" is a file name.
		if arg == "-" {
			args = append(args, arg)
			continue
		}
		if arg == "--" {
			args = append(args, argv[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") {
			args = append(args, arg)
			continue
		}

		name, inline, hasInline := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		value := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			if i+1 >= len(argv) {
				return "", fmt.Errorf("flag %s needs a value", arg)
			}
			i++
			return argv[i], nil
		}

		switch name {
		case "h", "help":
			usage(os.Stdout)
			return opts, nil, errQuiet
		case "v", "version":
			opts.version = true
		case "n", "no-pager", "plain":
			opts.noPager = true
		case "s", "style":
			v, err := value()
			if err != nil {
				return opts, nil, err
			}
			opts.style = v
		case "w", "width":
			v, err := value()
			if err != nil {
				return opts, nil, err
			}
			n, err := parseWidth(v)
			if err != nil {
				return opts, nil, err
			}
			opts.width = n
		default:
			usage(os.Stderr)
			return opts, nil, fmt.Errorf("unknown flag %s", arg)
		}
	}
	return opts, args, nil
}

func parseWidth(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid width %q", s)
	}
	return n, nil
}

func usage(w io.Writer) {
	fmt.Fprint(w, `readme — read a README in your terminal

usage:
  readme [flags] [file|directory|-]

with no argument it looks for README.md (and readme.md, README, .github/README.md,
docs/README.md, …) in the current directory.

flags:
  -s, --style <name>   auto, dark, light, dracula, tokyo-night, pink, ascii, notty,
                       or a path to a glamour style JSON file (default: auto)
  -w, --width <n>      wrap width in columns (default: terminal width, max 100)
  -n, --no-pager       print and exit instead of opening the pager
  -v, --version        print the version
  -h, --help           show this help

in the pager:
  j/k, ↑/↓   scroll          d/u   half page      g/G   top/bottom
  /          search          n/N   next/previous  q     quit
`)
}

// detectWidth returns a comfortable wrap width for the current terminal.
func detectWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return clampWidth(w)
}

func clampWidth(w int) int {
	if w > maxWidth {
		return maxWidth
	}
	if w < render.MinWidth {
		return render.MinWidth
	}
	return w
}

// fits reports whether the document is short enough to print without paging.
func fits(doc string) bool {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return false
	}
	return strings.Count(doc, "\n")+1 <= h-1
}

func isTerminal(f *os.File) bool { return term.IsTerminal(int(f.Fd())) }

// stdinWait is how long an argument-less run waits for a pipe to say something
// before deciding nothing was piped in. Interactive shells never pay it, since
// a terminal fails the check above.
const stdinWait = 250 * time.Millisecond

// hasPipedStdin reports whether stdin carries a document to render. A terminal
// never does. A pipe usually does, but a process launched from a script can
// inherit an idle pipe that nobody ever writes to, so we wait only briefly for
// one to produce something rather than blocking forever.
func hasPipedStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice != 0 {
		return false
	}
	if info.Mode().IsRegular() {
		return true // a redirect from a file is readable straight away
	}
	return stdinReadable(stdinWait)
}
