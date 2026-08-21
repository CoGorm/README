//go:build !unix

package main

import "time"

// stdinReadable assumes a redirected stdin carries input on platforms where we
// cannot cheaply poll for it.
func stdinReadable(time.Duration) bool { return true }
