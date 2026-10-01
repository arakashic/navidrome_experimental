package stream

import (
	"testing"

	"github.com/navidrome/navidrome/model"
	"github.com/stretchr/testify/require"
)

func TestExactCUECacheKeys(t *testing.T) {
	mf := model.MediaFile{ID: "cue", CUEFile: "album.cue", SubTrack: 0, CueSourceRate: 44100, CueEndSample: 44688, Title: "First"}
	key := (&streamJob{mf: &mf, format: "flac"}).Key()
	for _, change := range []func(*model.MediaFile){
		func(m *model.MediaFile) { m.CueStartSample++ },
		func(m *model.MediaFile) { m.CueEndSample++ },
		func(m *model.MediaFile) { m.Title = "Retagged" },
	} {
		changed := mf
		change(&changed)
		require.NotEqual(t, key, (&streamJob{mf: &changed, format: "flac"}).Key())
	}
	mf.CueSourceRate = 0
	require.Nil(t, cueSampleSegment(&mf))
	require.NotEqual(t, key, (&streamJob{mf: &mf, format: "flac"}).Key())
}
