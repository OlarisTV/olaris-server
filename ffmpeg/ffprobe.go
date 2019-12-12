package ffmpeg

import (
	_ "bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg/executable"
	"gitlab.com/olaris/olaris-server/filesystem"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var extraDataRegex = regexp.MustCompile(`0{8}: \d{2}(.{2})\s(.{4})`)

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
	TimeBase      string            `json:"time_base"`
	DurationTs    int               `json:"duration_ts"`
	RFrameRate    string            `json:"r_frame_rate"`
}

func (ps *ProbeStream) String() string {
	return fmt.Sprintf("Stream %v (%s)\nCodec: %s (%s)\nResolution: %vx%v\nBitrate: %v\n", ps.Index, ps.CodecType, ps.CodecName, ps.CodecLongName, ps.Width, ps.Height, ps.BitRate)
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

type ProbeData struct {
	Format *ProbeFormat `json:"format,omitempty"`
}

// TODO: limit size of cache, possibly user-configurable?
var probeCache = map[filesystem.FileLocator]ProbeContainer{}
var probeMutex = &sync.Mutex{}

// Probe analyzes the given file, attempting to parse the container's metadata
// and discover the number and types of streams contained within. Metadata from probed
// files is cached in a global probeCache, with access restricted by the probeMutex.
func Probe(fileLocator filesystem.FileLocator) (*ProbeContainer, error) {
	log.WithFields(log.Fields{"fileLocator": fileLocator}).Debugln("Probing file")

	// TODO: add per-file mutex lock to try to avoid double-scanning things
	probeMutex.Lock()
	probedFile, inCache := probeCache[fileLocator]
	probeMutex.Unlock()

	if inCache {
		return &probedFile, nil
	}

	log.WithFields(log.Fields{"fileLocator": fileLocator}).
		Debugln("File not in cache, probing it.")
	ffmpegUrl := buildFfmpegUrlFromFileLocator(fileLocator)
	cmd := exec.Command(
		executable.GetFFprobeExecutablePath(),
		"-show_data",
		"-show_format",
		"-show_streams", ffmpegUrl, "-print_format", "json", "-v", "quiet")
	cmd.Stderr = os.Stderr

	log.Infof("Starting %s with args %s", cmd.Path, cmd.Args)

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	cmdOut, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	var v ProbeContainer
	err = json.Unmarshal(cmdOut, &v)
	if err != nil {
		return nil, err
	}

	if len(v.Streams) == 0 {
		return nil, fmt.Errorf("no streams found, is this an actual media file")
	}

	probeMutex.Lock()
	probeCache[fileLocator] = v
	probeMutex.Unlock()

	return &v, nil
}

func parseRational(r string) (*big.Rat, error) {
	var a, b int64
	var err error

	components := strings.Split(r, "/")
	if len(components) != 2 {
		goto fail
	}
	a, err = strconv.ParseInt(components[0], 10, 64)
	if err != nil {
		goto fail
	}
	b, err = strconv.ParseInt(components[1], 10, 64)
	if err != nil {
		goto fail
	}
	return big.NewRat(a, b), nil

fail:
	return &big.Rat{}, fmt.Errorf("%s does not look like a timebase", r)

}
