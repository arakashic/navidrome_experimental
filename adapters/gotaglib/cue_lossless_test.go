package gotaglib

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/model/metadata/cue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("additional lossless CUE sources", func() {
	DescribeTable("extracts source properties and expands tracks",
		func(suffix, encoder, expectedCodec string) {
			dir := GinkgoT().TempDir()
			name := "album." + suffix
			path := filepath.Join(dir, name)
			output, err := exec.CommandContext(context.Background(), "ffmpeg", "-v", "error", "-f", "lavfi", "-i",
				"aevalsrc=0.3*sin(2*PI*997*t)|0.2*sin(2*PI*1231*t):s=48000:d=3",
				"-c:a", encoder, path).CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))

			e := extractor{fs: os.DirFS(dir)}
			info, err := e.extractMetadata(name)
			Expect(err).NotTo(HaveOccurred())
			stat, err := os.Stat(path)
			Expect(err).NotTo(HaveOccurred())
			info.FileInfo = testFileInfo{stat}
			md := metadata.New(name, *info)
			Expect(strings.ToLower(md.AudioProperties().Codec)).To(Equal(expectedCodec))
			Expect(md.AudioProperties().SampleRate).To(Equal(48000))

			sheet, err := cue.ReadCue(strings.NewReader(fmt.Sprintf("FILE %q WAVE\n"+
				"  TRACK 01 AUDIO\n    INDEX 01 00:00:00\n"+
				"  TRACK 02 AUDIO\n    INDEX 01 00:01:00\n", name)))
			Expect(err).NotTo(HaveOccurred())
			tracks, err := md.CUETracks(sheet, 1, "folder")
			Expect(err).NotTo(HaveOccurred())
			Expect(tracks).To(HaveLen(2))
			Expect(tracks[0].CueStartSample).To(BeZero())
			Expect(tracks[0].CueEndSample).To(Equal(int64(48000)))
			Expect(tracks[1].CueStartSample).To(Equal(int64(48000)))
		},
		Entry("ALAC", "m4a", "alac", "alac"),
		Entry("WavPack suffix fallback", "wv", "wavpack", ""),
	)
})
