package ffmpeg

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
)

// Magic segment index value to denote the inital segment
var InitialSegmentIdx int = -1

type TranscodingSession struct {
	cmd        *exec.Cmd
	Stream     StreamRepresentation
	outputDir  string
	terminated bool
	throttled  bool
}

func (s *TranscodingSession) Start() error {
	if err := s.cmd.Start(); err != nil {
		return err
	}
	// Prevent zombies
	go func() {
		s.cmd.Wait()
		s.terminated = true
	}()
	return nil
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

func (s *TranscodingSession) AvailableSegments() (map[int]string, error) {
	res := make(map[int]string)

	initialSegmentPath := filepath.Join(s.outputDir, "init.mp4")
	if stat, err := os.Stat(initialSegmentPath); err == nil {
		if stat.Size() > 0 {
			res[InitialSegmentIdx] = initialSegmentPath
		}
	}

	files, err := ioutil.ReadDir(s.outputDir)
	if err != nil {
		return nil, err
	}

	r := regexp.MustCompile("stream0_(?P<number>\\d+).m4s$")

	maxSegmentId := 0

	for _, f := range files {
		match := r.FindString(f.Name())
		if match != "" {
			segmentFsNumber, _ := strconv.Atoi(match[len("stream0_") : len(match)-len(".m4s")])
			res[segmentFsNumber] = filepath.Join(s.outputDir, f.Name())

			if segmentFsNumber > maxSegmentId {
				maxSegmentId = segmentFsNumber
			}
		}
	}

	// We delete the "newest" segment because it may still be written to to avoid races.
	if len(res) > 0 && !s.terminated {
		delete(res, maxSegmentId)
	}

	return res, nil
}

func (s *TranscodingSession) SetThrottled(throttled bool) {
	log.Infof("Toggling throttled state to %t on %s", throttled, s.outputDir)
	if throttled {
		s.cmd.Process.Signal(syscall.SIGUSR1)
	} else {
		s.cmd.Process.Signal(syscall.SIGUSR2)
	}
}
