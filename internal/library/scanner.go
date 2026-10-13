package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var supported = map[string]bool{
	".mp3": true, ".wav": true, ".flac": true, ".ogg": true,
	".opus": true, ".m4a": true, ".aac": true,
}

type Song struct {
	ID        string   `json:"id"`
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Format    string   `json:"format"`
	Available bool     `json:"available"`
	Tags      []string `json:"tags,omitempty"`
}

type Scanner struct{ root string }

func NewScanner(root string) (*Scanner, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolver carpeta musical: %w", err)
	}
	return &Scanner{root: filepath.Clean(absolute)}, nil
}

func (s *Scanner) Root() string { return s.root }

// Open resolves a catalog path through os.Root so symlinks cannot escape the library.
func (s *Scanner) Open(rel string) (*os.File, os.FileInfo, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if rel == "" || filepath.ToSlash(clean) != rel || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, nil, fmt.Errorf("ruta de canción no válida")
	}
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, nil, fmt.Errorf("abrir biblioteca: %w", err)
	}
	defer root.Close()
	file, err := root.Open(clean)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return nil, nil, fmt.Errorf("el archivo de audio no es regular")
	}
	return file, info, nil
}

// Scan returns supported audio files using slash-separated paths relative to the root.
func (s *Scanner) Scan() ([]Song, error) {
	info, err := os.Stat(s.root)
	if err != nil {
		return nil, fmt.Errorf("acceder a la carpeta musical %q: %w", s.root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("la ruta musical %q no es una carpeta", s.root)
	}
	var songs []Song
	err = filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("leer %q: %w", path, walkErr)
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !supported[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}
		rel, err := filepath.Rel(s.root, path)
		if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("ruta de audio fuera de la biblioteca: %q", path)
		}
		rel = filepath.ToSlash(rel)
		title, artist, format := ParseFilename(entry.Name())
		hash := sha256.Sum256([]byte(rel))
		songs = append(songs, Song{ID: hex.EncodeToString(hash[:]), Path: rel, Title: title, Artist: artist, Format: format, Available: true})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("escanear biblioteca: %w", err)
	}
	return songs, nil
}
