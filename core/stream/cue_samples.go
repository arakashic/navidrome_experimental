package stream

import (
	"strconv"

	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
)

func cueSampleSegment(mf *model.MediaFile) *ffmpeg.AudioSegment {
	if !mf.IsCueSubTrack() || mf.CueSourceRate <= 0 {
		return nil
	}
	tags := map[string]string{
		"title": mf.Title, "artist": mf.Artist, "album": mf.Album, "album_artist": mf.AlbumArtist,
		"track": strconv.Itoa(mf.TrackNumber), "disc": strconv.Itoa(mf.DiscNumber), "date": mf.Date,
		"genre": mf.Genre, "comment": mf.Comment,
	}
	for name, value := range map[string]*float64{
		"replaygain_album_gain": mf.RGAlbumGain, "replaygain_album_peak": mf.RGAlbumPeak,
		"replaygain_track_gain": mf.RGTrackGain, "replaygain_track_peak": mf.RGTrackPeak,
	} {
		if value != nil {
			tags[name] = strconv.FormatFloat(*value, 'f', -1, 64)
		}
	}
	return &ffmpeg.AudioSegment{
		StartSample: mf.CueStartSample, EndSample: mf.CueEndSample, SourceRate: mf.CueSourceRate,
		Tags: tags,
	}
}
