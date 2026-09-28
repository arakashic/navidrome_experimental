package cue

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
)

type TagsExtractor struct {
	md *metadata.Metadata
}

func NewExtractor(md *metadata.Metadata) (*TagsExtractor, error) {
	return &TagsExtractor{
		md: md,
	}, nil
}

func makeDuration(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 32)
}

func (e *TagsExtractor) Extract(cueSheet *Cuesheet, cueFile string) ([]model.RawTags, error) {
	result := make([]model.RawTags, 0)
	cueSource := cueFile
	if cueSource == "" {
		cueSource = model.EmbeddedCUEFile
	}

	for _, file := range cueSheet.File {
		if file.FileName != path.Base(e.md.FilePath()) {
			if cueFile != "" || len(cueSheet.File) != 1 || !embeddedTracksFit(file, e.md.AudioProperties().Duration) {
				continue
			}
		}
		totalTracks := len(file.Tracks)
		lastTrack := totalTracks - 1
		for i, t := range file.Tracks {
			tags := make(model.RawTags)

			addTagValue(model.TagCUEFile, tags, cueSource)
			addTagValue(model.TagTitle, tags, t.Title, fmt.Sprintf("Track %02d", t.TrackNumber))
			addTagValue(model.TagCUESubTrack, tags, strconv.Itoa(i))
			addTagValue(model.TagTrackArtist, tags, t.Performer, cueSheet.Performer,
				e.md.String(model.TagTrackArtist), e.md.String(model.TagAlbumArtist))
			addTagValue(model.TagAlbum, tags, cueSheet.Title)
			addTagValue(model.TagGenre, tags, cueSheet.Rem.Genre())
			if t.Rem.TotalDiscs() > 0 {
				if t.Rem.DiscNumber() > 0 {
					addTagValue(model.TagDiscNumber, tags, strconv.FormatUint(uint64(t.Rem.DiscNumber()), 10))
				}
				addTagValue(model.TagTotalDiscs, tags, strconv.FormatUint(uint64(t.Rem.TotalDiscs()), 10))
			}
			addTagValue(model.TagAlbumArtist, tags, cueSheet.Performer)
			addTagValue(model.TagComment, tags, t.Rem.Comment(), cueSheet.Rem.Comment())
			addTagValue(model.TagTrackNumber, tags, strconv.FormatUint(uint64(t.TrackNumber), 10))
			addTagValue(model.TagTotalTracks, tags, strconv.FormatUint(uint64(totalTracks), 10))
			addTagValue(model.TagReleaseDate, tags, cueSheet.Rem.Date())
			addTagValue(model.TagReplayGainAlbumGain, tags, cueSheet.Rem.AlbumGain())
			addTagValue(model.TagReplayGainAlbumPeak, tags, cueSheet.Rem.AlbumPeak())
			addTagValue(model.TagReplayGainTrackGain, tags, t.Rem.TrackGain())
			addTagValue(model.TagReplayGainTrackPeak, tags, t.Rem.TrackPeak())
			addTagValue(model.TagCUETrackOffset, tags, makeDuration(t.GetStartOffset()))
			addTagValue(model.TagISRC, tags, t.ISRC)

			var length time.Duration
			if i < lastTrack {
				nextTrack := file.Tracks[i+1]
				length = nextTrack.GetStartOffset() - t.GetStartOffset()
			} else {
				fileDuration := time.Millisecond * time.Duration(e.md.Length()*1000.0)
				length = fileDuration - t.GetStartOffset()
			}
			if length < 0 {
				length = 0
			}
			addTagValue(model.TagCUETrackDuration, tags, makeDuration(length))

			// Fallback to main metadata for missing tags
			addTagValue(model.TagAlbum, tags, e.md.String(model.TagAlbum))
			addTagValue(model.TagAlbumArtist, tags, e.md.String(model.TagAlbumArtist))
			addTagValue(model.TagReleaseDate, tags, e.md.String(model.TagReleaseDate))
			for _, genre := range e.md.Strings(model.TagGenre) {
				addTagValue(model.TagGenre, tags, genre)
			}
			addTagValue(model.TagMusicBrainzAlbumID, tags, e.md.String(model.TagMusicBrainzAlbumID))
			addTagValue(model.TagMusicBrainzArtistID, tags, e.md.String(model.TagMusicBrainzArtistID))
			addTagValue(model.TagMusicBrainzRecordingID, tags, e.md.String(model.TagMusicBrainzRecordingID))
			addTagValue(model.TagMusicBrainzTrackID, tags, e.md.String(model.TagMusicBrainzTrackID))
			for _, value := range e.md.CueTags()[fmt.Sprintf("cue_track%02d_musicbrainz_trackid", t.TrackNumber)] {
				addTagValue(model.TagMusicBrainzTrackID, tags, value)
			}
			disc, discs := e.md.NumAndTotal(model.TagDiscNumber)
			if disc > 0 && discs > 0 {
				addTagValue(model.TagDiscNumber, tags, strconv.FormatUint(uint64(disc), 10))
				addTagValue(model.TagTotalDiscs, tags, strconv.FormatUint(uint64(discs), 10))
			}

			result = append(result, tags)
		}
	}

	return result, nil
}

func embeddedTracksFit(file File, duration time.Duration) bool {
	if duration <= 0 || len(file.Tracks) == 0 {
		return false
	}
	previous := time.Duration(-1)
	for _, track := range file.Tracks {
		start := track.GetStartOffset()
		if start <= previous || start >= duration {
			return false
		}
		for _, index := range track.Index {
			if index.Frame.Duration() >= duration {
				return false
			}
		}
		previous = start
	}
	return true
}

func addTagValue(name model.TagName, tags model.RawTags, values ...string) {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			tags[name.String()] = append(tags[name.String()], v)
			return
		}
	}
}
