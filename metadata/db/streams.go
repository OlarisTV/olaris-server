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
		movieFile.Streams = CollectStreams("rclone://" + movieFile.FilePath)
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
	log.WithFields(log.Fields{"filePath": filePath}).Debugln("Reading stream information from file")
	var streams []Stream

	s, err := ffmpeg.GetStreams(filePath)
	//s, err := ffmpeg.GetStreams("file://" + filePath)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Debugln("Received error while opening file for stream inspection")
		return streams
	}

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
