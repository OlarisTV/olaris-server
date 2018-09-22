package ffmpeg

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg/executable"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
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

	startDuration := timestampToDuration(segments[0].StartTimestamp, stream.Stream.TimeBase)
	endDuration := timestampToDuration(segments[len(segments)-1].EndTimestamp, stream.Stream.TimeBase)

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.Stream.MediaFileURL,
		"-copyts",
		"-to", fmt.Sprintf("%.3f", endDuration.Seconds()),
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "copy",
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segments[0].SegmentId),
		"-hls_time", fmt.Sprintf("%.3f", TransmuxedSegDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u")}

	cmd := exec.Command(executable.GetFFmpegExecutablePath(), args...)
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Dir = outputDir

	logSink := getTranscodingLogSink("ffmpeg")
	//io.WriteString(logSink, fmt.Sprintf("%s %s\n\n", cmd.Args, options.String()))
	cmd.Stderr = logSink
	cmd.Stdout = os.Stdout

	//stdin, _ := cmd.StdinPipe()
	//stdin.Write(optionsSerialized)
	//stdin.Close()

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

	totalInterval := Interval{
		keyframeIntervals[0].TimeBase,
		keyframeIntervals[0].StartTimestamp,
		keyframeIntervals[len(keyframeIntervals)-1].EndTimestamp,
	}
	if stream.StreamType == "audio" {
		representation.SegmentStartTimestamps = []SegmentList{
			BuildConstantSegmentDurations(
				totalInterval, TransmuxedSegDuration, 0),
		}
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
