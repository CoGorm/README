package find

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// touch creates an empty file, making any missing parent directories.
func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInDirPrefersMarkdownAndConventionalCase(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{"only lowercase", []string{"readme.md"}, "readme.md"},
		{"markdown beats plain text", []string{"README.txt", "readme.md"}, "readme.md"},
		{"caps beats lowercase", []string{"readme.md", "README.md"}, "README.md"},
		{"extensionless is a last resort", []string{"README", "README.markdown"}, "README.markdown"},
		{"ignores other files", []string{"main.go", "readme.md"}, "readme.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				touch(t, filepath.Join(dir, f))
			}
			got, err := InDir(dir)
			if err != nil {
				t.Fatalf("InDir: %v", err)
			}
			if filepath.Base(got) != tt.want {
				t.Errorf("InDir = %s, want %s", filepath.Base(got), tt.want)
			}
		})
	}
}

func TestInDirFallsBackToSubDirectories(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "docs", "README.md"))
	touch(t, filepath.Join(dir, ".github", "README.md"))

	got, err := InDir(dir)
	if err != nil {
		t.Fatalf("InDir: %v", err)
	}
	// .github is searched before docs.
	if want := filepath.Join(dir, ".github", "README.md"); got != want {
		t.Errorf("InDir = %s, want %s", got, want)
	}
}

func TestInDirReportsWhereItLooked(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "docs", "notes.md"))

	_, err := InDir(dir)
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("InDir error = %v, want *NotFoundError", err)
	}
	if len(notFound.Searched) != 2 {
		t.Errorf("searched %v, want the directory and docs", notFound.Searched)
	}
}

func TestInDirIgnoresADirectoryNamedReadme(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "README.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InDir(dir); err == nil {
		t.Error("InDir accepted a directory as a readme")
	}
}

func TestLocate(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, "README.md"))
	touch(t, filepath.Join(dir, "CONTRIBUTING.md"))

	t.Run("file is used verbatim", func(t *testing.T) {
		path := filepath.Join(dir, "CONTRIBUTING.md")
		got, err := Locate(path)
		if err != nil || got != path {
			t.Errorf("Locate(%s) = %s, %v", path, got, err)
		}
	})

	t.Run("directory is searched", func(t *testing.T) {
		got, err := Locate(dir)
		if err != nil || got != filepath.Join(dir, "README.md") {
			t.Errorf("Locate(dir) = %s, %v", got, err)
		}
	})

	t.Run("bare name gains an extension", func(t *testing.T) {
		got, err := Locate(filepath.Join(dir, "CONTRIBUTING"))
		if err != nil || got != filepath.Join(dir, "CONTRIBUTING.md") {
			t.Errorf("Locate(bare) = %s, %v", got, err)
		}
	})

	t.Run("missing target errors", func(t *testing.T) {
		if _, err := Locate(filepath.Join(dir, "nope")); err == nil {
			t.Error("Locate accepted a missing target")
		}
	})
}
