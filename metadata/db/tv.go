package db

import (
	"context"
	"github.com/jinzhu/gorm"
)

type BaseItem struct {
	UUIDable
	Overview     string
	BackdropPath string
	PosterPath   string
}

type TvSeries struct {
	BaseItem
	gorm.Model
	Name         string
	FirstAirDate string
	FirstAirYear uint64
	OriginalName string
	Status       string
	TmdbID       int
	Type         string
}

type TvSeason struct {
	BaseItem
	gorm.Model
	Name         string
	AirDate      string
	SeasonNumber int
	TvSeries     *TvSeries
	TvEpisodes   []*TvEpisode
	TvSeriesID   uint
	TmdbID       int
}

type TvEpisode struct {
	gorm.Model
	BaseItem
	Name         string
	SeasonNum    string
	EpisodeNum   string
	TvSeasonID   uint
	TmdbID       int
	AirDate      string
	StillPath    string
	TvSeason     *TvSeason
	EpisodeFiles []EpisodeFile
	PlayState    PlayState `gorm:"polymorphic:Playstate;"`
}

type EpisodeFile struct {
	MediaItem
	TvEpisodeID uint
	TvEpisode   *TvEpisode
}

func LoadEpisodes(episodes []*TvEpisode) {
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
	for i, _ := range episodes {
		// TODO(Maran): DRY THIS SHIT UP
		ctx.Db.Model(episodes[i]).Association("EpisodeFiles").Find(&episodes[i].EpisodeFiles)
		ctx.Db.Where("uuid = ?", episodes[i].UUID).Find(&episodes[i].PlayState)
	}
	return episodes
}
func FindEpisodesInLibrary(ct context.Context, libraryID uint) (episodes []TvEpisode) {
	ctx.Db.Where("library_id =?", libraryID).Find(&episodes)
	for i, _ := range episodes {
		// TODO(Maran): DRY THIS SHIT UP
		ctx.Db.Model(episodes[i]).Association("EpisodeFiles").Find(&episodes[i].EpisodeFiles)
		ctx.Db.Where("uuid = ?", episodes[i].UUID).Find(&episodes[i].PlayState)
	}
	return episodes
}
func FindSeriesByUUID(uuid *string) (series []TvSeries) {
	ctx.Db.Where("uuid = ?", uuid).Find(&series)
	return series
}
func FindSeasonByUUID(uuid *string) (season TvSeason) {
	ctx.Db.Where("uuid = ?", uuid).Find(&season)
	return season
}
func FindEpisodeByUUID(uuid *string) (episode TvEpisode) {
	ctx.Db.Where("uuid = ?", uuid).Find(&episode)
	ctx.Db.Where("uuid = ?", uuid).Find(&episode.PlayState)
	// TODO(Maran): DRY THIS SHIT UP
	ctx.Db.Model(&episode).Association("EpisodeFiles").Find(&episode.EpisodeFiles)
	return episode
}
