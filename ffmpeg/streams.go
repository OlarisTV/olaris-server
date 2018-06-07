package ffmpeg

import (
	"fmt"
	"strconv"
	"time"
)

type StreamKey struct {
	MediaFilePath string
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId int64
}

type Stream struct {
	StreamKey

	TotalDuration time.Duration
	// codecs string ready for DASH/HLS serving
	Codecs  string
	BitRate int64

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

func GetAudioStreams(mediaFilePath string) ([]Stream, error) {
	streams := []Stream{}
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "audio" {
			continue
		}
		bitrate, _ := strconv.Atoi(stream.BitRate)

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					MediaFilePath: mediaFilePath,
					StreamId:      int64(stream.Index),
				},
				Codecs:           stream.GetMime(),
				BitRate:          int64(bitrate),
				TotalDuration:    container.Format.Duration(),
				StreamType:       stream.CodecType,
				Language:         GetLanguageTag(stream),
				Title:            GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault: stream.Disposition["default"] != 0,
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

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					MediaFilePath: mediaFilePath,
					StreamId:      int64(stream.Index),
				},
				Codecs:        stream.GetMime(),
				BitRate:       int64(bitrate),
				Width:         stream.Width,
				Height:        stream.Height,
				TotalDuration: container.Format.Duration(),
				StreamType:    stream.CodecType,
			})
	}

	return streams, nil
}

func GetSubtitleStreams(mediaFilePath string) ([]Stream, error) {
	streams := []Stream{}
	container, err := Probe(mediaFilePath)
	if err != nil {
		return nil, err
	}

	for _, stream := range container.Streams {
		if stream.CodecType != "subtitle" {
			continue
		}

		streams = append(streams,
			Stream{
				StreamKey: StreamKey{
					MediaFilePath: mediaFilePath,
					StreamId:      int64(stream.Index),
				},
				TotalDuration:    container.Format.Duration(),
				StreamType:       "subtitle",
				Language:         GetLanguageTag(stream),
				Title:            GetTitleOrHumanizedLanguage(stream),
				EnabledByDefault: stream.Disposition["default"] != 0,
			})
	}
	return streams, nil
}

func GetStream(streamKey StreamKey) (Stream, error) {
	// TODO(Leon Handreke): Error handling
	videoStreams, _ := GetVideoStreams(streamKey.MediaFilePath)
	audioStreams, _ := GetAudioStreams(streamKey.MediaFilePath)
	subtitleStreams, _ := GetSubtitleStreams(streamKey.MediaFilePath)

	streams := append(videoStreams, append(audioStreams, subtitleStreams...)...)
	for _, s := range streams {
		if s.StreamKey == streamKey {
			return s, nil
		}
	}
	return Stream{}, fmt.Errorf("Could not find stream %s", streamKey.MediaFilePath)
}
