package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"strconv"
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
		fmt.Println("Could not generate unique UID", err)
		return err
	}
	ud.UUID = uuid.String()
	return nil
}

// GetUUID returns the model's UUID.
func (ud *UUIDable) GetUUID() string {
	return ud.UUID
}

// MediaItem is an embeddeable struct that holds information about filesystem files (episode or movies).
type MediaItem struct {
	UUIDable
	Title     string
	Year      uint64
	FileName  string
	FilePath  string
	Size      int64
	Library   Library
	LibraryID uint
}

// YearAsString converts the year to string (no surprise there huh.)
func (mi *MediaItem) YearAsString() string {
	return strconv.FormatUint(mi.Year, 10)
}

// MediaResult is a struct that can either contain a movie or episode file.
type MediaResult struct {
	Movie   *MovieFile
	Episode *EpisodeFile
}

// FindContentByUUID can retrieve episode or movie data based on a UUID.
func FindContentByUUID(uuid string) *MediaResult {
	count := 0
	var movie MovieFile
	var episode EpisodeFile

	env.Db.Where("uuid = ?", uuid).Find(&movie).Count(&count)
	if count > 0 {
		return &MediaResult{Movie: &movie}
	}

	count = 0
	env.Db.Where("uuid = ?", uuid).Find(&episode).Count(&count)
	if count > 0 {
		return &MediaResult{Episode: &episode}
	}

	return &MediaResult{}
}

// RecentlyAddedMovies returns a list of the latest 10 movies added to the database.
func RecentlyAddedMovies() (movies []*Movie) {
	env.Db.Where("tmdb_id != 0").Order("created_at DESC").Limit(10).Find(&movies)
	return movies
}

// RecentlyAddedEpisodes returns a list of the latest 10 episodes added to the database.
func RecentlyAddedEpisodes() (eps []*Episode) {
	env.Db.Where("tmdb_id != 0").Order("created_at DESC").Limit(10).Find(&eps)
	return eps
}
