package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

var AudioEncoderPresets = map[string]EncoderParams{
	"64k-audio":  {audioBitrate: 64000, Codecs: "mp4a.40.2"},
	"128k-audio": {audioBitrate: 128000, Codecs: "mp4a.40.2"},
}

// This is exactly 234 AAC frames (1024 samples each) @ 48kHz.
// TODO(Leon Handreke): Do we need to set this differently for different sampling rates?
const transcodedAudioSegmentDuration = 4992 * time.Millisecond

func NewAudioTranscodingSession(
	stream StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	// TODO(Leon Handreke): Fix the prerun
	startDuration := timestampToDuration(segments[0].StartTimestamp, stream.Stream.TimeBase)
	endDuration := timestampToDuration(segments[len(segments)-1].EndTimestamp, stream.Stream.TimeBase)
	startSegmentIndex := segments[0].SegmentId

	// With AAC, we always encode an extra segment before to avoid encoder priming on the first segment we actually want
	if startSegmentIndex > 0 {
		startDuration -= transcodedAudioSegmentDuration
		startSegmentIndex--
	}

	//runStartDuration := startDuration - transcodedAudioSegmentDuration
	//
	//if runStartDuration < 0 {
	//	runStartDuration = 0
	//}
	//if startNumber < 0 {
	//	startNumber = 0
	//}

	encoderParams := stream.Representation.encoderParams

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.Stream.MediaFileURL,
		"-to", fmt.Sprintf("%.3f", endDuration.Seconds()),
		"-copyts",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "aac", "-ac", "2", "-ab", strconv.Itoa(encoderParams.audioBitrate),
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", startSegmentIndex),
		"-hls_time", fmt.Sprintf("%.3f", transcodedAudioSegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u")}

	splitTimes := []int64{}
	for _, s := range segments[1:] {
		splitTimes = append(splitTimes, int64(s.StartTimestamp))
	}

	cmd := exec.Command("ffmpeg", args...)
	log.Println("ffmpeg started with", cmd.Args)

	logSink := getTranscodingLogSink("ffmpeg_transcode_audio")
	//io.WriteString(logSink, fmt.Sprintf("%s %s\n\n", cmd.Args, options.String()))
	cmd.Stderr = logSink

	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

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

func GetTranscodedAudioRepresentation(stream Stream, representationId string, encoderParams EncoderParams) StreamRepresentation {
	keyFrameItervals, _ := GetKeyframeIntervals(stream)
	totalInterval := Interval{
		keyFrameItervals[0].TimeBase,
		keyFrameItervals[0].StartTimestamp,
		keyFrameItervals[len(keyFrameItervals)-1].EndTimestamp,
	}

	segmentStartTimestamps := buildAudioSegmentDurations(
		totalInterval, transcodedAudioSegmentDuration)

	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.audioBitrate,
			Container:        "audio/mp4",
			Codecs:           encoderParams.Codecs,
			transcoded:       true,
		},
		SegmentStartTimestamps: segmentStartTimestamps,
	}
}

func buildAudioSegmentDurations(interval Interval, segmentDuration time.Duration) []SegmentList {
	sessions := []SegmentList{}

	session := SegmentList{}
	currentTimestamp := interval.StartTimestamp
	segmentId := 0

	segmentDurationDts := DtsTimestamp(segmentDuration.Seconds() * float64(interval.TimeBase))

	for currentTimestamp < interval.EndTimestamp {
		if len(session) >= segmentsPerSession {
			sessions = append(sessions, session)
			session = SegmentList{}
		}

		session = append(session, Segment{
			Interval{
				interval.TimeBase,
				currentTimestamp,
				currentTimestamp + segmentDurationDts,
			},
			segmentId,
		})

		segmentId++
		currentTimestamp += segmentDurationDts
	}

	// Append the last segment to the end of the interval
	session = append(session, Segment{
		Interval{
			interval.TimeBase,
			currentTimestamp,
			interval.EndTimestamp,
		},
		segmentId,
	})
	sessions = append(sessions, session)

	return sessions

}
