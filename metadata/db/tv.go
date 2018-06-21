package db

import (
	"github.com/jinzhu/gorm"
)

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

func FindAllSeries() (series []TvSeries) {
	ctx.Db.Where("tmdb_id != 0").Find(&series)
	return series
}
func FindSeasonsForSeries(seriesID uint) (seasons []TvSeason) {
	ctx.Db.Where("tv_series_id = ?", seriesID).Find(&seasons)
	return seasons
}
func FindEpisodesForSeason(seasonID uint) (episodes []TvEpisode) {
	ctx.Db.Where("tv_season_id = ?", seasonID).Find(&episodes)
	return episodes
}
func FindEpisodesInLibrary(libraryID uint) (episodes []TvEpisode) {
	ctx.Db.Where("library_id =?", libraryID).Find(&episodes)
	return episodes
}
func FindSeriesByUUID(uuid *string) (series []TvSeries) {
	ctx.Db.Where("uuid = ?", uuid).Find(&series)
	return series
}
