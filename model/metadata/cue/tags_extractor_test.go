package cue

import (
	"strings"
	"testing"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedRenamedImage(t *testing.T) {
	sheet, err := ReadCue(strings.NewReader("FILE \"CDImage.wav\" WAVE\n  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n  TRACK 02 AUDIO\n    INDEX 01 04:41:56\n"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name     string
		duration time.Duration
		sidecar  string
		want     int
	}{
		{"renamed embedded album", 10 * time.Minute, "", 2},
		{"stale album cue in individual song", 4 * time.Minute, "", 0},
		{"unknown source duration", 0, "", 0},
		{"external filename remains strict", 10 * time.Minute, "album.cue", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := metadata.New("artist/album.flac", metadata.Info{
				AudioProperties: metadata.AudioProperties{Duration: tc.duration},
			})
			extractor, err := NewExtractor(&md)
			require.NoError(t, err)
			tags, err := extractor.Extract(sheet, tc.sidecar)
			require.NoError(t, err)
			require.Len(t, tags, tc.want)
			if tc.want > 0 {
				require.Equal(t, []string{model.EmbeddedCUEFile}, tags[0]["cuefile"])
			}
		})
	}
}
