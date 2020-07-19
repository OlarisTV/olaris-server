package ffmpeg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg/executable"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(
	stream StreamRepresentation,
	startTime time.Duration,
	segmentStartIndex int,
	outputDirBase string,
	feedbackURL string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

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
		"-map", fmt.Sprintf("0:%d", stream.Stream.StreamId),
		"-c:0", "copy",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentStartIndex),
		"-hls_time", fmt.Sprintf("%.3f", SegmentDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		"-olaris_feedback_url", feedbackURL,
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"),
	}...)

	cmd := exec.Command(executable.GetFFmpegExecutablePath(), args...)
	log.Infoln("ffmpeg started with", cmd.Args)
	cmd.Dir = outputDir

	logSink := getTranscodingLogSink("ffmpeg_transmux")
	//io.WriteString(logSink, fmt.Sprintf("%s %s\n\n", cmd.Args, options.String()))
	cmd.Stderr = logSink
	cmd.Stdout = os.Stdout

	//stdin, _ := cmd.StdinPipe()
	//stdin.Write(optionsSerialized)
	//stdin.Close()

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    stream,
		OutputDir: outputDir,
	}, nil
}

func GetTransmuxedRepresentation(stream Stream) StreamRepresentation {
	representation := StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: "direct",
			Container:        "video/mp4",
			Codecs:           stream.Codecs,
			BitRate:          int(stream.BitRate),
			Height:           stream.Height,
			Width:            stream.Width,
			Transmuxed:       true,
		},
	}

	return representation
}

func guessTransmuxedSegmentList(keyframeIntervals []Interval) [][]Segment {
	segmentId := 0
	var sessions [][]Segment
	timeBase := keyframeIntervals[0].TimeBase
	segDurationTs := DtsTimestamp(SegmentDuration.Seconds() * float64(timeBase))

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

	return sessions
}
