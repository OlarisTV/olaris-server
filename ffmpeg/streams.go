package ffmpeg

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WARNING: These structs are cached in the database, so adding fields or changing types
// will cause mayhem. See https://gitlab.com/olaris/olaris-server/issues/55

type StreamKey struct {
	MediaFileURL string
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

func GetStreams(mediaFileURL string) (*Streams, error) {
	streams := Streams{}
	container, err := Probe(mediaFileURL)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {

		timeBase, err := parseRational(stream.TimeBase)
		if err != nil {
			return nil, err
		}
		totalDurationTs := DtsTimestamp(stream.DurationTs)
		if stream.DurationTs == 0 {
			totalDurationTs = DtsTimestamp(container.Format.DurationSeconds * float64(timeBase.Denom().Int64()))
		}

		if stream.CodecType == "audio" {
			bitrate, _ := strconv.Atoi(stream.BitRate)

			streams.AudioStreams = append(streams.AudioStreams,
				Stream{
					StreamKey: StreamKey{
						MediaFileURL: mediaFileURL,
						StreamId:     int64(stream.Index),
					},
					Codecs:           stream.GetMime(),
					BitRate:          int64(bitrate),
					TotalDuration:    container.Format.Duration(),
					TotalDurationDts: totalDurationTs,
					StreamType:       stream.CodecType,
					Language:         GetLanguageTag(stream),
					Title:            GetTitleOrHumanizedLanguage(stream),
					EnabledByDefault: stream.Disposition["default"] != 0,
					TimeBase:         timeBase,
				})
		} else if stream.CodecType == "video" {
			bitrate, _ := strconv.Atoi(stream.BitRate)

			if bitrate == 0 {
				filepath, err := mediaFileURLToFilepath(mediaFileURL)
				if err != nil {
					return nil, fmt.Errorf("Could not determine local path for file %s", mediaFileURL)
				}
				fileinfo, err := os.Stat(filepath)
				if err != nil {
					return nil, fmt.Errorf("Could not determine filesize for file %s", mediaFileURL)
				}
				filesize := fileinfo.Size()

				durationSeconds := int64(container.Format.DurationSeconds)
				if durationSeconds > 0 {
					// TODO(Leon Handreke): Is there a nicer way to do bits/bytes conversion?
					bitrate = int((filesize / durationSeconds) * 8)
				} else {
					// TODO(Leon Handreke): Figure out if there is any other way to get an
					//  approximate duration before we fall back to a "default" bitrate.
					// See https://gitlab.com/olaris/olaris-server/issues/67
					bitrate = 10000000
				}
			}
			frameRate, err := parseRational(stream.RFrameRate)
			if err != nil {
				return nil, fmt.Errorf("Could not parse r_frame_rate %s", stream.RFrameRate)
			}

			streams.VideoStreams = append(streams.VideoStreams, Stream{
				StreamKey: StreamKey{
					MediaFileURL: mediaFileURL,
					StreamId:     int64(stream.Index),
				},
				Codecs:           stream.GetMime(),
				BitRate:          int64(bitrate),
				FrameRate:        frameRate,
				Width:            stream.Width,
				Height:           stream.Height,
				TotalDuration:    container.Format.Duration(),
				TotalDurationDts: totalDurationTs,
				TimeBase:         timeBase,
				StreamType:       stream.CodecType,
				CodecName:        stream.CodecName,
				Profile:          stream.Profile,
			})
		} else if stream.CodecType == "subtitle" {
			totalDuration := container.Format.Duration()
			// TODO(Leon Handreke): This usually happens for next-to-the-file .srt files, ffprobe doesn't return
			// a duration for them. Do something more intelligent (such as actually parsing the file).
			if totalDuration == 0 {
				totalDuration = time.Duration(time.Second * 100000)
			}

			streams.SubtitleStreams = append(streams.SubtitleStreams, Stream{
				StreamKey: StreamKey{
					MediaFileURL: mediaFileURL,
					StreamId:     int64(stream.Index),
				},
				TotalDuration:    totalDuration,
				TotalDurationDts: DtsTimestamp(totalDuration.Seconds() * 1000),
				TimeBase:         big.NewRat(1, 1000),
				StreamType:       "subtitle",
				Language:         GetLanguageTag(stream),
				Title:            GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault: stream.Disposition["default"] != 0,
			})

		}
	}

	externalSubtitles, _ := buildExternalSubtitleStreams(mediaFileURL, container.Format.Duration())
	streams.SubtitleStreams = append(streams.SubtitleStreams, externalSubtitles...)

	return &streams, nil

}

func (s *Streams) GetVideoStream() Stream {
	// TODO(Leon Handreke): Figure out something better to do here - does this ever happen?
	if len(s.VideoStreams) > 1 {
		log.Infof("File %s does not contain exactly one video stream", s.VideoStreams[0].MediaFileURL)
	}
	return s.VideoStreams[0]

}

func buildExternalSubtitleStreams(mediaFileURL string, duration time.Duration) ([]Stream, error) {
	streams := []Stream{}

	mediaFilePath, err := mediaFileURLToFilepath(mediaFileURL)
	if err == nil {
		mediaFilePathWithoutExt := strings.TrimSuffix(mediaFilePath, filepath.Ext(mediaFileURL))
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
						MediaFileURL: subtitleFile,
						StreamId:     0,
					},
					TotalDuration:    duration,
					StreamType:       "subtitle",
					Language:         lang,
					Title:            tag,
					EnabledByDefault: false,
				})
		}
	}

	return streams, nil
}

func GetStream(streamKey StreamKey) (Stream, error) {
	// TODO(Leon Handreke): Error handling
	c, err := GetStreams(streamKey.MediaFileURL)
	if err != nil {
		return Stream{}, err
	}

	streams := append(c.VideoStreams, append(c.AudioStreams, c.SubtitleStreams...)...)
	for _, s := range streams {
		if s.StreamKey == streamKey {
			return s, nil
		}
	}
	return Stream{}, fmt.Errorf("Could not find stream %s", streamKey.MediaFileURL)
}
