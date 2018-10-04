package db

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"sync"
)

// Stream holds information about the various streams included in a mediafile. This can be audio/video or even subtitle data.
type Stream struct {
	ffmpeg.Stream
	gorm.Model
	UUIDable
	OwnerID   uint
	OwnerType string
}

var mutex = &sync.Mutex{}
var collectingKeyframes bool

// CollectStreamKeyFrames indexes all keyframes for accurate seeking
func CollectStreamKeyFrames() {
	log.Infoln("Starting keyframe cache generation.")
	if collectingKeyframes == false {
		mutex.Lock()
		collectingKeyframes = true
		mutex.Unlock()
		var streams []Stream
		db.Where("stream_type = 'video' OR stream_type = 'audio'").Find(&streams)
		for _, stream := range streams {
			_, err := ffmpeg.GetOrCacheKeyFrames(ffmpeg.Stream{StreamKey: ffmpeg.StreamKey{StreamId: stream.StreamId, MediaFileURL: stream.MediaFileURL}})
			if err != nil {
				log.WithFields(log.Fields{"error": err, "file": stream.MediaFileURL}).Warnln("Error creating keyframe data")
			}
		}
		mutex.Lock()
		collectingKeyframes = false
		mutex.Unlock()
	} else {
		log.Warnln("Already generating keyframe cache, skipping")
	}
	log.Infoln("Finished keyframe cache generation.")
}

// UpdateAllStreams updates all streams for all mediaItems
func UpdateAllStreams() {
	for _, movie := range FindAllMovieFiles() {
		UpdateStreams(&movie.UUID)
	}
	for _, ep := range FindAllEpisodeFiles() {
		UpdateStreams(&ep.UUID)
	}
}

// UpdateStreams deletes stream information and rescans the file
func UpdateStreams(mediaUUID *string) bool {
	log.WithFields(log.Fields{"UUID": *mediaUUID}).Infoln("Updating Stream information.")
	count := 0
	var movieFile MovieFile
	var episodeFile EpisodeFile

	db.Where("uuid = ?", mediaUUID).Find(&movieFile).Count(&count)
	if count > 0 {
		log.WithFields(log.Fields{"UUID": *mediaUUID}).Infoln("Found movie, probing file.")
		db.Exec("DELETE FROM streams WHERE owner_id = ? AND owner_type = 'movie_files'", movieFile.ID)
		movieFile.Streams = CollectStreams(movieFile.FilePath)
		db.Save(&movieFile)
		return true
	}

	count = 0
	db.Where("uuid = ?", mediaUUID).Find(&episodeFile).Count(&count)
	if count > 0 {
		log.WithFields(log.Fields{"UUID": *mediaUUID}).Infoln("Found series probing file.")
		db.Exec("DELETE FROM streams WHERE owner_id = ? AND owner_type = 'episode_files'", episodeFile.ID)
		episodeFile.Streams = CollectStreams(episodeFile.FilePath)
		db.Save(&episodeFile)
		return true
	}
	return false
}

// CollectStreams collects all stream information for the given file.
func CollectStreams(filePath string) []Stream {
	s, _ := ffmpeg.GetStreams(filePath)

	var streams []Stream

	streams = append(streams, Stream{Stream: s.GetVideoStream()})

	for _, s := range s.AudioStreams {
		streams = append(streams, Stream{Stream: s})
	}

	for _, s := range s.SubtitleStreams {
		streams = append(streams, Stream{Stream: s})
	}

	return streams
}

// CreateStream persists a stream object in the database.
func CreateStream(stream *Stream) {

	db.Create(&stream)
}
