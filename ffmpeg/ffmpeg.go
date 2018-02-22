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
	InputPath      string
	outputDir      string
	firstSegmentId int
	// Usually something like "direct-stream", to which "-video" and "-audio" will be appended
	RepresentationIdBase string
	// Output streams of this session
	streams []string
}

// MinSegDuration defines the duration of segments that ffmpeg will generate. In the transmuxing case this is really
// just a minimum time, the actual segments will be longer because they are cut at keyframes. For transcoding, we can
// force keyframes to occur exactly every MinSegDuration, so MinSegDuration will be the actualy duration of the
// segments.
const MinSegDuration = 5 * time.Second

// fragmentsPerSession defines the number of segments to encode per launch of ffmpeg. This constant should strike a
// balance between minimizing the overhead cause by launching new ffmpeg processes and minimizing the minutes of video
// transcoded but never watched by the user. Note that this constant is currently only used for the transcoding case.
const segmentsPerSession = 6

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(inputPath string, outputDirBase string, startDuration time.Duration, segmentOffset int) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", strconv.FormatInt(int64(startDuration/time.Second), 10),
		"-i", inputPath,
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "dash",
		"-min_seg_duration", strconv.FormatInt(int64(MinSegDuration/time.Microsecond), 10),
		// segment_start_number requires a custom ffmpeg
		// +1 because ffmpeg likes to start segments at 1. The reverse transformation happens in AvailableSegments.
		"-segment_start_number", strconv.FormatInt(int64(startDuration/MinSegDuration)+1, 10),
		"-media_seg_name", "stream$RepresentationID$_$Number$.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.mpd"))
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout

	return &TranscodingSession{cmd: cmd, InputPath: inputPath, outputDir: outputDir, firstSegmentId: segmentOffset}, nil
}

type EncoderParams struct {
	// One of these may be -1 to keep aspect ratio
	width        int
	height       int
	videoBitrate int
	audioBitrate int
}

var EncoderPresets = map[string]EncoderParams{
	"480-1000k":   EncoderParams{height: 480, width: -1, videoBitrate: 1000000, audioBitrate: 64000},
	"720-5000k":   EncoderParams{height: 720, width: -1, videoBitrate: 5000000, audioBitrate: 128000},
	"1080-10000k": EncoderParams{height: 1080, width: -1, videoBitrate: 10000000, audioBitrate: 128000},
}

// NewTranscodingSession starts a new transcoding session.
// It returns the process that was started and any error it encountered while starting it.
func NewTranscodingSession(
	inputPath string,
	outputDirBase string,
	startDuration time.Duration,
	segmentOffset int,
	transcodingParams EncoderParams) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", strconv.FormatInt(int64(startDuration/time.Second), 10),
		"-i", inputPath,
		"-to", strconv.FormatInt(int64((startDuration+segmentsPerSession*MinSegDuration)/time.Second), 10),
		"-copyts",
		"-c:v", "libx264", "-b:v", strconv.Itoa(transcodingParams.videoBitrate), "-preset:v", "veryfast",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", MinSegDuration/time.Second),
		"-c:a", "aac", "-ac", "2", "-ab", strconv.Itoa(transcodingParams.audioBitrate),
		"-f", "hls",
		"-start_number", strconv.FormatInt(int64(startDuration/MinSegDuration), 10),
		"-hls_time", strconv.FormatInt(int64(MinSegDuration/time.Second), 10),
		"-hls_segment_type", "1",
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.mpd"))
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{cmd: cmd, InputPath: inputPath, outputDir: outputDir, firstSegmentId: segmentOffset}, nil
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
func (s *TranscodingSession) GetSegment(streamId string, segmentId int, deadline time.Duration) (string, error) {

	if !s.IsProjectedAvailable(streamId, segmentId, deadline) {
		return "", fmt.Errorf("Segment not projected to be available within deadline %s", deadline)
	}

	for {
		availableSegments, _ := s.AvailableSegments(streamId)
		if path, ok := availableSegments[segmentId]; ok {
			return path, nil
		}
		// TODO(Leon Handreke): Maybe a condition variable? Or maybe this blocking should move to the server module?
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *TranscodingSession) IsProjectedAvailable(streamId string, segmentId int, deadline time.Duration) bool {
	// For transmuxed content we currently just spew out the whole file and serve it.
	if s.RepresentationIdBase == "direct-stream" {
		return true
	}

	return s.firstSegmentId <= segmentId && segmentId < s.firstSegmentId+segmentsPerSession
}

func (s *TranscodingSession) AvailableSegments(streamId string) (map[int]string, error) {
	res := make(map[int]string)

	files, err := ioutil.ReadDir(s.outputDir)
	if err != nil {
		return nil, err
	}

	var streamFilenamePrefix string

	if streamId == "video" {
		streamFilenamePrefix = "stream0"
	} else if streamId == "audio" {
		streamFilenamePrefix = "stream1"
	} else {
		return nil, fmt.Errorf("Invalid stream ID %s", streamId)
	}

	r := regexp.MustCompile(streamFilenamePrefix + "_(?P<number>\\d+).m4s")

	for _, f := range files {
		match := r.FindString(f.Name())
		if match != "" {
			segmentFsNumber, _ := strconv.Atoi(match[len("segment_") : len(match)-len(".m4s")])
			res[segmentFsNumber] = filepath.Join(s.outputDir, f.Name())
		}

	}

	return res, nil
}

// InitialSegment returns the path of the initial segment for the given stream
// or error if no initial segment is available for the given stream.
func (s *TranscodingSession) InitialSegment(streamId string) (string, error) {
	if streamId == "video" {
		return filepath.Join(s.outputDir, "init.mp4"), nil
	}
	if streamId == "audio" {
		return filepath.Join(s.outputDir, "init-stream1.m4s"), nil
	}
	return "", fmt.Errorf("No initial segment for the given stream \"%s\"", streamId)
}

func GuessSegmentDurations(keyframeTimestamps []time.Duration, totalDuration time.Duration, minSegDuration time.Duration) []time.Duration {
	// Insert dummy keyframe timestamp at the end so that the last segment duration is correctly reported
	keyframeTimestamps = append(keyframeTimestamps, totalDuration)

	segmentDurations := []time.Duration{}
	lastKeyframe := 0
	for i, keyframe := range keyframeTimestamps {
		if i == 0 {
			continue
		}
		d := keyframe - keyframeTimestamps[lastKeyframe]
		if d > minSegDuration {
			segmentDurations = append(segmentDurations, d)
			lastKeyframe = i
		}
	}

	return segmentDurations
}
