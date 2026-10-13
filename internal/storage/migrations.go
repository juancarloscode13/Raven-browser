package storage

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	sql     string
}

var migrations = []migration{{version: 1, sql: `
CREATE TABLE songs (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  artist TEXT NOT NULL,
  format TEXT NOT NULL,
  available INTEGER NOT NULL DEFAULT 1 CHECK (available IN (0, 1)),
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE playlists (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE playlist_songs (
  playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
  song_id TEXT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  position INTEGER NOT NULL CHECK (position >= 0),
  PRIMARY KEY (playlist_id, song_id),
  UNIQUE (playlist_id, position)
);
CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE song_tags (
  song_id TEXT NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
  tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (song_id, tag_id)
);
CREATE INDEX idx_songs_title ON songs(title COLLATE NOCASE);
CREATE INDEX idx_songs_artist ON songs(artist COLLATE NOCASE);
CREATE INDEX idx_songs_available ON songs(available);
CREATE INDEX idx_playlist_songs_order ON playlist_songs(playlist_id, position);
CREATE INDEX idx_song_tags_tag ON song_tags(tag_id, song_id);
`}}

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return fmt.Errorf("crear registro de migraciones: %w", err)
	}
	latest := 0
	for _, m := range migrations {
		if m.version > latest {
			latest = m.version
		}
	}
	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("leer versión del esquema: %w", err)
	}
	if current > latest {
		return fmt.Errorf("la base de datos usa la versión de esquema %d y esta aplicación solo admite hasta la %d", current, latest)
	}
	for _, m := range migrations {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, m.version).Scan(&exists); err != nil {
			return fmt.Errorf("consultar migración %d: %w", m.version, err)
		}
		if exists != 0 {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("iniciar migración %d: %w", m.version, err)
		}
		if _, err = tx.Exec(m.sql); err == nil {
			_, err = tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, m.version)
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("aplicar migración %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("confirmar migración %d: %w", m.version, err)
		}
	}
	return nil
}
