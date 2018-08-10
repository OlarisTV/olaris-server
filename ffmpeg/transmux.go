package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(
	streamRepresentation StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startTimestamp := segments[0].StartTimestamp
	endTimestamp := segments[len(segments)-1].EndTimestamp

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startTimestamp.Seconds()),
		"-i", streamRepresentation.Stream.MediaFileURL,
		"-copyts",
		"-to", fmt.Sprintf("%.3f", endTimestamp.Seconds()),
		"-map", fmt.Sprintf("0:%d", streamRepresentation.Stream.StreamId),
		"-c:0", "copy",
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segments[0].SegmentId),
		"-hls_time", fmt.Sprintf("%.3f", TransmuxedSegDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"))
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    streamRepresentation,
		outputDir: outputDir,
		segments:  segments,
	}, nil
}

func GetTransmuxedRepresentation(stream Stream) (StreamRepresentation, error) {
	representation := StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: "direct",
			Container:        "video/mp4",
			Codecs:           stream.Codecs,
			BitRate:          int(stream.BitRate),
			transmuxed:       true,
		},
	}

	keyframeIntervals, err := GetKeyframeIntervals(stream)
	if err != nil {
		return StreamRepresentation{}, err
	}

	if stream.StreamType == "audio" {
		representation.SegmentStartTimestamps = BuildConstantSegmentDurations(
			keyframeIntervals, TransmuxedSegDuration)
	} else if stream.StreamType == "video" {
		representation.SegmentStartTimestamps = guessTransmuxedSegmentList(keyframeIntervals)
	}

	return representation, nil
}

func guessTransmuxedSegmentList(keyframeIntervals []Interval) []SegmentList {
	segmentId := 0
	sessions := []SegmentList{}

	earliestNextCut := keyframeIntervals[0].StartTimestamp + TransmuxedSegDuration
	session := []Segment{
		{
			Interval{
				keyframeIntervals[0].StartTimestamp,
				keyframeIntervals[0].StartTimestamp},
			segmentId,
		}}
	segmentId++

	for _, keyframeInterval := range keyframeIntervals {
		if session[len(session)-1].EndTimestamp >= earliestNextCut {
			session = append(session,
				Segment{
					Interval{
						keyframeInterval.StartTimestamp,
						keyframeInterval.StartTimestamp},
					segmentId})
			segmentId++

		} else {
			session[len(session)-1].EndTimestamp = keyframeInterval.EndTimestamp
		}

		if len(session) >= segmentsPerSession {
			sessions = append(sessions, session)
			session = []Segment{
				{
					Interval{
						keyframeInterval.StartTimestamp,
						keyframeInterval.StartTimestamp},
					segmentId,
				},
			}
			segmentId++
			earliestNextCut = keyframeInterval.StartTimestamp + TransmuxedSegDuration
		}
	}
	sessions = append(sessions, session)

	return sessions
}
