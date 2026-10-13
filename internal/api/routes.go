package api

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"local-music/internal/catalog"
	"local-music/internal/library"
)

var songIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func New(db *sql.DB, webFiles fs.FS, libraryService *catalog.Service, scanner *library.Scanner) http.Handler {
	mux := http.NewServeMux()
	registerOrganizationRoutes(mux, libraryService)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/songs", func(w http.ResponseWriter, r *http.Request) {
		songs, err := libraryService.Songs(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, songs)
	})
	mux.HandleFunc("POST /api/scan", func(w http.ResponseWriter, r *http.Request) {
		count, err := libraryService.Scan(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"songs": count})
	})
	mux.HandleFunc("GET /api/songs/{id}/audio", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !songIDPattern.MatchString(id) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identificador de canción no válido"})
			return
		}
		rel, err := libraryService.AudioPath(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "canción no disponible; vuelve a escanear la biblioteca"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no se pudo consultar la canción"})
			return
		}
		file, info, err := scanner.Open(rel)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "el archivo no existe o no se puede abrir"})
			return
		}
		defer file.Close()
		extension := strings.ToLower(filepath.Ext(rel))
		contentType := map[string]string{".flac": "audio/flac", ".opus": "audio/ogg", ".ogg": "audio/ogg", ".m4a": "audio/mp4", ".aac": "audio/aac"}[extension]
		if contentType == "" {
			contentType = mime.TypeByExtension(extension)
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, filepath.Base(rel), info.ModTime(), file)
	})
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(static)))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
