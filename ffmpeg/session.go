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

// InitialSegmentIdx is a magic segment index value to denote the initial segment
var InitialSegmentIdx int = -1

// TranscodingSession contains many attributes which are only public because they are displayed on the debug page.
type TranscodingSession struct {
	cmd             *exec.Cmd
	Stream          StreamRepresentation
	OutputDir       string
	Terminated      bool
	Throttled       bool
	ProgressPercent float32
}

func (s *TranscodingSession) Start() error {
	if err := s.cmd.Start(); err != nil {
		return err
	}
	// Prevent zombies
	go func() {
		s.cmd.Wait()
		s.Terminated = true
	}()

	return nil
}

func (s *TranscodingSession) Destroy() error {
	// Signal the process group (-pid), not just the process, so that the process
	// and all its children are signaled. Else, child procs can keep running and
	// keep the stdout/stderr fd open and cause cmd.Wait to hang.
	log.WithFields(log.Fields{"pid": s.cmd.Process.Pid}).Debugln("killing ffmpeg process")

	syscall.Kill(-s.cmd.Process.Pid, syscall.SIGTERM)
	// No error handling, we don't care if ffmpeg errors out, we're done here anyway.
	s.cmd.Wait()

	log.WithFields(log.Fields{"dir": s.OutputDir}).Debugln("removing ffmpeg outputdir")

	err := os.RemoveAll(s.OutputDir)
	if err != nil {
		return err
	}

	return nil
}

func (s *TranscodingSession) AvailableSegments() (map[int]string, error) {
	res := make(map[int]string)

	initialSegmentPath := filepath.Join(s.OutputDir, "init.mp4")
	if stat, err := os.Stat(initialSegmentPath); err == nil {
		if stat.Size() > 0 {
			res[InitialSegmentIdx] = initialSegmentPath
		}
	}

	files, err := ioutil.ReadDir(s.OutputDir)
	if err != nil {
		return nil, err
	}

	r := regexp.MustCompile("stream0_(?P<number>\\d+).m4s$")

	maxSegmentID := 0

	for _, f := range files {
		match := r.FindString(f.Name())
		if match != "" {
			segmentFsNumber, _ := strconv.Atoi(match[len("stream0_") : len(match)-len(".m4s")])
			res[segmentFsNumber] = filepath.Join(s.OutputDir, f.Name())

			if segmentFsNumber > maxSegmentID {
				maxSegmentID = segmentFsNumber
			}
		}
	}

	// We delete the "newest" segment because it may still be written to to avoid races.
	if len(res) > 0 && !s.Terminated {
		delete(res, maxSegmentID)
	}

	return res, nil

}
