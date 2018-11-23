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

var AudioEncoderPresets = map[string]EncoderParams{
	"64k-audio":  {audioBitrate: 64000, Codecs: "mp4a.40.2"},
	"128k-audio": {audioBitrate: 128000, Codecs: "mp4a.40.2"},
}

// This is exactly 234 AAC frames (1024 samples each) @ 48kHz.
// TODO(Leon Handreke): Do we need to set this differently for different sampling rates?
const transcodedAudioSegmentDuration = 4992 * time.Millisecond

func NewAudioTranscodingSession(
	stream StreamRepresentation,
	startTime time.Duration,
	segmentStartIndex int,
	outputDirBase string) (*TranscodingSession, error) {

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
		"-c:0", "aac", "-ac", "2", "-ab", strconv.Itoa(encoderParams.audioBitrate),
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentStartIndex),
		"-hls_time", fmt.Sprintf("%.3f", transcodedAudioSegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u")}

	cmd := exec.Command(executable.GetFFmpegExecutablePath(), args...)
	log.Println("ffmpeg started with", cmd.Args)

	logSink := getTranscodingLogSink("ffmpeg_transcode_audio")
	cmd.Stderr = logSink

	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    stream,
		outputDir: outputDir,
	}, nil
}

func GetTranscodedAudioRepresentation(stream Stream, representationId string, encoderParams EncoderParams) StreamRepresentation {
	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.audioBitrate,
			Container:        "audio/mp4",
			Codecs:           encoderParams.Codecs,
			Transcoded:       true,
		},
	}
}

func buildAudioSegmentDurations(interval Interval, segmentDuration time.Duration) [][]Segment {
	sessions := [][]Segment{}

	session := []Segment{}
	currentTimestamp := interval.StartTimestamp
	segmentId := 0

	segmentDurationDts := DtsTimestamp(segmentDuration.Seconds() * float64(interval.TimeBase))

	for currentTimestamp < interval.EndTimestamp {
		if len(session) >= segmentsPerSession {
			sessions = append(sessions, session)
			session = []Segment{}
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
