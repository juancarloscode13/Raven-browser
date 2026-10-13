package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"local-music/internal/catalog"
	"local-music/internal/library"
	"local-music/internal/storage"
)

func audioTestServer(t *testing.T) (http.Handler, string, string, string) {
	t.Helper()
	root := t.TempDir()
	trackPath := filepath.Join(root, "Range_Test.mp3")
	content := "0123456789abcdefghijklmnopqrstuvwxyz"
	if err := os.WriteFile(trackPath, []byte(content), 0o600); err != nil {
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
	t.Cleanup(func() { db.Close() })
	service := catalog.New(db, scanner)
	if _, err := service.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	songs, err := service.Songs(t.Context())
	if err != nil || len(songs) != 1 {
		t.Fatalf("songs = %v, error = %v", songs, err)
	}
	assets := fstest.MapFS{"web/index.html": &fstest.MapFile{Data: []byte("ok")}}
	return New(db, assets, service, scanner), songs[0].ID, content, trackPath
}

func TestAudioEndpointSupportsRangeRequests(t *testing.T) {
	handler, id, content, _ := audioTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/songs/"+id+"/audio", nil)
	req.Header.Set("Range", "bytes=4-9")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Range"); got != "bytes 4-9/36" {
		t.Fatalf("Content-Range = %q", got)
	}
	if got, want := response.Body.String(), content[4:10]; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestAudioEndpointRejectsInvalidIDsAndMissingFiles(t *testing.T) {
	handler, id, _, path := audioTestServer(t)
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, "/api/songs/../../etc/passwd/audio", nil))
	if bad.Code == http.StatusOK || bad.Code == http.StatusPartialContent {
		t.Fatalf("invalid ID status = %d", bad.Code)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/songs/"+id+"/audio", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing file status = %d, want 404", missing.Code)
	}
}
