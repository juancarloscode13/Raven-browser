package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchemaAndAppliesMigrationsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "music.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration count = %d, want 1", count)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration count after second run = %d, want 1", count)
	}
}

func TestOpenDoesNotOverwriteCorruptDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "music.db")
	original := []byte("this is not a sqlite database")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if db, err := Open(path); err == nil {
		db.Close()
		t.Fatal("Open accepted a corrupt database")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("database changed after failed open: %q", after)
	}
}

func TestMigrateRejectsNewerSchemaAndEnforcesForeignKeys(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "music.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO schema_migrations(version) VALUES(999)`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err == nil {
		t.Fatal("Migrate accepted a newer schema version")
	}
	if _, err := db.Exec(`INSERT INTO playlist_songs(playlist_id,song_id,position) VALUES('missing','missing',0)`); err == nil {
		t.Fatal("foreign key constraint was not enforced")
	}
}
