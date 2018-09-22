package ffmpeg

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg/executable"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

var VideoEncoderPresets = map[string]EncoderParams{
	"480-1000k-video":   {height: 480, width: -2, videoBitrate: 1000000, Codecs: "avc1.64001e"},
	"720-5000k-video":   {height: 720, width: -2, videoBitrate: 5000000, Codecs: "avc1.64001f"},
	"1080-10000k-video": {height: 1080, width: -2, videoBitrate: 10000000, Codecs: "avc1.640028"},
}

// Doesn't have to be the same as audio, but why not.
const transcodedVideoSegmentDuration = 4992 * time.Millisecond

func NewVideoTranscodingSession(
	stream StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startDuration := timestampToDuration(segments[0].StartTimestamp, stream.Stream.TimeBase)
	endDuration := timestampToDuration(segments[len(segments)-1].EndTimestamp, stream.Stream.TimeBase)
	encoderParams := stream.Representation.encoderParams

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.Stream.MediaFileURL,
		"-to", fmt.Sprintf("%.3f", endDuration.Seconds()),
		"-copyts",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "libx264", "-b:v", strconv.Itoa(encoderParams.videoBitrate),
		"-preset:0", "veryfast",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%.3f)", transcodedVideoSegmentDuration.Seconds()),
		"-filter:0", fmt.Sprintf("scale=%d:%d", encoderParams.width, encoderParams.height),
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segments[0].SegmentId),
		"-hls_time", fmt.Sprintf("%.3f", transcodedVideoSegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"),
	}

	cmd := exec.Command(executable.GetFFmpegExecutablePath(), args...)
	log.Println("ffmpeg started with", cmd.Path, cmd.Args)

	logSink := getTranscodingLogSink("ffchunk_transcode_video")
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

func GetTranscodedVideoRepresentation(
	stream Stream,
	representationId string,
	encoderParams EncoderParams) StreamRepresentation {

	keyFrameItervals, _ := GetKeyframeIntervals(stream)

	segmentStartTimestamps := buildVideoSegmentDurations(
		keyFrameItervals, transcodedVideoSegmentDuration)

	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.videoBitrate,
			Container:        "video/mp4",
			Codecs:           encoderParams.Codecs,
			transcoded:       true,
		},
		SegmentStartTimestamps: segmentStartTimestamps,
	}
}

func buildVideoSegmentDurations(keyframeIntervals []Interval, segmentDuration time.Duration) []SegmentList {
	timeBase := keyframeIntervals[0].TimeBase
	sessionIntervals := []Interval{
		Interval{
			timeBase,
			keyframeIntervals[0].StartTimestamp,
			keyframeIntervals[0].StartTimestamp,
		},
	}

	for _, keyframeInterval := range keyframeIntervals {
		sessionIntervals[len(sessionIntervals)-1].EndTimestamp = keyframeInterval.EndTimestamp
		sessionDuration := sessionIntervals[len(sessionIntervals)-1].Duration()
		if sessionDuration >= (segmentsPerSession * segmentDuration) {
			// TODO(Leon Handreke): We may end up with a zero-length session here in rare cases.
			sessionIntervals = append(sessionIntervals, Interval{
				timeBase,
				keyframeInterval.EndTimestamp,
				keyframeInterval.EndTimestamp,
			})
		}
	}

	segmentIndex := 0
	sessions := []SegmentList{}
	for _, sessionInterval := range sessionIntervals {
		session := BuildConstantSegmentDurations(sessionInterval, segmentDuration, segmentIndex)
		sessions = append(sessions, session)
		segmentIndex += len(session)
	}
	return sessions
}
