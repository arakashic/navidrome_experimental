package cue

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	conf.LoadFromFile(filepath.Join("..", "..", "..", "tests", "navidrome-test.toml"))
	os.Exit(m.Run())
}

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

func TestExactCUESampleBoundaries(t *testing.T) {
	for _, suffix := range []string{"flac", "wav", "ape", "m4a", "wv"} {
		t.Run(suffix, func(t *testing.T) {
			md := metadata.New("album."+suffix, metadata.Info{
				AudioProperties: metadata.AudioProperties{Duration: 3*time.Second + time.Millisecond, SampleRate: 48000, Codec: "alac"},
			})
			sheet, err := ReadCue(strings.NewReader("FILE \"album." + suffix + "\" WAVE\n  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n  TRACK 02 AUDIO\n    INDEX 00 00:01:00\n    INDEX 01 00:01:01\n"))
			require.NoError(t, err)
			e, err := NewExtractor(&md)
			require.NoError(t, err)
			tags, err := e.Extract(sheet, "album.cue")
			require.NoError(t, err)
			require.Equal(t, []string{"0"}, tags[0]["cuestartsample"])
			require.Equal(t, []string{"48640"}, tags[0]["cueendsample"])
			require.Equal(t, []string{"48640"}, tags[1]["cuestartsample"])
			require.Equal(t, []string{"0"}, tags[1]["cueendsample"])
			require.Equal(t, int64(48000), md.WithCUERawTags(tags[1]).Int(model.TagName("cuesourcerate")))
		})
	}
}

func TestExactCUEMultiFile(t *testing.T) {
	sheet, err := ReadCue(strings.NewReader("FILE \"disc1.flac\" WAVE\n  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n  TRACK 02 AUDIO\n    INDEX 01 00:01:01\nFILE \"disc2.flac\" WAVE\n  TRACK 03 AUDIO\n    INDEX 01 00:00:00\n  TRACK 04 AUDIO\n    INDEX 01 00:01:02\n"))
	require.NoError(t, err)
	for _, tc := range []struct{ name, boundary string }{{"disc1.flac", "44688"}, {"disc2.flac", "45276"}} {
		md := metadata.New(tc.name, metadata.Info{AudioProperties: metadata.AudioProperties{Duration: 3 * time.Second, SampleRate: 44100}})
		e, err := NewExtractor(&md)
		require.NoError(t, err)
		tags, err := e.Extract(sheet, "discs.cue")
		require.NoError(t, err)
		require.Len(t, tags, 2)
		require.Equal(t, []string{tc.boundary}, tags[0]["cueendsample"])
		require.Equal(t, []string{tc.boundary}, tags[1]["cuestartsample"])
		require.Equal(t, []string{"0"}, tags[1]["cueendsample"])
	}
}

func TestCUECompatibilityFallbacks(t *testing.T) {
	for _, tc := range []struct {
		suffix, codec string
		rate, bits    int
	}{
		{"dsf", "dsd", 2822400, 1}, {"wv", "dsd", 352800, 1},
		{"m4a", "aac", 44100, 0}, {"flac", "flac", 32000, 16},
	} {
		t.Run(tc.suffix+tc.codec, func(t *testing.T) {
			md := metadata.New("album."+tc.suffix, metadata.Info{AudioProperties: metadata.AudioProperties{Duration: 3 * time.Second, SampleRate: tc.rate, BitDepth: tc.bits, Codec: tc.codec}})
			sheet, err := ReadCue(strings.NewReader("FILE \"album." + tc.suffix + "\" WAVE\n  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n  TRACK 02 AUDIO\n    INDEX 01 00:01:01\n"))
			require.NoError(t, err)
			e, err := NewExtractor(&md)
			require.NoError(t, err)
			tags, err := e.Extract(sheet, "album.cue")
			require.NoError(t, err)
			require.Len(t, tags, 2)
			require.NotContains(t, tags[0], "cuesourcerate")
		})
	}
}
