package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"local-music/internal/library"
	"local-music/internal/storage"
)

func TestScanRetainsMissingSongsAndRestoresThemByPath(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "Track_Artist.mp3")
	if err := os.WriteFile(track, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	scanner, err := library.NewScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(t.TempDir(), "music.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(db, scanner)
	ctx := context.Background()
	if count, err := service.Scan(ctx); err != nil || count != 1 {
		t.Fatalf("first scan = %d, %v", count, err)
	}
	songs, err := service.Songs(ctx)
	if err != nil || len(songs) != 1 {
		t.Fatalf("songs = %v, %v", songs, err)
	}
	originalID := songs[0].ID
	if _, err := db.Exec(`INSERT INTO playlists(id,name) VALUES('list','Favorites')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO playlist_songs(playlist_id,song_id,position) VALUES('list',?,0)`, originalID); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(track); err != nil {
		t.Fatal(err)
	}
	if count, err := service.Scan(ctx); err != nil || count != 0 {
		t.Fatalf("missing scan = %d, %v", count, err)
	}
	songs, err = service.Songs(ctx)
	if err != nil || len(songs) != 1 || songs[0].Available {
		t.Fatalf("missing song state = %v, %v", songs, err)
	}
	var relations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM playlist_songs WHERE song_id=?`, originalID).Scan(&relations); err != nil {
		t.Fatal(err)
	}
	if relations != 1 {
		t.Fatalf("playlist relations = %d, want 1", relations)
	}
	if err := os.WriteFile(track, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if count, err := service.Scan(ctx); err != nil || count != 1 {
		t.Fatalf("restored scan = %d, %v", count, err)
	}
	songs, err = service.Songs(ctx)
	if err != nil || len(songs) != 1 || !songs[0].Available || songs[0].ID != originalID {
		t.Fatalf("restored song = %v, %v", songs, err)
	}
}

func TestDifferentPathsDoNotShareIdentity(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{"one/Same_Title.mp3", "two/Same_Title.mp3"} {
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	scanner, _ := library.NewScanner(root)
	db, err := storage.Open(filepath.Join(t.TempDir(), "music.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(db, scanner)
	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	songs, err := service.Songs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 2 || songs[0].ID == songs[1].ID {
		t.Fatalf("expected distinct records, got %+v", songs)
	}
}

func TestFailedScanDoesNotMarkCatalogUnavailable(t *testing.T) {
	root := t.TempDir()
	track := filepath.Join(root, "Track_Artist.mp3")
	if err := os.WriteFile(track, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	scanner, err := library.NewScanner(root)
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(t.TempDir(), "music.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(db, scanner)
	if _, err := service.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(track); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(context.Background()); err == nil {
		t.Fatal("Scan succeeded after its root directory was removed")
	}
	songs, err := service.Songs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 || !songs[0].Available {
		t.Fatalf("failed scan changed availability: %+v", songs)
	}
}
