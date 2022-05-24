package db

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"math/big"
	"time"
)

// These are copies of the same structs in the ffmpeg package to a) avoid a dependency of the db
// package on the ffmpeg package and b) allow the ffmpeg package to be advanced separately
// from the database.
type StreamKey struct {
	FileLocator filesystem.FileLocator
	// StreamId from ffmpeg
	// StreamId is always 0 for transmuxing
	StreamId int64
}

type Stream struct {
	gorm.Model
	UUIDable
	OwnerID   uint
	OwnerType string

	StreamKey

	TotalDuration time.Duration

	TimeBase         *big.Rat
	TotalDurationDts int64
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

// UpdateAllStreams updates all streams for all mediaItems
func UpdateAllStreams() {
	for _, movie := range FindAllMovieFiles() {
		UpdateStreams(movie.UUID)
	}
	for _, ep := range FindAllEpisodeFiles() {
		UpdateStreams(ep.UUID)
	}
}

// UpdateStreams deletes stream information and rescans the file
func UpdateStreams(mediaUUID string) bool {
	log.WithFields(log.Fields{"UUID": mediaUUID}).Infoln("Updating Stream information.")
	count := 0
	var movieFile MovieFile
	var episodeFile EpisodeFile

	db.Where("uuid = ?", mediaUUID).Find(&movieFile).Count(&count)
	if count > 0 {
		log.WithFields(log.Fields{"UUID": mediaUUID}).Infoln("Found movie, probing file.")
		db.Exec("DELETE FROM streams WHERE owner_id = ? AND owner_type = 'movie_files'", movieFile.ID)
		//movieFile.Streams = CollectStreams("rclone://" + movieFile.FilePath)
		db.Save(&movieFile)
		return true
	}

	count = 0
	db.Where("uuid = ?", mediaUUID).Find(&episodeFile).Count(&count)
	if count > 0 {
		log.WithFields(log.Fields{"UUID": mediaUUID}).Infoln("Found series probing file.")
		db.Exec("DELETE FROM streams WHERE owner_id = ? AND owner_type = 'episode_files'", episodeFile.ID)
		//episodeFile.Streams = CollectStreams(episodeFile.FilePath)
		db.Save(&episodeFile)
		return true
	}
	return false
}

// CreateStream persists a stream object in the database.
func CreateStream(stream *Stream) {
	db.Create(&stream)
}

// FindStreamsForEpisodeFileUUID returns all the streams for the given episode file
func FindStreamsForEpisodeFileUUID(uuid string) (streams []Stream, err error) {
	q := db.Model(&Stream{}).
		Joins("INNER JOIN episode_files ON streams.owner_id = episode_files.id").
		Where("episode_files.uuid = ?", uuid)

	err = q.Find(&streams).Error
	return
}
