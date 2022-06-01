package ffmpeg

import (
	"fmt"
	"github.com/abema/go-mp4"
	"github.com/shirou/gopsutil/v3/process"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

// InitialSegmentIdx is a magic segment index value to denote the initial segment
var InitialSegmentIdx int = -1

// TranscodingSession contains many attributes which are only public because they are displayed on the debug page.
type TranscodingSession struct {
	cmd               *exec.Cmd
	stateChangeMutex  sync.Mutex
	State             SessionState
	Stream            StreamRepresentation
	OutputDir         string
	ProcessWaitGroup  sync.WaitGroup
	ProgressPercent   float32
	SegmentStartIndex int
}

func (s *TranscodingSession) Start() error {
	s.ProcessWaitGroup.Add(1)

	// Tells it to create a separate process group for FFmpeg that can be
	// terminated separately from Olaris' process group
	s.cmd.SysProcAttr = &syscall.SysProcAttr{}
	s.cmd.SysProcAttr.Setpgid = true

	s.stateChangeMutex.Lock()
	defer s.stateChangeMutex.Unlock()

	if err := s.cmd.Start(); err != nil {
		s.ProcessWaitGroup.Done()
		return err
	}
	s.State = SessionStateRunning

	// Prevent zombies
	go func() {
		s.cmd.Wait()

		s.stateChangeMutex.Lock()
		s.State = SessionStateExited
		s.stateChangeMutex.Unlock()

		s.ProcessWaitGroup.Done()
	}()

	return nil
}

// Destroy destroys the FFmpeg process group if it's still running, and removes
// its output directory.
func (s *TranscodingSession) Destroy() error {
	if s.State != SessionStateExited {
		s.stateChangeMutex.Lock()
		s.resumeUnlocked()
		s.State = SessionStateStopping
		s.stateChangeMutex.Unlock()

		// Signal the process group (-pid), not just the process, so that the
		// process and all its children are signaled. Else, child procs can keep
		// running and keep the stdout/stderr fd open and cause cmd.Wait to
		// hang. No error handling, we don't care if ffmpeg errors out, we're
		// done here anyway.
		pgid, err := syscall.Getpgid(s.cmd.Process.Pid)
		if err == nil {
			log.WithFields(log.Fields{
				"pid":  s.cmd.Process.Pid,
				"pgid": pgid,
			}).Debugln("killing ffmpeg process")

			syscall.Kill(-pgid, syscall.SIGTERM)
		}
	}

	// Wait for the FFmpeg process to be done and then clean up the output directory
	s.ProcessWaitGroup.Wait()
	log.WithFields(log.Fields{"dir": s.OutputDir}).Debugln("removing ffmpeg outputdir")
	err := os.RemoveAll(s.OutputDir)

	return err
}

// FindSegmentByIndex looks for a media segment file with the provided index
// and returns the file path if it exists.
//
// Note: Also makes sure the next file exists to verify that the segment being
// requested is not still being written.
func (s *TranscodingSession) FindSegmentByIndex(index int) (string, error) {
	segmentPath := s.segmentPathForIndex(index)
	_, err := os.Stat(segmentPath)
	if err != nil {
		return segmentPath, err
	}

	// If the FFmpeg process has exited, assume this segment is written
	if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
		return segmentPath, nil
	}

	// Look for the next segment to make sure the current one is finished
	nextSegmentPath := s.segmentPathForIndex(index + 1)
	_, err = os.Stat(nextSegmentPath)
	if err != nil {
		return segmentPath, err
	}

	return segmentPath, nil
}

// segmentPathForIndex returns the full path to the requested segment. It
// does not guarantee that the segment exists.
func (s *TranscodingSession) segmentPathForIndex(index int) string {
	if index == InitialSegmentIdx {
		return filepath.Join(s.OutputDir, "init.mp4")
	}

	return filepath.Join(s.OutputDir, fmt.Sprintf("stream0_%d.m4s", index))
}

// PatchSegment patches a media segment so that it can be played via HLS/DASH as
// if it is part of an existing transcode session. If this is not patched, the
// timestamps will be off if not starting at segment 0.
func (s *TranscodingSession) PatchSegment(segmentPath string) (io.ReadSeeker, error) {
	segmentFile, err := os.OpenFile(segmentPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer segmentFile.Close()

	memoryBuffer := &MemoryWriteSeeker{}
	w := mp4.NewWriter(memoryBuffer)

	// Copy each box, patching anything we need to
	_, err = mp4.ReadBoxStructure(segmentFile, func(h *mp4.ReadHandle) (interface{}, error) {
		switch h.BoxInfo.Type {

		// Some box types need to be expanded, so we can reach their children
		case mp4.BoxTypeMoof(), mp4.BoxTypeMoov():
			if _, err := w.StartBox(&h.BoxInfo); err != nil {
				return nil, err
			}
			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}
			if _, err := mp4.Marshal(w, box, h.BoxInfo.Context); err != nil {
				return nil, err
			}
			if _, err := h.Expand(); err != nil {
				return nil, err
			}
			_, err = w.EndBox()
			return nil, err

		// Need to update the sequence number in the mfhd box
		case mp4.BoxTypeMfhd():
			if _, err := w.StartBox(&h.BoxInfo); err != nil {
				return nil, err
			}

			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}

			mfhd := box.(*mp4.Mfhd)
			mfhd.SequenceNumber = mfhd.SequenceNumber + uint32(s.SegmentStartIndex)

			if _, err := mp4.Marshal(w, mfhd, h.BoxInfo.Context); err != nil {
				return nil, err
			}

			_, err = w.EndBox()
			return nil, err

		// Any other box type can be copied as-is
		default:
			return nil, w.CopyBox(segmentFile, &h.BoxInfo)
		}
	})

	return memoryBuffer.BytesReader(), nil
}

// Suspend pauses the transcoding session's FFmpeg process. Returns an error if
// the process is not found.
func (s *TranscodingSession) Suspend() error {
	s.stateChangeMutex.Lock()
	defer s.stateChangeMutex.Unlock()

	if s.State != SessionStateRunning {
		return nil
	}

	p, err := process.NewProcess(int32(s.cmd.Process.Pid))
	if err != nil {
		return err
	}

	err = p.Suspend()
	if err != nil {
		return err
	}

	s.State = SessionStateThrottled
	return nil
}

// Resume resumes the transcoding session's FFmpeg process. Returns an error if
// the process is not found.
func (s *TranscodingSession) Resume() error {
	s.stateChangeMutex.Lock()
	defer s.stateChangeMutex.Unlock()

	return s.resumeUnlocked()
}

// Private method that resumes the FFmpeg process without acquiring the state
// change lock. This is required because sometimes the process needs to be
// resumed as part of another state change.
// CAUTION: You should acquire a lock on the state change mutex before calling
// this method.
func (s *TranscodingSession) resumeUnlocked() error {
	if s.State != SessionStateThrottled {
		return nil
	}

	p, err := process.NewProcess(int32(s.cmd.Process.Pid))
	if err != nil {
		return err
	}

	err = p.Resume()
	if err != nil {
		return err
	}

	s.State = SessionStateRunning
	return nil
}
