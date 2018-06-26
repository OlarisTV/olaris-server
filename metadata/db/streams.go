package db

import (
	"github.com/jinzhu/gorm"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
)

type Stream struct {
	ffmpeg.Stream
	gorm.Model
	OwnerID   uint
	OwnerType string
}

// TODO-NOW! UUIDs!
func CollectStreams(filePath string) []Stream {
	videoStream, _ := ffmpeg.GetVideoStream(filePath)
	audioStreams, _ := ffmpeg.GetAudioStreams(filePath)
	subs, _ := ffmpeg.GetSubtitleStreams(filePath)

	var streams []Stream

	streams = append(streams, Stream{Stream: videoStream})

	for _, s := range audioStreams {
		streams = append(streams, Stream{Stream: s})
	}

	for _, s := range subs {
		streams = append(streams, Stream{Stream: s})
	}
	return streams
}
