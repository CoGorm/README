// Package theme works out whether the terminal has a dark background.
//
// Terminals answer that question over an OSC 11 escape sequence, and some
// never answer at all — tmux and a number of SSH and editor terminals among
// them. Charm's older stack waited five seconds for those, which is why this
// package exists: the wait is ours to bound.
package theme

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Timeout is how long we wait for the terminal to answer. Terminals that do
// answer take a round trip, so this only has to cover the link; terminals that
// never answer cost exactly this much and nothing more.
const Timeout = 250 * time.Millisecond

// Dark reports whether the terminal background is dark, and whether we managed
// to find out at all. When ok is false the caller should pick a default rather
// than trust dark.
func Dark() (dark, ok bool) {
	if d, ok := queryDark(Timeout); ok {
		return d, true
	}
	return colorFGBGDark()
}

// colorFGBGDark reads $COLORFGBG, the convention some terminals use to publish
// their palette without being asked. Its last field is the background as an
// ANSI colour number, where 0-6 and 8 are the dark ones.
func colorFGBGDark() (dark, ok bool) {
	v := os.Getenv("COLORFGBG")
	if !strings.Contains(v, ";") {
		return false, false
	}
	fields := strings.Split(v, ";")
	n, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		return false, false
	}
	return n < 7 || n == 8, true
}
