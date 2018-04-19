package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

const subtitleSegmentDuration = 60 * time.Second

func NewSubtitleSession(
	stream OfferedStream,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "subtitle-session-")
	if err != nil {
		return nil, err
	}

	extractSubtitlesCmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-i", stream.MediaFilePath,
		"-map", fmt.Sprintf("0:%d", stream.StreamId),
		"-threads", "2",
		"-f", "webvtt",
		"-")
	extractSubtitlesCmd.Stderr, _ = os.Open(os.DevNull)
	segmentSubtitlesCmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-i", "pipe:0",
		"-map", "0:0",
		"-c:0", "copy",
		"-f", "segment",
		"-segment_format", "webvtt",
		"-segment_time", "10",
		// These are not actually MP4 fragments, but we hardcode the filenames elsewhere...
		"stream0_%d.m4s")
	segmentSubtitlesCmd.Stderr, _ = os.Open(os.DevNull)
	segmentSubtitlesCmd.Dir = outputDir

	segmentSubtitlesCmd.Stdin, _ = extractSubtitlesCmd.StdoutPipe()

	extractSubtitlesCmd.Start()

	return &TranscodingSession{
		cmd:            segmentSubtitlesCmd,
		Stream:         stream,
		outputDir:      outputDir,
		firstSegmentId: 0,
	}, nil
}

func GetOfferedSubtitleStreams(container ProbeContainer) []OfferedStream {
	offeredStreams := []OfferedStream{}

	numFullSegments := container.Format.Duration() / subtitleSegmentDuration
	segmentDurations := []time.Duration{}
	// We want one more segment to cover the end. For the moment we don't
	// care that it's a bit longer in the manifest, the client will play till EOF
	for i := 0; i < int(numFullSegments)+1; i++ {
		segmentDurations = append(segmentDurations, subtitleSegmentDuration)
	}

	for _, probeStream := range container.Streams {
		if probeStream.CodecType != "subtitle" {
			continue
		}

		offeredStreams = append(offeredStreams, OfferedStream{
			StreamKey: StreamKey{
				StreamId:         int64(probeStream.Index),
				RepresentationId: "webvtt",
			},
			TotalDuration: container.Format.Duration(),
			StreamType:    "subtitle",
			Language:      probeStream.Tags["language"],
			// TODO(Leon Handreke): Pick up the "title" field or render a user-presentable language string.
			Title:            probeStream.Tags["language"],
			SegmentDurations: segmentDurations,
		})
	}

	return offeredStreams
}
