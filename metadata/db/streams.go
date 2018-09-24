package db

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
)

// Stream holds information about the various streams included in a mediafile. This can be audio/video or even subtitle data.
type Stream struct {
	ffmpeg.Stream
	gorm.Model
	UUIDable
	OwnerID   uint
	OwnerType string
}

// CollectStreams collects all stream information for the given file.
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

	_, err := ffmpeg.GetOrCacheKeyFrames(videoStream)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warnln("Error creating keyframe data")

	}

	return streams
}

// CreateStream persists a stream object in the database.
func CreateStream(stream *Stream) {

	db.Create(&stream)
}
