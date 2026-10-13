package library

import "testing"

func TestParseFilenameUsesLastUnderscore(t *testing.T) {
	title, artist, format := ParseFilename("album/live_song_band_member.MP3")
	if title != "live_song_band" || artist != "member" || format != "mp3" {
		t.Fatalf("got (%q, %q, %q)", title, artist, format)
	}
}

func TestParseFilenameFallbacks(t *testing.T) {
	for _, test := range []struct{ name, title, artist string }{
		{"mystery.flac", "mystery", "Desconocido"},
		{"_artist.mp3", "_artist", "artist"},
		{"title_.wav", "title", "Desconocido"},
	} {
		title, artist, _ := ParseFilename(test.name)
		if title != test.title || artist != test.artist {
			t.Errorf("ParseFilename(%q) = (%q, %q), want (%q, %q)", test.name, title, artist, test.title, test.artist)
		}
	}
}
