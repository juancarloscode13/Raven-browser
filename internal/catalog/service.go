package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"local-music/internal/library"
)

type Service struct {
	db      *sql.DB
	scanner *library.Scanner
	scanMu  sync.Mutex
}

func New(db *sql.DB, scanner *library.Scanner) *Service { return &Service{db: db, scanner: scanner} }

func (s *Service) Scan(ctx context.Context) (int, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	songs, err := s.scanner.Scan()
	if err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("iniciar actualización del catálogo: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE songs SET available = 0, updated_at = CURRENT_TIMESTAMP`); err != nil {
		return 0, fmt.Errorf("marcar archivos ausentes: %w", err)
	}
	for _, song := range songs {
		_, err := tx.ExecContext(ctx, `INSERT INTO songs(id,path,title,artist,format,available)
VALUES(?,?,?,?,?,1)
ON CONFLICT(path) DO UPDATE SET title=excluded.title, artist=excluded.artist, format=excluded.format,
available=1, updated_at=CURRENT_TIMESTAMP`, song.ID, song.Path, song.Title, song.Artist, song.Format)
		if err != nil {
			return 0, fmt.Errorf("actualizar canción %q: %w", song.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("guardar catálogo: %w", err)
	}
	return len(songs), nil
}

func (s *Service) Songs(ctx context.Context) ([]library.Song, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id,s.path,s.title,s.artist,s.format,s.available,t.name
FROM songs s LEFT JOIN song_tags st ON st.song_id=s.id LEFT JOIN tags t ON t.id=st.tag_id
ORDER BY s.title COLLATE NOCASE,s.artist COLLATE NOCASE,s.path,t.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("consultar catálogo: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

// AudioPath returns only the relative path of a catalogued, currently available song.
func (s *Service) AudioPath(ctx context.Context, id string) (string, error) {
	var path string
	err := s.db.QueryRowContext(ctx, `SELECT path FROM songs WHERE id = ? AND available = 1`, id).Scan(&path)
	if err != nil {
		return "", err
	}
	return path, nil
}
