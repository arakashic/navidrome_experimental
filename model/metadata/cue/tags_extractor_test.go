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

type replayGainFileInfo struct{ os.FileInfo }

func (replayGainFileInfo) BirthTime() time.Time { return time.Time{} }

func TestCUEAlbumReplayGainFallback(t *testing.T) {
	for _, source := range []string{"album.cue", ""} {
		for _, tc := range []struct {
			name               string
			fileGain, filePeak string
			cueGain, cuePeak   string
			wantGain, wantPeak *float64
		}{
			{"file album values", "-3.20 dB", "0.9", "", "", new(-3.2), new(0.9)},
			{"explicit cue values", "-3.20 dB", "0.9", "+2.00 dB", "0.8", new(2.0), new(0.8)},
			{"explicit zero gain", "-3.20 dB", "0.9", "0.00 dB", "0.8", new(0.0), new(0.8)},
			{"cue gain only", "-3.20 dB", "0.9", "+2.00 dB", "", new(2.0), new(0.9)},
			{"cue peak only", "-3.20 dB", "0.9", "", "0.8", new(-3.2), new(0.8)},
			{"no album values", "", "", "", "", nil, nil},
			{"file zero gain only", "0.00 dB", "", "", "", new(0.0), nil},
			{"file peak only", "", "0.9", "", "", nil, new(0.9)},
			{"blank cue values", "-3.20 dB", "0.9", "   ", "   ", new(-3.2), new(0.9)},
		} {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				fileInfo, err := os.Stat(t.TempDir())
				require.NoError(t, err)
				md := metadata.New("album.flac", metadata.Info{
					FileInfo: replayGainFileInfo{fileInfo},
					Tags: model.RawTags{
						"replaygain_album_gain": {tc.fileGain},
						"replaygain_album_peak": {tc.filePeak},
						"replaygain_track_gain": {"-9.00 dB"},
						"replaygain_track_peak": {"0.99"},
					},
					AudioProperties: metadata.AudioProperties{Duration: 3 * time.Second, SampleRate: 44100},
				})
				text := "FILE \"album.flac\" WAVE\n  TRACK 01 AUDIO\n    TITLE \"First\"\n    REM REPLAYGAIN_TRACK_GAIN -1.50 dB\n    REM REPLAYGAIN_TRACK_PEAK 0.75\n    INDEX 01 00:00:00\n  TRACK 02 AUDIO\n    TITLE \"Second\"\n    INDEX 01 00:01:01\n"
				if tc.cueGain != "" {
					text = "REM REPLAYGAIN_ALBUM_GAIN \"" + tc.cueGain + "\"\n" + text
				}
				if tc.cuePeak != "" {
					text = "REM REPLAYGAIN_ALBUM_PEAK \"" + tc.cuePeak + "\"\n" + text
				}
				sheet, err := ReadCue(strings.NewReader(text))
				require.NoError(t, err)
				extractor, err := NewExtractor(&md)
				require.NoError(t, err)
				tags, err := extractor.Extract(sheet, source)
				require.NoError(t, err)
				require.Len(t, tags, 2)
				for _, raw := range tags {
					mf := md.WithCUERawTags(raw).ToMediaFile(1, "folder")
					require.Equal(t, tc.wantGain, mf.RGAlbumGain)
					require.Equal(t, tc.wantPeak, mf.RGAlbumPeak)
				}
				first := md.WithCUERawTags(tags[0]).ToMediaFile(1, "folder")
				second := md.WithCUERawTags(tags[1]).ToMediaFile(1, "folder")
				require.Equal(t, new(-1.5), first.RGTrackGain)
				require.Equal(t, new(0.75), first.RGTrackPeak)
				require.Nil(t, second.RGTrackGain)
				require.Nil(t, second.RGTrackPeak)
				require.Equal(t, "First", first.Title)
				require.Equal(t, "Second", second.Title)
				require.Equal(t, int64(0), first.CueStartSample)
				require.Equal(t, int64(44688), first.CueEndSample)
				require.Equal(t, int64(44688), second.CueStartSample)
				require.Equal(t, int64(0), second.CueEndSample)
				require.Equal(t, 44100, first.CueSourceRate)
				if tc.wantGain == nil {
					require.NotContains(t, tags[0], model.TagReplayGainAlbumGain.String())
				}
				if tc.wantPeak == nil {
					require.NotContains(t, tags[0], model.TagReplayGainAlbumPeak.String())
				}
			})
		}
	}
}
