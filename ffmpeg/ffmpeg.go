// Convenience wrapper around ffmpeg as a transcoder to DASH chunks
// https://github.com/go-cmd/cmd/blob/master/cmd.go was very useful while writing this module.
package ffmpeg

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

type TranscodingSession struct {
	cmd            *exec.Cmd
	Stream         OfferedStream
	outputDir      string
	firstSegmentId int64
}

type StreamKey struct {
	MediaFilePath string
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId         int64
	RepresentationId string
}

type OfferedStream struct {
	StreamKey

	// The rest is just metadata for display
	BitRate       int64
	TotalDuration time.Duration
	// codecs string ready for DASH/HLS serving
	Codecs string
	// "audio", "video", "subtitle"
	StreamType string
	// Only relevant for audio and subtitles. Language code.
	Language string
	// User-visible string for this audio or subtitle track
	Title            string
	EnabledByDefault bool

	SegmentStartTimestamps []time.Duration

	// Mutually exclusive
	transcoded bool
	transmuxed bool
}

// MinSegDuration defines the duration of segments that ffmpeg will generate. In the transmuxing case this is really
// just a minimum time, the actual segments will be longer because they are cut at keyframes. For transcoding, we can
// force keyframes to occur exactly every MinSegDuration, so MinSegDuration will be the actualy duration of the
// segments.
const MinTransmuxedSegDuration = 5000 * time.Millisecond

// fragmentsPerSession defines the number of segments to encode per launch of ffmpeg. This constant should strike a
// balance between minimizing the overhead cause by launching new ffmpeg processes and minimizing the minutes of video
// transcoded but never watched by the user. Note that this constant is currently only used for the transcoding case.
const segmentsPerSession = 12

func sum(input ...time.Duration) time.Duration {
	var sum time.Duration
	for _, i := range input {
		sum += i
	}
	return sum
}

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(stream OfferedStream, outputDirBase string, segmentOffset int64) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startTimestamp := stream.SegmentStartTimestamps[segmentOffset]
	var endTimestamp time.Duration
	if segmentOffset+segmentsPerSession >= int64(len(stream.SegmentStartTimestamps)) {
		endTimestamp = stream.TotalDuration
	} else {
		endTimestamp = stream.SegmentStartTimestamps[segmentOffset+segmentsPerSession]
	}

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startTimestamp.Seconds()),
		"-i", stream.MediaFilePath,
		"-copyts",
		"-to", fmt.Sprintf("%.3f", endTimestamp.Seconds()),
		"-map", fmt.Sprintf("0:%d", stream.StreamId),
		"-c:0", "copy",
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentOffset),
		"-hls_time", fmt.Sprintf("%.3f", MinTransmuxedSegDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"))
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:            cmd,
		Stream:         stream,
		outputDir:      outputDir,
		firstSegmentId: segmentOffset,
	}, nil
}

type EncoderParams struct {
	// One of these may be -1 to keep aspect ratio
	width        int
	height       int
	videoBitrate int
	audioBitrate int
}

func (s *TranscodingSession) Start() error {
	return s.cmd.Start()
}

func (s *TranscodingSession) Destroy() error {
	// Signal the process group (-pid), not just the process, so that the process
	// and all its children are signaled. Else, child procs can keep running and
	// keep the stdout/stderr fd open and cause cmd.Wait to hang.
	syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
	// No error handling, we don't care if ffmpeg errors out, we're done here anyway.
	s.cmd.Wait()

	err := os.RemoveAll(s.outputDir)
	if err != nil {
		return err
	}

	return nil
}

// GetSegment return the filename of the given segment if it is projected to be available by the given deadline.
// It will block for at most deadline.
func (s *TranscodingSession) GetSegment(segmentId int64, deadline time.Duration) (string, error) {

	if !s.IsProjectedAvailable(segmentId, deadline) {
		return "", fmt.Errorf("Segment not projected to be available within deadline %s", deadline)
	}

	for {
		availableSegments, _ := s.AvailableSegments()
		if path, ok := availableSegments[segmentId]; ok {
			return path, nil
		}
		// TODO(Leon Handreke): Maybe a condition variable? Or maybe this blocking should move to the server module?
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *TranscodingSession) IsProjectedAvailable(segmentId int64, deadline time.Duration) bool {
	if s.Stream.RepresentationId == "webvtt" {
		return true
	}

	return s.firstSegmentId <= segmentId && segmentId < s.firstSegmentId+segmentsPerSession
}

func (s *TranscodingSession) AvailableSegments() (map[int64]string, error) {
	res := make(map[int64]string)

	files, err := ioutil.ReadDir(s.outputDir)
	if err != nil {
		return nil, err
	}

	r := regexp.MustCompile("stream0_(?P<number>\\d+).m4s")

	for _, f := range files {
		match := r.FindString(f.Name())
		if match != "" {
			segmentFsNumber, _ := strconv.Atoi(match[len("segment_") : len(match)-len(".m4s")])
			res[int64(segmentFsNumber)] = filepath.Join(s.outputDir, f.Name())
		}

	}

	return res, nil
}

// InitialSegment returns the path of the initial segment for the given stream
// or error if no initial segment is available for the given stream.
func (s *TranscodingSession) InitialSegment() string {
	return filepath.Join(s.outputDir, "init.mp4")
}

func GuessTransmuxedSegmentStartTimestamps(keyframeTimestamps []time.Duration) []time.Duration {
	segmentTimestamps := []time.Duration{
		// First keyframe should equal first frame, but who knows, video is weird...
		keyframeTimestamps[0],
	}
	for _, keyframe := range keyframeTimestamps {
		d := keyframe - segmentTimestamps[len(segmentTimestamps)-1]
		if d > MinTransmuxedSegDuration {
			segmentTimestamps = append(segmentTimestamps, keyframe)
		}
	}

	return segmentTimestamps
}

func ComputeSegmentDurations(
	segmentStartTimestamps []time.Duration,
	totalDuration time.Duration) []time.Duration {

	// Insert dummy keyframe timestamp at the end so that the last segment duration is correctly reported
	segmentStartTimestamps = append(segmentStartTimestamps, totalDuration)

	segmentDurations := []time.Duration{}

	for i := 1; i < len(segmentStartTimestamps); i++ {
		segmentDurations = append(segmentDurations,
			segmentStartTimestamps[i]-segmentStartTimestamps[i-1])
	}

	return segmentDurations
}

func GetOfferedTranscodedStreams(mediaFilePath string) ([]OfferedStream, error) {
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err

	}

	streams := append(
		GetOfferedTranscodedVideoStreams(*container),
		GetOfferedTranscodedAudioStreams(*container)...)

	for i, _ := range streams {
		streams[i].MediaFilePath = mediaFilePath
	}

	return streams, nil
}

func GetOfferedTransmuxedStreams(mediaFilePath string) ([]OfferedStream, error) {
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err
	}

	keyframeTimestamps, err := ProbeKeyframes(mediaFilePath)
	if err != nil {
		return []OfferedStream{}, err
	}
	segmentStartTimestamps := GuessTransmuxedSegmentStartTimestamps(keyframeTimestamps)

	offeredStreams := []OfferedStream{}
	for _, stream := range container.Streams {
		if stream.CodecType != "audio" && stream.CodecType != "video" {
			continue
		}
		bitrate, _ := strconv.Atoi(stream.BitRate)

		offeredStreams = append(offeredStreams,
			OfferedStream{
				StreamKey: StreamKey{
					MediaFilePath:    mediaFilePath,
					StreamId:         int64(stream.Index),
					RepresentationId: "direct-stream",
				},
				Codecs:                 stream.GetMime(),
				BitRate:                int64(bitrate),
				TotalDuration:          container.Format.Duration(),
				StreamType:             stream.CodecType,
				Language:               GetLanguageTag(stream),
				Title:                  GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault:       stream.Disposition["default"] != 0,
				transmuxed:             true,
				SegmentStartTimestamps: segmentStartTimestamps,
			})
	}

	return offeredStreams, nil
}

func GetOfferedStreams(mediaFilePath string) ([]OfferedStream, error) {
	transcoded, err := GetOfferedTranscodedStreams(mediaFilePath)
	if err != nil {
		return []OfferedStream{}, err
	}
	transmuxed, err := GetOfferedTransmuxedStreams(mediaFilePath)
	if err != nil {
		return []OfferedStream{}, err
	}

	subtitles, err := GetOfferedSubtitleStreams(mediaFilePath)
	if err != nil {
		return []OfferedStream{}, err
	}

	return append(transcoded, append(transmuxed, subtitles...)...), nil
}

func FindStream(streams []OfferedStream, streamId int64, representationId string) (OfferedStream, bool) {
	for _, s := range streams {
		if s.StreamId == streamId && s.RepresentationId == representationId {
			return s, true
		}
	}
	return OfferedStream{}, false
}
