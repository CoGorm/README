//go:build unix

package main

import (
	"time"

	"golang.org/x/sys/unix"
)

// stdinReadable reports whether stdin has data or end-of-file waiting for us
// within wait. It polls the descriptor directly because os.Stdin is left in
// blocking mode, so SetReadDeadline is not available on it.
func stdinReadable(wait time.Duration) bool {
	fds := []unix.PollFd{{Fd: 0, Events: unix.POLLIN}}
	deadline := time.Now().Add(wait)
	for {
		n, err := unix.Poll(fds, int(time.Until(deadline).Milliseconds()))
		if err == unix.EINTR {
			continue // a signal interrupted the wait; finish out the deadline
		}
		return err == nil && n > 0
	}
}
