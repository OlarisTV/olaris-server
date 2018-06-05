package db

import (
	"github.com/jinzhu/gorm"
)

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
