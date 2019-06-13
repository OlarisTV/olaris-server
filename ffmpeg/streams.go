package ffmpeg

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"math/big"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const TotalDurationInvalid = float64(-1)

// WARNING: These structs are cached in the database, so adding fields or changing types
// will cause mayhem. See https://gitlab.com/olaris/olaris-server/issues/55

type StreamKey struct {
	FileLocator filesystem.FileLocator
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId int64
}

type Stream struct {
	StreamKey

	TotalDuration time.Duration

	TimeBase         *big.Rat
	TotalDurationDts DtsTimestamp
	// codecs string ready for DASH/HLS serving
	Codecs    string
	CodecName string
	Profile   string
	BitRate   int64
	FrameRate *big.Rat

	Width  int
	Height int

	// "audio", "video", "subtitle"
	StreamType string
	// Only relevant for audio and subtitles. Language code.
	Language string
	// User-visible string for this audio or subtitle track
	Title            string
	EnabledByDefault bool
}

type Streams struct {
	VideoStreams    []Stream
	AudioStreams    []Stream
	SubtitleStreams []Stream
}

func GetStreams(fileLocator filesystem.FileLocator) (*Streams, error) {
	streams := Streams{}

	container, err := Probe(fileLocator)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to probe with ffmpeg")
	}

	totalDurationSeconds := TotalDurationInvalid
	if container.Format.DurationSeconds > 0 {
		totalDurationSeconds = container.Format.DurationSeconds
	}

	for _, stream := range container.Streams {

		timeBase, err := parseRational(stream.TimeBase)
		if err != nil {
			return nil, err
		}

		totalDurationTs := DtsTimestampInvalid
		if stream.DurationTs > 0 {
			totalDurationTs = DtsTimestamp(stream.DurationTs)
		} else if totalDurationSeconds != TotalDurationInvalid {
			totalDurationTs = DtsTimestamp(container.Format.DurationSeconds * float64(timeBase.Denom().Int64()))
		}

		if stream.CodecType == "audio" {
			if totalDurationSeconds == TotalDurationInvalid {
				return nil, errors.New("Failed to probe file duration")
			}

			bitrate, _ := strconv.Atoi(stream.BitRate)

			streams.AudioStreams = append(streams.AudioStreams,
				Stream{
					StreamKey: StreamKey{
						FileLocator: fileLocator,
						StreamId:    int64(stream.Index),
					},
					Codecs:           stream.GetMime(),
					BitRate:          int64(bitrate),
					TotalDuration:    time.Duration(totalDurationSeconds * float64(time.Second)),
					TotalDurationDts: totalDurationTs,
					StreamType:       stream.CodecType,
					Language:         GetLanguageTag(stream),
					Title:            GetTitleOrHumanizedLanguage(stream),
					EnabledByDefault: stream.Disposition["default"] != 0,
					TimeBase:         timeBase,
				})
		} else if stream.CodecType == "video" {
			if totalDurationSeconds == TotalDurationInvalid {
				return nil, errors.New("Failed to probe file duration")
			}

			bitrate, _ := strconv.Atoi(stream.BitRate)

			if bitrate == 0 {
				node, err := filesystem.GetNodeFromFileLocator(fileLocator)
				if err != nil {
					return nil, errors.Wrap(err, "Failed to get filesystem node")
				}

				filesize := node.Size()
				// TODO(Leon Handreke): Is there a nicer way to do bits/bytes conversion?
				bitrate = int((filesize / int64(totalDurationSeconds)) * 8)
			}
			frameRate, err := parseRational(stream.RFrameRate)
			if err != nil {
				return nil, fmt.Errorf("Could not parse r_frame_rate %s", stream.RFrameRate)
			}

			streams.VideoStreams = append(streams.VideoStreams, Stream{
				StreamKey: StreamKey{
					FileLocator: fileLocator,
					StreamId:    int64(stream.Index),
				},
				Codecs:           stream.GetMime(),
				BitRate:          int64(bitrate),
				FrameRate:        frameRate,
				Width:            stream.Width,
				Height:           stream.Height,
				TotalDuration:    time.Duration(totalDurationSeconds * float64(time.Second)),
				TotalDurationDts: totalDurationTs,
				TimeBase:         timeBase,
				StreamType:       stream.CodecType,
				CodecName:        stream.CodecName,
				Profile:          stream.Profile,
			})
		} else if stream.CodecType == "subtitle" {
			// TODO(Leon Handreke): This usually happens for next-to-the-file .srt files, ffprobe doesn't return
			// a duration for them. Do something more intelligent (such as actually parsing the file).
			if totalDurationSeconds == TotalDurationInvalid {
				totalDurationSeconds = float64(time.Second * 100000)
			}

			streams.SubtitleStreams = append(streams.SubtitleStreams, Stream{
				StreamKey: StreamKey{
					FileLocator: fileLocator,
					StreamId:    int64(stream.Index),
				},
				TotalDuration:    time.Duration(totalDurationSeconds * float64(time.Second)),
				TotalDurationDts: DtsTimestamp(totalDurationSeconds * 1000),
				TimeBase:         big.NewRat(1, 1000),
				StreamType:       "subtitle",
				Language:         GetLanguageTag(stream),
				Title:            GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault: stream.Disposition["default"] != 0,
			})

		}
	}

	externalSubtitles, _ := buildExternalSubtitleStreams(
		fileLocator, time.Duration(totalDurationSeconds*float64(time.Second)))
	streams.SubtitleStreams = append(streams.SubtitleStreams, externalSubtitles...)

	return &streams, nil

}

func (s *Streams) GetVideoStream() Stream {
	// TODO(Leon Handreke): Figure out something better to do here - does this ever happen?
	if len(s.VideoStreams) > 1 {
		log.Infof(
			"File %s does not contain exactly one video stream",
			s.VideoStreams[0].FileLocator)
	}
	return s.VideoStreams[0]

}

func buildExternalSubtitleStreams(
	fileLocator filesystem.FileLocator,
	duration time.Duration) ([]Stream, error) {

	streams := []Stream{}

	// TODO(Leon Handreke): We should be able to support other backends, but it will be another
	// point where the abstraction will be a bit leaky
	if fileLocator.Backend != filesystem.BackendLocal {
		return []Stream{}, nil
	}

	mediaFilePath := fileLocator.Path
	mediaFilePathWithoutExt := strings.TrimSuffix(mediaFilePath, filepath.Ext(mediaFilePath))
	r := regexp.MustCompile(regexp.QuoteMeta(mediaFilePathWithoutExt) + "\\.?(?P<lang>.*).srt")

	subtitleFiles, _ := filepath.Glob(mediaFilePathWithoutExt + "*.srt")
	for _, subtitleFile := range subtitleFiles {
		match := r.FindStringSubmatch(subtitleFile)
		// TODO(Leon Handreke): This is a case of aggressive programming, can this ever fail?
		tag := match[1]
		lang := "unk"

		if tag == "" {
			tag = "External"
		} else {
			if humanizedToLangTag[tag] != "" {
				lang = humanizedToLangTag[tag]
			}
		}

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					FileLocator: filesystem.FileLocator{
						Backend: fileLocator.Backend,
						Path:    subtitleFile,
					},
					StreamId: 0,
				},
				TotalDuration:    duration,
				StreamType:       "subtitle",
				Language:         lang,
				Title:            tag,
				EnabledByDefault: false,
			})
	}

	return streams, nil
}

func GetStream(streamKey StreamKey) (Stream, error) {
	// TODO(Leon Handreke): Error handling
	c, err := GetStreams(streamKey.FileLocator)
	if err != nil {
		return Stream{}, err
	}

	streams := append(c.VideoStreams, append(c.AudioStreams, c.SubtitleStreams...)...)
	for _, s := range streams {
		if s.StreamKey == streamKey {
			return s, nil
		}
	}
	return Stream{}, fmt.Errorf("Could not find stream %s", streamKey.FileLocator)
}
