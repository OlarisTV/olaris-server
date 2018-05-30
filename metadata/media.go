package metadata

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strconv"
)

type MediaType int

const (
	MediaTypeMovie = iota
	MediaTypeSeries
	MediaTypeMusic
	MediaTypeOtherMovie
)

type MediaItem struct {
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
	gorm.Model
	BackdropPath    string
	PosterPath      string
	Name            string
	Overview        string
	FirstAirDate    string
	OriginalName    string
	Status          string
	Seasons         []*TvSeason
	SeasonResolvers []*seasonResolver
	TmdbID          int
	Type            string
}

type TvSeason struct {
	gorm.Model
	Name             string
	Overview         string
	AirDate          string
	SeasonNumber     int
	PosterPath       string
	TvSeries         *TvSeries
	TvEpisodes       []*TvEpisode
	TvSeriesID       uint
	TmdbID           int
	EpisodeResolvers []*episodeResolver
}

type TvEpisode struct {
	gorm.Model
	MediaItem
	Name       string
	SeasonNum  string
	EpisodeNum string
	TvSeasonID uint
	TmdbID     int
	AirDate    string
	StillPath  string
	TvSeason   *TvSeason
}
