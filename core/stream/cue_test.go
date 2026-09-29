package stream

import (
	"testing"

	"github.com/navidrome/navidrome/model"
	"github.com/stretchr/testify/assert"
)

func TestCUECacheMetadata(t *testing.T) {
	original := model.MediaFile{ID: "cue-track", CueTrack: 1, Title: "First", Artist: "Artist", Album: "Album",
		AlbumArtist: "Album Artist", TrackNumber: 1, DiscNumber: 1, Date: "2026"}
	key := (&streamJob{mf: &original}).Key()
	for i := 0; i < 20; i++ {
		assert.Equal(t, key, (&streamJob{mf: &original}).Key())
	}
	for name, change := range map[string]func(*model.MediaFile){
		"title":        func(m *model.MediaFile) { m.Title = "Second" },
		"artist":       func(m *model.MediaFile) { m.Artist = "Other" },
		"album":        func(m *model.MediaFile) { m.Album = "Other" },
		"album artist": func(m *model.MediaFile) { m.AlbumArtist = "Other" },
		"track":        func(m *model.MediaFile) { m.TrackNumber++ },
		"disc":         func(m *model.MediaFile) { m.DiscNumber++ },
		"date":         func(m *model.MediaFile) { m.Date = "2025" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := original
			change(&changed)
			assert.NotEqual(t, key, (&streamJob{mf: &changed}).Key())
			originalFile, changedFile := original, changed
			originalFile.CueTrack, changedFile.CueTrack = 0, 0
			assert.Equal(t, (&streamJob{mf: &originalFile}).Key(), (&streamJob{mf: &changedFile}).Key())
		})
	}
}

func TestCUEOutputFormat(t *testing.T) {
	for _, tc := range []struct {
		name, suffix, requested, expected string
		cueTrack                          int
	}{
		{name: "FLAC", suffix: "flac", cueTrack: 1, requested: "raw", expected: "flac"},
		{name: "WAV", suffix: "wav", cueTrack: 1, requested: "raw", expected: "wav"},
		{name: "APE", suffix: "ape", cueTrack: 1, requested: "raw", expected: "flac"},
		{name: "ALAC", suffix: "m4a", cueTrack: 1, requested: "raw", expected: "flac"},
		{name: "WavPack", suffix: "wv", cueTrack: 1, requested: "raw", expected: "flac"},
		{name: "explicit format", suffix: "wv", cueTrack: 1, requested: "wav", expected: "wav"},
		{name: "ordinary file", suffix: "m4a", requested: "raw", expected: "m4a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mf := model.MediaFile{Suffix: tc.suffix, CueTrack: tc.cueTrack}
			assert.Equal(t, tc.expected, OutputFormat(&mf, tc.requested))
		})
	}
}

func TestCUECacheSegments(t *testing.T) {
	mf := model.MediaFile{ID: "ape-track", Suffix: "ape", CueTrack: 1, CueEndSample: 44100}
	key := (&streamJob{mf: &mf, format: "flac"}).Key()
	for _, change := range []func(*model.MediaFile){
		func(m *model.MediaFile) { m.CueTrack++ },
		func(m *model.MediaFile) { m.CueStartSample++ },
		func(m *model.MediaFile) { m.CueEndSample++ },
	} {
		changed := mf
		change(&changed)
		assert.NotEqual(t, key, (&streamJob{mf: &changed, format: "flac"}).Key())
	}
	assert.NotEqual(t, key, (&streamJob{mf: &mf, format: "flac", offset: 1}).Key())
	assert.NotEqual(t, key, (&streamJob{mf: &mf, format: "wav"}).Key())
}
