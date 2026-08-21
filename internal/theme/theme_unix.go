//go:build unix

package theme

import (
	"bytes"
	"os"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// queryDark asks the terminal for its background colour and waits at most
// timeout for the reply.
func queryDark(timeout time.Duration) (dark, ok bool) {
	in, out := int(os.Stdin.Fd()), int(os.Stdout.Fd())
	if !term.IsTerminal(in) || !term.IsTerminal(out) {
		return false, false
	}

	// The reply arrives on stdin as ordinary input, so the terminal has to stop
	// line-buffering and echoing it first.
	state, err := term.MakeRaw(in)
	if err != nil {
		return false, false
	}
	defer term.Restore(in, state) //nolint:errcheck

	// The cursor position report is a sentinel. Terminals answer queries in the
	// order they arrive, and nearly all of them answer this one even when they
	// ignore OSC 11 — tmux is the common example. Its reply therefore means "no
	// background colour is coming", which turns the usual timeout into an
	// instant answer. Reading through to it also keeps the reply out of the
	// pager's input, where it would arrive as a stray key press.
	if _, err := os.Stdout.WriteString("\x1b]11;?\x1b\\\x1b[6n"); err != nil {
		return false, false
	}

	spec, ok := readReply(in, timeout)
	if !ok {
		return false, false
	}
	c := ansi.XParseColor(spec)
	if c == nil {
		return false, false
	}
	return uv.BackgroundColorEvent{Color: c}.IsDark(), true
}

// readReply collects input until the cursor position report lands or the
// deadline passes, then reports any OSC 11 payload that came with it.
func readReply(fd int, timeout time.Duration) (string, bool) {
	deadline := time.Now().Add(timeout)

	var buf []byte
	chunk := make([]byte, 256)
	for !hasCPR(buf) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(remaining.Milliseconds()))
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			break
		}
		read, err := unix.Read(fd, chunk)
		if err == unix.EINTR {
			continue
		}
		if err != nil || read <= 0 {
			break
		}
		buf = append(buf, chunk[:read]...)
	}
	return splitOSC(buf, []byte("\x1b]11;"))
}

// splitOSC pulls the payload out of an OSC reply, which ends at either a BEL or
// a string terminator.
func splitOSC(buf, prefix []byte) (string, bool) {
	i := bytes.Index(buf, prefix)
	if i < 0 {
		return "", false
	}
	rest := buf[i+len(prefix):]
	if j := bytes.IndexByte(rest, 0x07); j >= 0 {
		return string(rest[:j]), true
	}
	if j := bytes.Index(rest, []byte("\x1b\\")); j >= 0 {
		return string(rest[:j]), true
	}
	return "", false
}

// hasCPR reports whether buf holds a cursor position report, ESC [ row ; col R.
func hasCPR(buf []byte) bool {
	for i := 0; i+2 < len(buf); i++ {
		if buf[i] != 0x1b || buf[i+1] != '[' {
			continue
		}
		for j := i + 2; j < len(buf); j++ {
			c := buf[j]
			if c == 'R' {
				return true
			}
			if (c < '0' || c > '9') && c != ';' {
				break
			}
		}
	}
	return false
}
