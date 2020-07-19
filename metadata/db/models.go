package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Defines various mediatypes, only Movie and Series support atm.
const (
	MediaTypeMovie = iota
	MediaTypeSeries
	MediaTypeOtherMovie
)

// MediaType describes the type of media in a library.
type MediaType int

// UUIDable ensures a UUID is added to each model this is embedded in.
type UUIDable struct {
	UUID string `json:"uuid"`
}

// BeforeCreate ensures a UUID is set before model creation.
func (ud *UUIDable) BeforeCreate(tx *gorm.DB) (err error) {
	ud.SetUUID()
	return
}

// SetUUID creates a new v4 UUID.
func (ud *UUIDable) SetUUID() error {
	uuid, err := uuid.NewV4()

	if err != nil {
		return err
	}
	ud.UUID = uuid.String()
	return nil
}

// GetUUID returns the model's UUID.
func (ud *UUIDable) GetUUID() string {
	return ud.UUID
}

// MediaFile is an interface for various methods can be done on both episodes and movies
type MediaFile interface {
	GetFilePath() string
	GetFileName() string
	GetLibrary() *Library
	GetStreams() []Stream
}

// MediaItem is an embeddeable struct that holds information about filesystem files (episode or movies).
type MediaItem struct {
	UUIDable
	FileName  string
	FilePath  string
	Size      int64
	Library   Library
	LibraryID uint
}

// FindContentByUUID can retrieve episode or movie data based on a UUID.
func FindContentByUUID(uuid string) MediaFile {
	count := 0
	var movie MovieFile
	var episode EpisodeFile

	db.Where("uuid = ?", uuid).Preload("Streams").Preload("Library").Find(&movie).Count(&count)
	if count > 0 {
		return movie
	}

	count = 0
	db.Where("uuid = ?", uuid).Preload("Streams").Preload("Library").Find(&episode).Count(&count)
	if count > 0 {
		return episode
	}

	return nil
}

// RecentlyAddedMovies returns a list of the latest 10 movies added to the database.
func RecentlyAddedMovies(userID uint) (movies []*Movie) {
	db.Select("movies.*,play_states.*").Preload("MovieFiles.Streams").Joins("LEFT JOIN play_states ON play_states.media_uuid = movies.uuid").Where("play_states.user_id = ? OR play_states.user_id IS NULL", userID).Where("tmdb_id != 0").Order("movies.created_at DESC").Limit(10).Find(&movies)
	return movies
}

// RecentlyAddedEpisodes returns a list of the latest 10 episodes added to the database.
func RecentlyAddedEpisodes(userID uint) (eps []*Episode) {
	db.Select("episodes.*, play_states.*").Preload("EpisodeFiles.Streams").Joins("LEFT JOIN play_states ON play_states.media_uuid = episodes.uuid").Where("play_states.user_id = ? OR play_states.user_id IS NULL", userID).Where("tmdb_id != 0").Order("episodes.created_at DESC").Limit(10).Find(&eps)
	return eps
}
