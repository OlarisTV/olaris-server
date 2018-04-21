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
	"64k-audio":  {audioBitrate: 64000},
	"128k-audio": {audioBitrate: 128000},
}

// This is exactly 234 AAC frames (1024 samples each) @ 48kHz.
// TODO(Leon Handreke): Do we need to set this differently for different sampling rates?
const transcodedAudioSegmentDuration = 4992 * time.Millisecond

func NewAudioTranscodingSession(
	stream OfferedStream,
	outputDirBase string,
	segmentOffset int64,
	transcodingParams EncoderParams) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startDuration := time.Duration(int64(transcodedAudioSegmentDuration) * segmentOffset)

	// With AAC, we always encode an extra segment before to avoid encoder priming on the first segment we actually want
	runDuration := segmentsPerSession*transcodedAudioSegmentDuration + transcodedAudioSegmentDuration
	runStartDuration := startDuration - transcodedAudioSegmentDuration
	startNumber := segmentOffset - 1

	if runStartDuration < 0 {
		runStartDuration = 0
	}
	if startNumber < 0 {
		startNumber = 0
	}

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.MediaFilePath,
		"-to", fmt.Sprintf("%.3f", (startDuration + runDuration).Seconds()),
		"-copyts",
		"-map", fmt.Sprintf("0:%d", stream.StreamId),
		"-c:0", "aac", "-ac", "2", "-ab", strconv.Itoa(transcodingParams.audioBitrate),
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", startNumber),
		"-hls_time", fmt.Sprintf("%.3f", transcodedAudioSegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u")}

	cmd := exec.Command("ffmpeg", args...)
	log.Println("ffmpeg initialized with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	//cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:            cmd,
		Stream:         stream,
		outputDir:      outputDir,
		firstSegmentId: segmentOffset,
	}, nil
}

func GetOfferedTranscodedAudioStreams(container ProbeContainer) []OfferedStream {
	offeredStreams := []OfferedStream{}

	numFullSegments := int64(container.Format.Duration() / transcodedAudioSegmentDuration)
	segmentStartTimestamps := []time.Duration{}
	for i := int64(0); i < numFullSegments+1; i++ {
		segmentStartTimestamps = append(segmentStartTimestamps,
			time.Duration(i*int64(transcodedAudioSegmentDuration)))
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "audio" {
			continue
		}

		language := stream.Tags["language"]
		if language == "" {
			language = "unk"
		}
		title := stream.Tags["title"]
		if title == "" {
			title = language
		}
		// TODO(Leon Handreke): Render a user-presentable language string.

		for representationId, encoderParams := range AudioEncoderPresets {
			offeredStreams = append(offeredStreams, OfferedStream{
				StreamKey: StreamKey{
					StreamId:         int64(stream.Index),
					RepresentationId: representationId,
				},
				BitRate:                int64(encoderParams.audioBitrate),
				TotalDuration:          container.Format.Duration(),
				Codecs:                 "mp4a.40.2",
				StreamType:             "audio",
				Language:               GetLanguageTag(stream),
				Title:                  GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault:       stream.Disposition["default"] != 0,
				transcoded:             true,
				SegmentStartTimestamps: segmentStartTimestamps,
			})
		}
	}

	return offeredStreams
}
