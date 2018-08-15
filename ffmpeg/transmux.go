package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(
	stream StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	splitTimes := []string{}
	for _, s := range segments[1:] {
		splitTimes = append(splitTimes, fmt.Sprintf("%d", s.StartTimestamp))
	}

	args := []string{
		stream.Stream.MediaFileURL,
		outputDir,
		fmt.Sprintf("%d", stream.Stream.StreamId),
		fmt.Sprintf("%d", segments[0].StartTimestamp),
		fmt.Sprintf("%d", segments[len(segments)-1].EndTimestamp),
		fmt.Sprintf("%d", segments[0].SegmentId),
	}
	args = append(args, splitTimes...)

	cmd := exec.Command("ffchunk", args...)
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    stream,
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
	//fmt.Println(keyframeIntervals)
	segmentId := 0
	sessions := []SegmentList{}
	timeBase := keyframeIntervals[0].TimeBase
	segDurationTs := DtsTimestamp(TransmuxedSegDuration.Seconds() * float64(timeBase))

	earliestNextCut := keyframeIntervals[0].StartTimestamp + segDurationTs
	session := []Segment{
		{
			Interval{
				timeBase,
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
						timeBase,
						keyframeInterval.StartTimestamp,
						keyframeInterval.EndTimestamp},
					segmentId})
			segmentId++
			earliestNextCut += segDurationTs
		} else {
			session[len(session)-1].EndTimestamp = keyframeInterval.EndTimestamp
		}

		if len(session) >= segmentsPerSession {
			sessions = append(sessions, session)
			session = []Segment{
				{
					Interval{
						timeBase,
						keyframeInterval.EndTimestamp,
						keyframeInterval.EndTimestamp},
					segmentId,
				},
			}
			segmentId++
			earliestNextCut = keyframeInterval.EndTimestamp + segDurationTs
		}
	}
	sessions = append(sessions, session)

	//fmt.Println(sessions)
	return sessions
}
