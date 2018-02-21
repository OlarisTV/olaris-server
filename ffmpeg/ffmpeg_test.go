package ffmpeg

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestTranscodingSession_AvailableSegments(t *testing.T) {
	tempDir, _ := ioutil.TempDir(os.TempDir(), "test-transcoding-session-available-segments")
	seg1 := createEmptyFile(tempDir, "stream0_1.m4s")
	seg2 := createEmptyFile(tempDir, "stream0_2.m4s")
	seg3 := createEmptyFile(tempDir, "stream0_3.m4s")
	seg5 := createEmptyFile(tempDir, "stream0_5.m4s")
	createEmptyFile(tempDir, "stream1_6.m4s")
	createEmptyFile(tempDir, "other_file.mpd")

	s := TranscodingSession{outputDir: tempDir}

	availableSegments, _ := s.AvailableSegments("video")
	fmt.Println(availableSegments)
	assert.Equal(t, map[int]string{0: seg1, 1: seg2, 2: seg3, 4: seg5}, availableSegments)
}

func createEmptyFile(path string, name string) string {
	f, _ := os.OpenFile(filepath.Join(path, name), os.O_RDONLY|os.O_CREATE, 0666)
	f.Close()
	return filepath.Join(path, name)
}
