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

	if _, err := os.Stdout.WriteString("\x1b]11;?\x1b\\"); err != nil {
		return false, false
	}

	spec, ok := readOSC(in, 11, timeout)
	if !ok {
		return false, false
	}
	c := ansi.XParseColor(spec)
	if c == nil {
		return false, false
	}
	return uv.BackgroundColorEvent{Color: c}.IsDark(), true
}

// readOSC collects input until it holds a complete OSC reply for ps, or until
// the deadline passes. It returns the reply's payload.
func readOSC(fd, ps int, timeout time.Duration) (string, bool) {
	prefix := []byte("\x1b]" + itoa(ps) + ";")
	deadline := time.Now().Add(timeout)

	var buf []byte
	chunk := make([]byte, 256)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return "", false
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, int(remaining.Milliseconds()))
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			return "", false
		}
		read, err := unix.Read(fd, chunk)
		if err == unix.EINTR {
			continue
		}
		if err != nil || read <= 0 {
			return "", false
		}
		buf = append(buf, chunk[:read]...)
		if payload, done := splitOSC(buf, prefix); done {
			return payload, true
		}
	}
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
