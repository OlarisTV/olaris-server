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

var VideoEncoderPresets = map[string]EncoderParams{
	"480-1000k-video":   EncoderParams{height: 480, width: -2, videoBitrate: 1000000},
	"720-5000k-video":   EncoderParams{height: 720, width: -2, videoBitrate: 5000000},
	"1080-10000k-video": EncoderParams{height: 1080, width: -2, videoBitrate: 10000000},
}

// Doesn't have to be the same as audio, but why not.
const transcodedVideoSegmentDuration = 4992 * time.Millisecond

func NewVideoTranscodingSession(
	stream StreamRepresentation,
	outputDirBase string,
	segmentOffset int64,
	transcodingParams EncoderParams) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startDuration := time.Duration(int64(transcodedVideoSegmentDuration) * segmentOffset)
	runDuration := segmentsPerSession * transcodedVideoSegmentDuration

	args := []string{
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.Stream.MediaFilePath,
		"-to", fmt.Sprintf("%.3f", (startDuration + runDuration).Seconds()),
		"-copyts",
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "libx264", "-b:v", strconv.Itoa(transcodingParams.videoBitrate),
		"-preset:0", "veryfast",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%.3f)", transcodedVideoSegmentDuration.Seconds()),
		"-filter:0", fmt.Sprintf("scale=%d:%d", transcodingParams.width, transcodingParams.height),
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentOffset),
		"-hls_time", fmt.Sprintf("%.3f", transcodedVideoSegmentDuration.Seconds()),
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

func GetTranscodedVideoRepresentations(stream Stream) []StreamRepresentation {
	representations := []StreamRepresentation{}

	numFullSegments := int64(stream.TotalDuration / transcodedVideoSegmentDuration)
	segmentStartTimestamps := []time.Duration{}
	for i := int64(0); i < numFullSegments+1; i++ {
		segmentStartTimestamps = append(segmentStartTimestamps,
			time.Duration(i*int64(transcodedVideoSegmentDuration)))
	}

	for representationId, encoderParams := range VideoEncoderPresets {
		codecsString := "avc1.64001e"
		if representationId == "480-1000k-video" {
			codecsString = "avc1.64001e"
		}
		if representationId == "720-5000k-video" {
			codecsString = "avc1.64001f"
		}
		if representationId == "1080-10000k-video" {
			codecsString = "avc1.640028"
		}

		representations = append(representations, StreamRepresentation{
			Stream: stream,
			Representation: Representation{
				RepresentationId: representationId,
				BitRate:          int64(encoderParams.audioBitrate),
				// TODO(Leon Handreke): Container/Codecs belongs in encoderParams
				Container:  "video/mp4",
				Codecs:     codecsString,
				transcoded: true,
			},
			SegmentStartTimestamps: segmentStartTimestamps,
		})
	}
	return representations
}
