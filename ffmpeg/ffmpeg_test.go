package ffmpeg

import (
	"testing"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestTranscodingSession_AvailableSegments(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "test-transcoding-session-available-segments")
	seg1 := createEmptyFile(tempDir, "segment_1.m4s")
	seg2 := createEmptyFile(tempDir, "segment_2.m4s")
	seg3 := createEmptyFile(tempDir, "segment_3.m4s")
	seg5 := createEmptyFile(tempDir, "segment_5.m4s")
	createEmptyFile(tempDir, "segmentother.m4s")
	createEmptyFile(tempDir, "other_file.mpd")

	s := TranscodingSession{segmentOffset: 10, outputDir: tempDir}

	availableSegments, _ := s.AvailableSegments()
	fmt.Println(availableSegments)
	assert.Equal(t, map[int]string{11: seg1, 12: seg2, 13: seg3, 15: seg5}, availableSegments)
}

func createEmptyFile(path string, name string) string {
	f, _ := os.OpenFile(filepath.Join(path, name), os.O_RDONLY|os.O_CREATE, 0666)
	f.Close()
	return filepath.Join(path, name)
}