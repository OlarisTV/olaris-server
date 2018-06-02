package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

func NewSubtitleSession(
	stream StreamRepresentation,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "subtitle-session-")
	if err != nil {
		return nil, err
	}

	extractSubtitlesCmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-i", stream.Stream.MediaFilePath,
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-threads", "2",
		"-f", "webvtt",
		"stream0_0.m4s")
	extractSubtitlesCmd.Stderr, _ = os.Open(os.DevNull)
	extractSubtitlesCmd.Dir = outputDir

	log.Println("ffmpeg initialized with", extractSubtitlesCmd.Args)

	return &TranscodingSession{
		cmd:            extractSubtitlesCmd,
		Stream:         stream,
		outputDir:      outputDir,
		firstSegmentId: 0,
	}, nil
}

func GetSubtitleStreamRepresentation(stream Stream) StreamRepresentation {
	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: "webvtt",
		},
		SegmentStartTimestamps: []time.Duration{0},
	}
}

func GetSubtitleStreamRepresentations(streams []Stream) []StreamRepresentation {
	subtitleRepresentations := []StreamRepresentation{}
	for _, s := range streams {
		subtitleRepresentations = append(subtitleRepresentations,
			GetSubtitleStreamRepresentation(s))
	}
	return subtitleRepresentations
}
