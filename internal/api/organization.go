package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"local-music/internal/catalog"
)

var organizationIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func registerOrganizationRoutes(mux *http.ServeMux, service *catalog.Service) {
	mux.HandleFunc("GET /api/playlists", func(w http.ResponseWriter, r *http.Request) {
		items, err := service.Playlists(r.Context())
		respond(w, items, err)
	})
	mux.HandleFunc("POST /api/playlists", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		item, err := service.CreatePlaylist(r.Context(), body.Name)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	})
	mux.HandleFunc("PATCH /api/playlists/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validOrganizationID(w, id) {
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if err := service.RenamePlaylist(r.Context(), id, body.Name); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /api/playlists/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validOrganizationID(w, id) {
			return
		}
		if err := service.DeletePlaylist(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/playlists/{id}/songs", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validOrganizationID(w, id) {
			return
		}
		items, err := service.PlaylistSongs(r.Context(), id)
		respond(w, items, err)
	})
	mux.HandleFunc("PUT /api/playlists/{id}/songs", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validOrganizationID(w, id) {
			return
		}
		var body struct {
			SongIDs []string `json:"song_ids"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		if body.SongIDs == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "song_ids debe ser una lista"})
			return
		}
		for _, songID := range body.SongIDs {
			if !validSongID(w, songID) {
				return
			}
		}
		if err := service.ReorderPlaylist(r.Context(), id, body.SongIDs); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("PUT /api/playlists/{id}/songs/{songID}", func(w http.ResponseWriter, r *http.Request) {
		id, songID := r.PathValue("id"), r.PathValue("songID")
		if !validOrganizationID(w, id) || !validSongID(w, songID) {
			return
		}
		if err := service.SetPlaylistMembership(r.Context(), id, songID, true); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /api/playlists/{id}/songs/{songID}", func(w http.ResponseWriter, r *http.Request) {
		id, songID := r.PathValue("id"), r.PathValue("songID")
		if !validOrganizationID(w, id) || !validSongID(w, songID) {
			return
		}
		if err := service.SetPlaylistMembership(r.Context(), id, songID, false); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/tags", func(w http.ResponseWriter, r *http.Request) {
		items, err := service.Tags(r.Context())
		respond(w, items, err)
	})
	mux.HandleFunc("POST /api/tags", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if !decodeBody(w, r, &body) {
			return
		}
		item, err := service.CreateTag(r.Context(), body.Name)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	})
	mux.HandleFunc("DELETE /api/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validOrganizationID(w, id) {
			return
		}
		if err := service.DeleteTag(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/songs/{id}/organization", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validSongID(w, id) {
			return
		}
		item, err := service.Organization(r.Context(), id)
		respond(w, item, err)
	})
	mux.HandleFunc("PUT /api/songs/{id}/tags/{tagID}", func(w http.ResponseWriter, r *http.Request) {
		songID, tagID := r.PathValue("id"), r.PathValue("tagID")
		if !validSongID(w, songID) || !validOrganizationID(w, tagID) {
			return
		}
		if err := service.SetTagMembership(r.Context(), songID, tagID, true); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /api/songs/{id}/tags/{tagID}", func(w http.ResponseWriter, r *http.Request) {
		songID, tagID := r.PathValue("id"), r.PathValue("tagID")
		if !validSongID(w, songID) || !validOrganizationID(w, tagID) {
			return
		}
		if err := service.SetTagMembership(r.Context(), songID, tagID, false); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cuerpo JSON no válido"})
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "se esperaba un único objeto JSON"})
		return false
	}
	return true
}

func validOrganizationID(w http.ResponseWriter, id string) bool {
	if organizationIDPattern.MatchString(id) {
		return true
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identificador no válido"})
	return false
}

func validSongID(w http.ResponseWriter, id string) bool {
	if songIDPattern.MatchString(id) {
		return true
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identificador de canción no válido"})
	return false
}

func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, catalog.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, catalog.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		status = http.StatusNotFound
	case errors.Is(err, catalog.ErrConflict):
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": fmt.Sprint(err)})
}
