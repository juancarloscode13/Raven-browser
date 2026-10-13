package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerIncludesSupportedFormatsAndIgnoresOtherFiles(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Track_Artist.MP3", "nested/live_set.opus", "odd name.m4a", "notes.txt"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("sample"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	scanner, err := NewScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	songs, err := scanner.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 3 {
		t.Fatalf("got %d songs, want 3", len(songs))
	}
	for _, song := range songs {
		if filepath.IsAbs(song.Path) {
			t.Errorf("path %q is absolute", song.Path)
		}
	}
}

func TestOpenRejectsTraversalAndSymlinksOutsideLibrary(t *testing.T) {
	root := t.TempDir()
	scanner, err := NewScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	if file, _, err := scanner.Open("../secret.mp3"); err == nil {
		file.Close()
		t.Fatal("Open accepted a path outside the library")
	}
	if file, _, err := scanner.Open("nested/../secret.mp3"); err == nil {
		file.Close()
		t.Fatal("Open accepted a non-canonical path")
	}
	outside := filepath.Join(t.TempDir(), "secret.mp3")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.mp3")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if file, _, err := scanner.Open("linked.mp3"); err == nil {
		file.Close()
		t.Fatal("Open followed a symlink outside the library")
	}
}
