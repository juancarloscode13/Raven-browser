package catalog

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"local-music/internal/library"
	"local-music/internal/storage"
)

func organizationFixture(t *testing.T) (*Service, context.Context, []string) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "music.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	ids := []string{"song-one", "song-two", "song-three"}
	for _, id := range ids {
		if _, err := db.ExecContext(ctx, `INSERT INTO songs(id,path,title,artist,format) VALUES(?,?,?,?,?)`, id, id+".mp3", "Song", "Artist", "mp3"); err != nil {
			t.Fatal(err)
		}
	}
	return New(db, &library.Scanner{}), ctx, ids
}

func TestPlaylistMembershipAndOrderingAreIndependent(t *testing.T) {
	service, ctx, songs := organizationFixture(t)
	first, err := service.CreatePlaylist(ctx, "Road trip")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreatePlaylist(ctx, "Quiet")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range songs {
		if err := service.SetPlaylistMembership(ctx, first.ID, id, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.SetPlaylistMembership(ctx, second.ID, songs[1], true); err != nil {
		t.Fatal(err)
	}
	if err := service.ReorderPlaylist(ctx, first.ID, []string{songs[2], songs[0], songs[1]}); err != nil {
		t.Fatal(err)
	}
	ordered, err := service.PlaylistSongs(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].ID != songs[2] || ordered[1].ID != songs[0] || ordered[2].ID != songs[1] {
		t.Fatalf("first playlist order = %+v", ordered)
	}
	independent, err := service.PlaylistSongs(ctx, second.ID)
	if err != nil || len(independent) != 1 || independent[0].ID != songs[1] {
		t.Fatalf("second playlist = %+v, %v", independent, err)
	}
	if err := service.ReorderPlaylist(ctx, first.ID, []string{songs[0], songs[0], songs[1]}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate order error = %v", err)
	}
	ordered, err = service.PlaylistSongs(ctx, first.ID)
	if err != nil || ordered[0].ID != songs[2] {
		t.Fatalf("invalid reorder changed saved order: %+v, %v", ordered, err)
	}
}

func TestTagAssignmentAndPlaylistCRUD(t *testing.T) {
	service, ctx, songs := organizationFixture(t)
	playlist, err := service.CreatePlaylist(ctx, "Favorites")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePlaylist(ctx, " favorites "); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate playlist error = %v", err)
	}
	if err := service.RenamePlaylist(ctx, playlist.ID, "  New name  "); err != nil {
		t.Fatal(err)
	}
	tag, err := service.CreateTag(ctx, "Workout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTag(ctx, "workout"); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate tag error = %v", err)
	}
	if err := service.SetTagMembership(ctx, songs[0], tag.ID, true); err != nil {
		t.Fatal(err)
	}
	catalogSongs, err := service.Songs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var tagged *library.Song
	for i := range catalogSongs {
		if catalogSongs[i].ID == songs[0] {
			tagged = &catalogSongs[i]
			break
		}
	}
	if tagged == nil || len(tagged.Tags) != 1 || tagged.Tags[0] != "Workout" {
		t.Fatalf("catalog tags = %+v", catalogSongs)
	}
	if err := service.SetPlaylistMembership(ctx, playlist.ID, songs[0], true); err != nil {
		t.Fatal(err)
	}
	organization, err := service.Organization(ctx, songs[0])
	if err != nil || len(organization.Tags) != 1 || organization.Tags[0].ID != tag.ID || len(organization.Playlists) != 1 || organization.Playlists[0].ID != playlist.ID {
		t.Fatalf("organization = %+v, %v", organization, err)
	}
	if err := service.DeletePlaylist(ctx, playlist.ID); err != nil {
		t.Fatal(err)
	}
	organization, err = service.Organization(ctx, songs[0])
	if err != nil || len(organization.Playlists) != 0 || len(organization.Tags) != 1 {
		t.Fatalf("after deleting playlist = %+v, %v", organization, err)
	}
	if err := service.DeleteTag(ctx, tag.ID); err != nil {
		t.Fatal(err)
	}
	organization, err = service.Organization(ctx, songs[0])
	if err != nil || len(organization.Tags) != 0 {
		t.Fatalf("after deleting tag = %+v, %v", organization, err)
	}
}

func TestNamesAreValidated(t *testing.T) {
	service, ctx, _ := organizationFixture(t)
	if _, err := service.CreatePlaylist(ctx, "   "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty name error = %v", err)
	}
	if _, err := service.CreateTag(ctx, string(make([]rune, 61))); !errors.Is(err, ErrInvalid) {
		t.Fatalf("long name error = %v", err)
	}
}
