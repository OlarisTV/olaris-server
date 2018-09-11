package ffmpeg

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type StreamKey struct {
	MediaFileURL string
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId int64
}

type Stream struct {
	StreamKey

	TotalDuration time.Duration

	TimeBase         int64
	TotalDurationDts DtsTimestamp
	// codecs string ready for DASH/HLS serving
	Codecs    string
	CodecName string
	Profile   string
	BitRate   int64

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

func GetAudioStreams(mediaFileURL string) ([]Stream, error) {
	streams := []Stream{}
	container, err := Probe(mediaFileURL)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "audio" {
			continue
		}
		bitrate, _ := strconv.Atoi(stream.BitRate)

		timeBase, err := parseTimeBaseString(stream.TimeBase)
		if err != nil {
			return []Stream{}, err
		}

		totalDurationTs := DtsTimestamp(stream.DurationTs)
		if stream.DurationTs == 0 {
			totalDurationTs = DtsTimestamp(container.Format.DurationSeconds * float64(timeBase))
		}

		streams = append(streams,
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
	}

	return streams, nil
}

func GetVideoStream(mediaFilePath string) (Stream, error) {
	streams, err := GetVideoStreams(mediaFilePath)
	if err != nil {
		return Stream{}, err
	}
	// TODO(Leon Handreke): Figure out something better to do here - does this ever happen?
	if len(streams) != 1 {
		return Stream{}, fmt.Errorf("File %s does not contain exactly one video stream", mediaFilePath)
	}
	return streams[0], nil

}

func GetVideoStreams(mediaFilePath string) ([]Stream, error) {
	streams := []Stream{}
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "video" {
			continue
		}
		bitrate, _ := strconv.Atoi(stream.BitRate)

		timeBase, err := parseTimeBaseString(stream.TimeBase)
		if err != nil {
			return []Stream{}, err
		}

		totalDurationTs := DtsTimestamp(stream.DurationTs)
		if stream.DurationTs == 0 {
			totalDurationTs = DtsTimestamp(container.Format.DurationSeconds * float64(timeBase))
		}

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					MediaFileURL: mediaFilePath,
					StreamId:     int64(stream.Index),
				},
				Codecs:           stream.GetMime(),
				BitRate:          int64(bitrate),
				Width:            stream.Width,
				Height:           stream.Height,
				TotalDuration:    container.Format.Duration(),
				TotalDurationDts: totalDurationTs,
				TimeBase:         timeBase,
				StreamType:       stream.CodecType,
				CodecName:        stream.CodecName,
				Profile:          stream.Profile,
			})
	}

	return streams, nil
}

func GetSubtitleStreams(mediaFileURL string) ([]Stream, error) {
	streams := []Stream{}
	container, err := Probe(mediaFileURL)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "subtitle" {
			continue
		}

		totalDuration := container.Format.Duration()
		// TODO(Leon Handreke): This usually happens for next-to-the-file .srt files, ffprobe doesn't return
		// a duration for them. Do something more intelligent (such as actually parsing the file).
		if totalDuration == 0 {
			totalDuration = time.Duration(time.Second * 100000)
		}

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					MediaFileURL: mediaFileURL,
					StreamId:     int64(stream.Index),
				},
				TotalDuration:    totalDuration,
				TotalDurationDts: DtsTimestamp(totalDuration.Seconds() * 1000),
				TimeBase:         1000,
				StreamType:       "subtitle",
				Language:         GetLanguageTag(stream),
				Title:            GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault: stream.Disposition["default"] != 0,
			})
	}

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
					TotalDuration:    container.Format.Duration(),
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
	videoStreams, _ := GetVideoStreams(streamKey.MediaFileURL)
	audioStreams, _ := GetAudioStreams(streamKey.MediaFileURL)
	subtitleStreams, _ := GetSubtitleStreams(streamKey.MediaFileURL)

	streams := append(videoStreams, append(audioStreams, subtitleStreams...)...)
	for _, s := range streams {
		if s.StreamKey == streamKey {
			return s, nil
		}
	}
	return Stream{}, fmt.Errorf("Could not find stream %s", streamKey.MediaFileURL)
}
