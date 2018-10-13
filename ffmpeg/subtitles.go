package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func NewSubtitleSession(
	stream StreamRepresentation,
	segments []Segment,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "subtitle-session-")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-i", stream.Stream.MediaFileURL,
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-threads", "2",
		"-f", "webvtt",
		"stream0_0.m4s")
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Dir = outputDir

	log.Println("ffmpeg initialized with", cmd.Args, " in dir ", cmd.Dir)

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    stream,
		outputDir: outputDir,
		segments:  segments,
	}, nil
}

func GetSubtitleStreamRepresentation(stream Stream) StreamRepresentation {
	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: "webvtt",
		},
		SegmentStartTimestamps: [][]Segment{
			[]Segment{
				Segment{
					Interval: Interval{
						stream.TimeBase,
						0,
						stream.TotalDurationDts,
					},
					SegmentId: 0},
			},
		},
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
