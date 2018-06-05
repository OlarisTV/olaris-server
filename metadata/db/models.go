package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"strconv"
)

type MediaType int
type UUIDable struct {
	UUID string
}

func (self *UUIDable) SetUUID() error {
	uuid, err := uuid.NewV4()

	if err != nil {
		fmt.Println("Could not generate unique UID", err)
		return err
	}
	self.UUID = uuid.String()
	return nil
}

func (self *UUIDable) GetUUID() string {
	return self.UUID
}

type MediaItem struct {
	UUIDable
	Title        string
	Year         uint64
	FileName     string
	FilePath     string
	BackdropPath string
	PosterPath   string
	Size         int64
	Overview     string
	Library      Library
	LibraryID    uint
}

func (self *MediaItem) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}

type MovieItem struct {
	gorm.Model
	MediaItem
	TmdbID        int
	ReleaseDate   string
	OriginalTitle string
	ImdbID        string
}

func (self *MovieItem) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}

type TvSeries struct {
	UUIDable
	gorm.Model
	BackdropPath string
	PosterPath   string
	Name         string
	Overview     string
	FirstAirDate string
	FirstAirYear uint64
	OriginalName string
	Status       string
	TmdbID       int
	Type         string
}

type TvSeason struct {
	UUIDable
	gorm.Model
	Name         string
	Overview     string
	AirDate      string
	SeasonNumber int
	PosterPath   string
	TvSeries     *TvSeries
	TvEpisodes   []*TvEpisode
	TvSeriesID   uint
	TmdbID       int
}
