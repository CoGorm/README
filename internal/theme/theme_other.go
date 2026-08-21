//go:build !unix

package theme

import "time"

// queryDark does not probe the terminal on platforms where we cannot poll the
// input descriptor; Dark falls back to $COLORFGBG there.
func queryDark(time.Duration) (dark, ok bool) { return false, false }
