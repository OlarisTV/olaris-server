package ffmpeg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"
)

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
	encoderParams.Codecs = GetAVC1Tag(
		scaledWidth, scaledHeight,
		int64(encoderParams.videoBitrate),
		stream.FrameRate)

	return encoderParams, nil
}

// List of standard presets that are offered by default
var standardPresets = []string{
	"preset:480-1000k-video",
	"preset:720-5000k-video",
	"preset:1080-10000k-video"}

func GetStandardPresetVideoRepresentations(stream Stream) []StreamRepresentation {
	representations := []StreamRepresentation{}
	for _, preset := range standardPresets {
		r, _ := StreamRepresentationFromRepresentationId(stream, preset)
		representations = append(representations, r)
	}
	return representations
}

func NewVideoTranscodingSession(
	stream StreamRepresentation,
	startTime time.Duration,
	segmentStartIndex int,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	encoderParams := stream.Representation.encoderParams

	args := []string{}
	if startTime != 0 {
		args = append(args, []string{
			// -ss being before -i is important for fast seeking
			"-ss", fmt.Sprintf("%.3f", startTime.Seconds()),
		}...)
	}

	args = append(args, []string{
		"-i", buildFfmpegUrlFromFileLocator(stream.Stream.FileLocator),
		"-copyts",
		"-start_at_zero",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "libx264", "-b:v", strconv.Itoa(encoderParams.videoBitrate),
		"-preset:0", "veryfast",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%.3f)", SegmentDuration.Seconds()),
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentStartIndex),
		"-hls_time", fmt.Sprintf("%.3f", SegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
	}...)

	// Set the HLS output format options
	args = setHlsTsOptions(args, segmentStartIndex)

	if encoderParams.width != 0 || encoderParams.height != 0 {
		args = append(args, []string{
			"-filter:0", fmt.Sprintf("scale=%d:%d", encoderParams.width, encoderParams.height),
		}...)
	}
	// We serve our own manifest, so we don't really care about this.
	args = append(args, path.Join(outputDir, "generated_by_ffmpeg.m3u"))

	cmd := exec.Command("ffmpeg", args...)
	log.Infoln("ffmpeg started with", cmd.Path, cmd.Args)

	logSink := getTranscodingLogSink("ffmpeg_transcode_video")
	//io.WriteString(logSink, fmt.Sprintf("%s %s\n\n", cmd.Args, options.String()))
	cmd.Stderr = logSink

	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	//stdin, _ := cmd.StdinPipe()
	//stdin.Write(optionsSerialized)
	//stdin.Close()

	return &TranscodingSession{
		cmd:               cmd,
		Stream:            stream,
		OutputDir:         outputDir,
		SegmentStartIndex: segmentStartIndex,
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
