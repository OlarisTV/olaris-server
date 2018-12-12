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

// Doesn't have to be the same as audio, but why not.
const transcodedVideoSegmentDuration = 4992 * time.Millisecond

func GetVideoEncoderPreset(stream Stream, name string) (EncoderParams, error) {
	encoderParams, exists := map[string]EncoderParams{
		"480-1000k-video": {
			height: 480, width: -2,
			videoBitrate: 1000000},
		"720-5000k-video": {
			height: 720, width: -2,
			videoBitrate: 5000000},
		"1080-10000k-video": {
			height: 1080, width: -2,
			videoBitrate: 10000000},
	}[name]

	if !exists {
		return EncoderParams{}, fmt.Errorf("no preset \"%s\"", name)
	}
	scaledWidth, scaledHeight := scalePreserveAspectRatio(
		stream.Width, stream.Height,
		-2, encoderParams.height)
	encoderParams.Codecs = GetAVC1Tag(uint64(encoderParams.videoBitrate), scaledWidth, scaledHeight)

	return encoderParams, nil
}

func NewVideoTranscodingSession(
	stream StreamRepresentation,
	startTime time.Duration,
	segmentStartIndex int,
	outputDirBase string,
	feedbackURL string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	encoderParams := stream.Representation.encoderParams

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startTime.Seconds()),
		"-i", stream.Stream.MediaFileURL,
		"-copyts",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "libx264", "-b:v", strconv.Itoa(encoderParams.videoBitrate),
		"-preset:0", "veryfast",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%.3f)", transcodedVideoSegmentDuration.Seconds()),
		"-filter:0", fmt.Sprintf("scale=%d:%d", encoderParams.width, encoderParams.height),
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentStartIndex),
		"-hls_time", fmt.Sprintf("%.3f", transcodedVideoSegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		"-olaris_feedback_url", feedbackURL,
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"),
	}

	cmd := exec.Command(executable.GetFFmpegExecutablePath(), args...)
	log.Println("ffmpeg started with", cmd.Path, cmd.Args)

	logSink := getTranscodingLogSink("ffmpeg_transcode_video")
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
	}, nil
}

func GetTranscodedVideoRepresentation(
	stream Stream,
	representationId string,
	encoderParams EncoderParams) StreamRepresentation {

	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.videoBitrate,
			Height:           encoderParams.height,
			Width:            encoderParams.width,
			Container:        "video/mp4",
			Codecs:           encoderParams.Codecs,
			Transcoded:       true,
			encoderParams:    encoderParams,
		},
	}
}

func buildVideoSegmentDurations(keyframeIntervals []Interval, segmentDuration time.Duration) [][]Segment {
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
	sessions := [][]Segment{}
	for _, sessionInterval := range sessionIntervals {
		session := BuildConstantSegmentDurations(sessionInterval, segmentDuration, segmentIndex)
		sessions = append(sessions, session)
		segmentIndex += len(session)
	}
	return sessions
}
