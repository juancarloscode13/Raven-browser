package library

import (
	"path/filepath"
	"strings"
)

// ParseFilename derives display fields from the name without reading audio metadata.
func ParseFilename(name string) (title, artist, format string) {
	base := filepath.Base(name)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	format = strings.ToLower(ext)
	separator := strings.LastIndex(stem, "_")
	if separator < 0 {
		return stem, "Desconocido", format
	}
	title, artist = strings.TrimSpace(stem[:separator]), strings.TrimSpace(stem[separator+1:])
	if title == "" {
		title = stem
	}
	if artist == "" {
		artist = "Desconocido"
	}
	return title, artist, format
}
