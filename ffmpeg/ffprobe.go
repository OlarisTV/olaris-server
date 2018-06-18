package ffmpeg

import (
	"bufio"
	_ "bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var extraDataRegex = regexp.MustCompile(`0{8}: \d{2}(.{2})\s(.{4})`)

// TODO(Leon Handreke): Really hacky way to just cache stdout in memory
var probeCache = map[string][]byte{}

type ProbeContainer struct {
	Streams []ProbeStream `json:"streams"`
	Format  ProbeFormat   `json:"format"`
}

type ProbeStream struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	CodecTag      string            `json:"codec_tag"`
	Profile       string            `json:"profile"`
	Level         int               `json:"level"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	CodecType     string            `json:"codec_type"`
	BitRate       string            `json:"bit_rate"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Extradata     string            `json:"extradata"`
	Tags          map[string]string `json:"tags"`
	Disposition   map[string]int    `json:"disposition"`
}

func FilterProbeStreamByCodecType(streams []ProbeStream, codecType string) []ProbeStream {
	res := []ProbeStream{}
	for _, s := range streams {
		if s.CodecType == codecType {
			res = append(res, s)
		}
	}
	return res
}

func (self *ProbeStream) GetMime() string {
	if self.CodecName == "h264" {
		res := extraDataRegex.FindAllStringSubmatch(self.Extradata, -1)
		if len(res) > 0 && len(res[0]) > 0 {
			return fmt.Sprintf("avc1.%s%s", res[0][1], res[0][2])
		}
	}
	// TODO(Leon Handreke): I think this belongs somewhere else, codec vs container...
	if self.CodecName == "aac" {
		return fmt.Sprintf("mp4a.40.2")
	}
	return self.CodecName
}

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

func Probe(fileURL string) (*ProbeContainer, error) {
	cmdOut, inCache := probeCache[fileURL]

	if !inCache {
		cmd := exec.Command("ffprobe",
			"-show_data",
			"-show_format",
			"-show_streams", fileURL, "-print_format", "json", "-v", "quiet")
		cmd.Stderr = os.Stderr

		r, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}

		err = cmd.Start()
		if err != nil {
			return nil, err
		}

		cmdOut, err = ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		probeCache[fileURL] = cmdOut

		err = cmd.Wait()
	}

	var v ProbeContainer
	err := json.Unmarshal(cmdOut, &v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// ProbeKeyframes scans for keyframes in a file and returns a list of timestamps at which keyframes were found.
func ProbeKeyframes(fileURL string) ([]time.Duration, error) {
	cmd := exec.Command("ffprobe",
		"-select_streams", "v",
		// Use dts_time here because ffmpeg seeking works by DTS,
		// see http://www.mjbshaw.com/2012/04/seeking-in-ffmpeg-know-your-timestamp.html
		"-show_entries", "packet=dts_time,flags",
		"-v", "quiet",
		"-of", "csv",
		fileURL)
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
		// Sometimes packets have no timestamp - whatever they are, we don't care about them.
		if line[1] == "N/A" {
			continue
		}
		if line[2][0] == 'K' {
			pts, err := strconv.ParseFloat(line[1], 64)
			if err != nil {
				return nil, err
			}
			keyframes = append(keyframes, time.Duration(pts*float64(time.Second)))
		}
	}

	cmd.Wait()
	return keyframes, nil
}
