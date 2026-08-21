// Package find locates README files on disk.
package find

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// extRank orders the extensions we are willing to treat as a readme, best
// first. The empty string matches an extensionless file such as plain README.
var extRank = []string{".md", ".markdown", ".mdown", ".mkdn", ".mkd", ".rst", ".txt", ""}

// subDirs are the conventional places a project hides its readme when the
// repository root does not hold one.
var subDirs = []string{".github", "docs", "doc"}

// NotFoundError reports that no readme turned up in any of the searched
// locations.
type NotFoundError struct {
	Searched []string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("no readme found in %s", strings.Join(e.Searched, ", "))
}

// Locate resolves target into a path to render. A target that names a file is
// used as-is, a directory is searched for a readme, and a bare name such as
// "contributing" is resolved against the known extensions.
func Locate(target string) (string, error) {
	if target == "" {
		target = "."
	}

	info, err := os.Stat(target)
	switch {
	case err == nil && !info.IsDir():
		return target, nil
	case err == nil:
		return InDir(target)
	case !os.IsNotExist(err):
		return "", err
	}

	// The target does not exist verbatim. Treat it as a base name and try the
	// extensions we know about, so "readme CONTRIBUTING" finds CONTRIBUTING.md.
	if path, ok := withExtension(target); ok {
		return path, nil
	}
	return "", fmt.Errorf("%s: no such file or directory", target)
}

// InDir returns the best readme inside dir, falling back to the conventional
// subdirectories before giving up.
func InDir(dir string) (string, error) {
	searched := []string{prettyDir(dir)}
	if path, ok := best(dir); ok {
		return path, nil
	}
	for _, sub := range subDirs {
		candidate := filepath.Join(dir, sub)
		if info, err := os.Stat(candidate); err != nil || !info.IsDir() {
			continue
		}
		searched = append(searched, prettyDir(candidate))
		if path, ok := best(candidate); ok {
			return path, nil
		}
	}
	return "", &NotFoundError{Searched: searched}
}

// best picks the highest ranked readme in dir.
func best(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	bestName, bestScore := "", -1
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if s := score(entry.Name()); s > bestScore {
			bestName, bestScore = entry.Name(), s
		}
	}
	if bestName == "" {
		return "", false
	}
	return filepath.Join(dir, bestName), true
}

// score rates a file name as a readme candidate, or returns -1 if it is not one.
// Better extensions win first; the conventional all-caps spelling breaks ties.
func score(name string) int {
	ext := filepath.Ext(name)
	if !strings.EqualFold(strings.TrimSuffix(name, ext), "readme") {
		return -1
	}
	rank := indexFold(extRank, ext)
	if rank < 0 {
		return -1
	}
	points := (len(extRank) - rank) * 10
	switch {
	case strings.HasPrefix(name, "README"):
		points += 2
	case strings.HasPrefix(name, "Readme"):
		points++
	}
	return points
}

// withExtension looks for base with each known extension appended.
func withExtension(base string) (string, bool) {
	for _, ext := range extRank {
		if ext == "" {
			continue
		}
		candidate := base + ext
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func indexFold(haystack []string, needle string) int {
	for i, s := range haystack {
		if strings.EqualFold(s, needle) {
			return i
		}
	}
	return -1
}

// prettyDir renders a directory for error messages, collapsing "." to "./".
func prettyDir(dir string) string {
	if dir == "." {
		return "./"
	}
	return dir
}
