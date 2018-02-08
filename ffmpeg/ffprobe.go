package ffmpeg

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ProbeFormat struct {
	Filename         string            `json:"filename"`
	NBStreams        int               `json:"nb_streams"`
	NBPrograms       int               `json:"nb_programs"`
	FormatName       string            `json:"format_name"`
	FormatLongName   string            `json:"format_long_name"`
	StartTimeSeconds float64           `json:"start_time,string"`
	DurationSeconds  float64           `json:"duration,string"`
	Size             uint64            `json:"size,string"`
	BitRate          uint64            `json:"bit_rate,string"`
	ProbeScore       float64           `json:"probe_score"`
	Tags             map[string]string `json:"tags"`
}

func (f ProbeFormat) StartTime() time.Duration {
	return time.Duration(f.StartTimeSeconds * float64(time.Second))
}

func (f ProbeFormat) Duration() time.Duration {
	return time.Duration(f.DurationSeconds * float64(time.Second))
}

type ProbeData struct {
	Format *ProbeFormat `json:"format,omitempty"`
}

func Probe(filename string) (*ProbeData, error) {
	cmd := exec.Command("ffprobe", "-show_format", filename, "-print_format", "json", "-v", "quiet")
	cmd.Stderr = os.Stderr

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var v ProbeData
	err = json.NewDecoder(r).Decode(&v)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// ProbeKeyframes scans for keyframes in a file and returns a list of timestamps at which keyframes were found.
func ProbeKeyframes(filename string) ([]time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-select_streams", "v",
		"-show_entries", "packet=pts_time,flags",
		"-v", "quiet",
		"-of", "csv",
		filename)
	cmd.Stderr = os.Stderr

	rawReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	keyframes := []time.Duration{}
	scanner := bufio.NewScanner(rawReader)
	for scanner.Scan() {
		// Each line has the format "packet,4.223000,K_"
		line := strings.Split(scanner.Text(), ",")
		if line[2][0] == 'K' {
			pts, err := strconv.ParseFloat(line[1], 64)
			if err != nil {
				return nil, err
			}
			keyframes = append(keyframes, time.Duration(pts*float64(time.Second)))
		}
	}

	return keyframes, nil
}
