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

type OfferedStream struct {
	MediaFilePath string
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId         int64
	RepresentationId string

	// The rest is just metadata for display
	// TODO(Leon Handreke): Should probably pull this "primary key"
	// out into a StreamKey struct or something

	BitRate       int64
	TotalDuration time.Duration
	// codecs string ready for DASH/HLS serving
	Codecs string
	// "audio", "video", "subtitle"
	StreamType string
	// Only relevant for audio and subtitles. Language code.
	Language string
	// User-visible string for this audio or subtitle track
	Title string

	SegmentDurations []time.Duration

	// Mutually exclusive
	transcoded bool
	transmuxed bool
}

func (s *OfferedStream) Equals(other OfferedStream) bool {
	return (s.MediaFilePath == other.MediaFilePath) &&
		(s.RepresentationId == other.RepresentationId) &&
		(s.StreamId == other.StreamId)
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

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(stream OfferedStream, outputDirBase string, segmentOffset int64) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	startDuration := time.Duration(int64(MinTransmuxedSegDuration) * segmentOffset)

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startDuration.Seconds()),
		"-i", stream.MediaFilePath,
		"-c:v", "copy",
		"-c:a", "copy",
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
	// For transmuxed content we currently just spew out the whole file and serve it.
	if s.Stream.RepresentationId == "direct-stream-video" {
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

func GuessTransmuxedSegmentDurations(filename string, totalDuration time.Duration) ([]time.Duration, error) {
	keyframeTimestamps, err := ProbeKeyframes(filename)
	if err != nil {
		return nil, err
	}

	// Insert dummy keyframe timestamp at the end so that the last segment duration is correctly reported
	keyframeTimestamps = append(keyframeTimestamps, totalDuration)

	segmentDurations := []time.Duration{}
	lastKeyframe := 0
	for i, keyframe := range keyframeTimestamps {
		if i == 0 {
			continue
		}
		d := keyframe - keyframeTimestamps[lastKeyframe]
		if d > MinTransmuxedSegDuration {
			segmentDurations = append(segmentDurations, d)
			lastKeyframe = i
		}
	}

	return segmentDurations, nil
}

func GetOfferedTranscodedStreams(mediaFilePath string) ([]OfferedStream, error) {
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err

	}

	streams := append(
		GetOfferedTranscodedVideoStreams(*container),
		GetOfferedTranscodedAudioStreams(*container)...)
	for _, s := range streams {
		s.MediaFilePath = mediaFilePath
	}

	return streams, nil
}

func GetOfferedTransmuxedStreams(mediaFilePath string) ([]OfferedStream, error) {
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err

	}

	var videoStream ProbeStream
	var audioStream ProbeStream

	for _, s := range container.Streams {
		if s.CodecType == "audio" {
			audioStream = s
		} else if s.CodecType == "video" {
			videoStream = s
		}
	}

	videoBitrate, _ := strconv.Atoi(videoStream.BitRate)
	audioBitrate, _ := strconv.Atoi(audioStream.BitRate)
	segmentDurations, _ := GuessTransmuxedSegmentDurations(mediaFilePath, container.Format.Duration())

	return []OfferedStream{
		{
			MediaFilePath:    mediaFilePath,
			StreamId:         0,
			RepresentationId: "direct-stream-video",
			Codecs:           fmt.Sprint("%s,%s", videoStream.GetMime(), audioStream.GetMime()),
			BitRate:          int64(videoBitrate + audioBitrate),
			TotalDuration:    container.Format.Duration(),
			StreamType:       "video",
			transmuxed:       true,
			SegmentDurations: segmentDurations,
		},
	}, nil
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

	return append(transcoded, transmuxed...), nil
}

func FindStream(streams *[]OfferedStream, streamId int64, representationId string) (OfferedStream, bool) {
	for _, s := range *streams {
		if s.StreamId == streamId && s.RepresentationId == representationId {
			return s, true
		}
	}
	return OfferedStream{}, false
}
