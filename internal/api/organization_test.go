package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrganizationEndpointsCreateAndAssign(t *testing.T) {
	handler, songID, _, _ := audioTestServer(t)
	playlistResponse := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"Favorites"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(playlistResponse, request)
	if playlistResponse.Code != http.StatusCreated {
		t.Fatalf("create playlist status = %d: %s", playlistResponse.Code, playlistResponse.Body.String())
	}
	var playlist struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(playlistResponse.Body.Bytes(), &playlist); err != nil {
		t.Fatal(err)
	}
	assign := httptest.NewRecorder()
	handler.ServeHTTP(assign, httptest.NewRequest(http.MethodPut, "/api/playlists/"+playlist.ID+"/songs/"+songID, nil))
	if assign.Code != http.StatusNoContent {
		t.Fatalf("assign playlist status = %d: %s", assign.Code, assign.Body.String())
	}
	tagResponse := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/tags", strings.NewReader(`{"name":"Focus"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(tagResponse, request)
	if tagResponse.Code != http.StatusCreated {
		t.Fatalf("create tag status = %d: %s", tagResponse.Code, tagResponse.Body.String())
	}
	var tag struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(tagResponse.Body.Bytes(), &tag); err != nil {
		t.Fatal(err)
	}
	assign = httptest.NewRecorder()
	handler.ServeHTTP(assign, httptest.NewRequest(http.MethodPut, "/api/songs/"+songID+"/tags/"+tag.ID, nil))
	if assign.Code != http.StatusNoContent {
		t.Fatalf("assign tag status = %d: %s", assign.Code, assign.Body.String())
	}
	organization := httptest.NewRecorder()
	handler.ServeHTTP(organization, httptest.NewRequest(http.MethodGet, "/api/songs/"+songID+"/organization", nil))
	if organization.Code != http.StatusOK {
		t.Fatalf("organization status = %d: %s", organization.Code, organization.Body.String())
	}
	var result struct {
		Playlists []struct {
			ID string `json:"id"`
		} `json:"playlists"`
		Tags []struct {
			ID string `json:"id"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(organization.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Playlists) != 1 || result.Playlists[0].ID != playlist.ID || len(result.Tags) != 1 || result.Tags[0].ID != tag.ID {
		t.Fatalf("organization response = %+v", result)
	}
	catalogResponse := httptest.NewRecorder()
	handler.ServeHTTP(catalogResponse, httptest.NewRequest(http.MethodGet, "/api/songs", nil))
	if catalogResponse.Code != http.StatusOK {
		t.Fatalf("catalog status = %d: %s", catalogResponse.Code, catalogResponse.Body.String())
	}
	var catalogSongs []struct {
		ID   string   `json:"id"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(catalogResponse.Body.Bytes(), &catalogSongs); err != nil {
		t.Fatal(err)
	}
	if len(catalogSongs) != 1 || catalogSongs[0].ID != songID || len(catalogSongs[0].Tags) != 1 || catalogSongs[0].Tags[0] != "Focus" {
		t.Fatalf("catalog tags response = %+v", catalogSongs)
	}
}

func TestOrganizationEndpointsRejectInvalidInputAndDuplicateNames(t *testing.T) {
	handler, _, _, _ := audioTestServer(t)
	for _, name := range []string{"Mix", "mix"} {
		request := httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"`+name+`"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		want := http.StatusCreated
		if name == "mix" {
			want = http.StatusConflict
		}
		if response.Code != want {
			t.Fatalf("create %q status = %d, want %d: %s", name, response.Code, want, response.Body.String())
		}
	}
	invalid := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/tags", strings.NewReader(`{"name":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(invalid, request)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid tag status = %d, want 400", invalid.Code)
	}
	unknown := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/playlists", strings.NewReader(`{"name":"ok","unexpected":true}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(unknown, request)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", unknown.Code)
	}
}
