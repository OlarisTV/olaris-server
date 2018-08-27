package ffmpeg

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/olaris/olaris-server/ffmpeg/ffchunk_options"
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

	splitTimes := []int64{}
	for _, s := range segments[1:] {
		splitTimes = append(splitTimes, int64(s.StartTimestamp))
	}

	options := ffchunk_options.FFChunkOptions{
		InputFile:         stream.Stream.MediaFileURL,
		OutputDir:         outputDir,
		StreamIndex:       stream.Stream.StreamId,
		StartDts:          int64(segments[0].StartTimestamp),
		EndDts:            int64(segments[len(segments)-1].EndTimestamp),
		SegmentStartIndex: int64(segments[0].SegmentId),
		SplitDts:          splitTimes,
	}
	optionsSerialized, _ := proto.Marshal(&options)

	cmd := exec.Command("ffchunk_transmux")
	log.Println("ffmpeg started with", cmd.Args, options.String())
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	stdin, _ := cmd.StdinPipe()
	stdin.Write(optionsSerialized)
	stdin.Close()

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
