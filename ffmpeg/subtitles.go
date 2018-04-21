package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
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

	log.Println("ffmpeg initialized with", extractSubtitlesCmd.Args)
	log.Println("ffmpeg initialized with", segmentSubtitlesCmd.Args)
	extractSubtitlesCmd.Start()

	return &TranscodingSession{
		cmd:            segmentSubtitlesCmd,
		Stream:         stream,
		outputDir:      outputDir,
		firstSegmentId: 0,
	}, nil
}

func GetOfferedSubtitleStreams(mediaFilePath string) ([]OfferedStream, error) {
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err
	}

	offeredStreams := []OfferedStream{}

	numFullSegments := int64(container.Format.Duration() / subtitleSegmentDuration)
	segmentStartTimestamps := []time.Duration{}
	for i := int64(0); i < numFullSegments+1; i++ {
		segmentStartTimestamps = append(segmentStartTimestamps,
			time.Duration(i*int64(subtitleSegmentDuration)))
	}

	for _, probeStream := range container.Streams {
		if probeStream.CodecType != "subtitle" {
			continue
		}

		offeredStreams = append(offeredStreams, OfferedStream{
			StreamKey: StreamKey{
				MediaFilePath:    mediaFilePath,
				StreamId:         int64(probeStream.Index),
				RepresentationId: "webvtt",
			},
			TotalDuration: container.Format.Duration(),
			StreamType:    "subtitle",
			Language:      GetLanguageTag(probeStream),
			Title:         GetTitleOrHumanizedLanguage(probeStream),
			SegmentStartTimestamps: segmentStartTimestamps,
		})
	}

	return offeredStreams, nil
}
