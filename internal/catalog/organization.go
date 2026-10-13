package catalog

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"local-music/internal/library"
)

var (
	ErrInvalid  = errors.New("datos no válidos")
	ErrNotFound = errors.New("elemento no encontrado")
	ErrConflict = errors.New("ya existe un elemento con ese nombre")
)

type Playlist struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SongCount int    `json:"song_count"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Organization struct {
	Playlists []Playlist `json:"playlists"`
	Tags      []Tag      `json:"tags"`
}

func newID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generar identificador: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 60 {
		return "", fmt.Errorf("%w: el nombre debe tener entre 1 y 60 caracteres", ErrInvalid)
	}
	return name, nil
}

func (s *Service) Playlists(ctx context.Context) ([]Playlist, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.name,COUNT(ps.song_id) FROM playlists p LEFT JOIN playlist_songs ps ON ps.playlist_id=p.id GROUP BY p.id ORDER BY p.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("consultar listas: %w", err)
	}
	defer rows.Close()
	items := make([]Playlist, 0)
	for rows.Next() {
		var item Playlist
		if err := rows.Scan(&item.ID, &item.Name, &item.SongCount); err != nil {
			return nil, fmt.Errorf("leer listas: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leer listas: %w", err)
	}
	return items, nil
}

func (s *Service) CreatePlaylist(ctx context.Context, name string) (Playlist, error) {
	name, err := cleanName(name)
	if err != nil {
		return Playlist{}, err
	}
	item, err := s.createNamed(ctx, "playlists", name)
	return Playlist{ID: item.ID, Name: item.Name}, err
}

func (s *Service) RenamePlaylist(ctx context.Context, id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return s.renameNamed(ctx, "playlists", id, name)
}

func (s *Service) DeletePlaylist(ctx context.Context, id string) error {
	return s.deleteNamed(ctx, "playlists", id)
}

func (s *Service) PlaylistSongs(ctx context.Context, id string) ([]library.Song, error) {
	if err := s.exists(ctx, "playlists", id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id,s.path,s.title,s.artist,s.format,s.available,t.name FROM playlist_songs ps JOIN songs s ON s.id=ps.song_id LEFT JOIN song_tags st ON st.song_id=s.id LEFT JOIN tags t ON t.id=st.tag_id WHERE ps.playlist_id=? ORDER BY ps.position,t.name COLLATE NOCASE`, id)
	if err != nil {
		return nil, fmt.Errorf("consultar canciones de la lista: %w", err)
	}
	defer rows.Close()
	return scanSongs(rows)
}

func (s *Service) SetPlaylistMembership(ctx context.Context, playlistID, songID string, present bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar cambio de lista: %w", err)
	}
	defer tx.Rollback()
	if err := existsTx(ctx, tx, "playlists", playlistID); err != nil {
		return err
	}
	if err := existsTx(ctx, tx, "songs", songID); err != nil {
		return err
	}
	if present {
		_, err = tx.ExecContext(ctx, `INSERT INTO playlist_songs(playlist_id,song_id,position) SELECT ?,?,COALESCE(MAX(position)+1,0) FROM playlist_songs WHERE playlist_id=? ON CONFLICT(playlist_id,song_id) DO NOTHING`, playlistID, songID, playlistID)
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM playlist_songs WHERE playlist_id=? AND song_id=?`, playlistID, songID)
	}
	if err != nil {
		return fmt.Errorf("actualizar pertenencia a la lista: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("guardar pertenencia a la lista: %w", err)
	}
	return nil
}

func (s *Service) ReorderPlaylist(ctx context.Context, playlistID string, songIDs []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar orden de lista: %w", err)
	}
	defer tx.Rollback()
	if err := existsTx(ctx, tx, "playlists", playlistID); err != nil {
		return err
	}
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_songs WHERE playlist_id=?`, playlistID).Scan(&current); err != nil {
		return err
	}
	if current != len(songIDs) {
		return fmt.Errorf("%w: el orden debe incluir todas las canciones de la lista", ErrInvalid)
	}
	seen := make(map[string]struct{}, len(songIDs))
	for _, songID := range songIDs {
		if _, duplicate := seen[songID]; duplicate {
			return fmt.Errorf("%w: canción repetida", ErrInvalid)
		}
		seen[songID] = struct{}{}
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM playlist_songs WHERE playlist_id=? AND song_id=?`, playlistID, songID).Scan(&exists); err != nil {
			return err
		}
		if exists != 1 {
			return fmt.Errorf("%w: la canción no pertenece a la lista", ErrInvalid)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE playlist_songs SET position=position+? WHERE playlist_id=?`, len(songIDs), playlistID); err != nil {
		return err
	}
	for position, songID := range songIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE playlist_songs SET position=? WHERE playlist_id=? AND song_id=?`, position, playlistID, songID); err != nil {
			return fmt.Errorf("ordenar canciones: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Service) Tags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name FROM tags ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("consultar tags: %w", err)
	}
	defer rows.Close()
	tags := make([]Tag, 0)
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("leer tags: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leer tags: %w", err)
	}
	return tags, nil
}

func (s *Service) CreateTag(ctx context.Context, name string) (Tag, error) {
	name, err := cleanName(name)
	if err != nil {
		return Tag{}, err
	}
	item, err := s.createNamed(ctx, "tags", name)
	return Tag{ID: item.ID, Name: item.Name}, err
}

func (s *Service) DeleteTag(ctx context.Context, id string) error {
	return s.deleteNamed(ctx, "tags", id)
}

func (s *Service) SetTagMembership(ctx context.Context, songID, tagID string, present bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar cambio de tag: %w", err)
	}
	defer tx.Rollback()
	if err := existsTx(ctx, tx, "songs", songID); err != nil {
		return err
	}
	if err := existsTx(ctx, tx, "tags", tagID); err != nil {
		return err
	}
	if present {
		_, err = tx.ExecContext(ctx, `INSERT INTO song_tags(song_id,tag_id) VALUES(?,?) ON CONFLICT DO NOTHING`, songID, tagID)
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM song_tags WHERE song_id=? AND tag_id=?`, songID, tagID)
	}
	if err != nil {
		return fmt.Errorf("actualizar asignación de tag: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("guardar asignación de tag: %w", err)
	}
	return nil
}

func (s *Service) Organization(ctx context.Context, songID string) (Organization, error) {
	if err := s.exists(ctx, "songs", songID); err != nil {
		return Organization{}, err
	}
	result := Organization{Playlists: make([]Playlist, 0), Tags: make([]Tag, 0)}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.name FROM playlist_songs ps JOIN playlists p ON p.id=ps.playlist_id WHERE ps.song_id=? ORDER BY p.name COLLATE NOCASE`, songID)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var item Playlist
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return result, err
		}
		result.Playlists = append(result.Playlists, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT t.id,t.name FROM song_tags st JOIN tags t ON t.id=st.tag_id WHERE st.song_id=? ORDER BY t.name COLLATE NOCASE`, songID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Tag
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return result, err
		}
		result.Tags = append(result.Tags, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	return result, nil
}

type named struct {
	ID   string
	Name string
}

func (s *Service) createNamed(ctx context.Context, table, name string) (named, error) {
	id, err := newID()
	if err != nil {
		return named{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return named{}, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE name=? COLLATE NOCASE`, name).Scan(&count); err != nil {
		return named{}, err
	}
	if count != 0 {
		return named{}, ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO `+table+`(id,name) VALUES(?,?)`, id, name); err != nil {
		return named{}, fmt.Errorf("crear elemento: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return named{}, err
	}
	return named{ID: id, Name: name}, nil
}

func (s *Service) renameNamed(ctx context.Context, table, id, name string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE `+table+` SET name=? WHERE id=? AND NOT EXISTS(SELECT 1 FROM `+table+` other WHERE other.id<>? AND other.name=? COLLATE NOCASE)`, name, id, id, name)
	if err != nil {
		return fmt.Errorf("renombrar elemento: %w", err)
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		if err := s.exists(ctx, table, id); err != nil {
			return err
		}
		return ErrConflict
	}
	return nil
}

func (s *Service) deleteNamed(ctx context.Context, table, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("eliminar elemento: %w", err)
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) exists(ctx context.Context, table, id string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE id=?`, id).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func existsTx(ctx context.Context, tx *sql.Tx, table, id string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE id=?`, id).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSongs(rows *sql.Rows) ([]library.Song, error) {
	songs := make([]library.Song, 0)
	byID := make(map[string]int)
	for rows.Next() {
		var song library.Song
		var available int
		var tag sql.NullString
		if err := rows.Scan(&song.ID, &song.Path, &song.Title, &song.Artist, &song.Format, &available, &tag); err != nil {
			return nil, err
		}
		song.Available = available != 0
		if index, exists := byID[song.ID]; exists {
			if tag.Valid {
				songs[index].Tags = append(songs[index].Tags, tag.String)
			}
			continue
		}
		song.Tags = make([]string, 0)
		if tag.Valid {
			song.Tags = append(song.Tags, tag.String)
		}
		byID[song.ID] = len(songs)
		songs = append(songs, song)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return songs, nil
}
