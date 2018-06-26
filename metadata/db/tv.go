package db

import (
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

func CollectEpisodeData(episodes []TvEpisode, userID uint) {
	for i, _ := range episodes {
		env.Db.Model(episodes[i]).Association("EpisodeFiles").Find(&episodes[i].EpisodeFiles)
		env.Db.Where("uuid = ?", episodes[i].UUID).Find(&episodes[i].PlayState)
	}
}

func FindAllSeries() (series []TvSeries) {
	env.Db.Where("tmdb_id != 0").Find(&series)
	return series
}
func FindSeasonsForSeries(seriesID uint) (seasons []TvSeason) {
	env.Db.Where("tv_series_id = ?", seriesID).Find(&seasons)
	return seasons
}
func FindEpisodesForSeason(seasonID uint, userID uint) (episodes []TvEpisode) {
	env.Db.Where("tv_season_id = ?", seasonID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}
func FindEpisodesInLibrary(libraryID uint, userID uint) (episodes []TvEpisode) {
	env.Db.Where("library_id =?", libraryID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}

func FindSeriesByUUID(uuid *string) (series []TvSeries) {
	env.Db.Where("uuid = ?", uuid).Find(&series)
	return series
}
func FindSeasonByUUID(uuid *string) (season TvSeason) {
	env.Db.Where("uuid = ?", uuid).Find(&season)
	return season
}
func FindEpisodeByUUID(uuid *string, userID uint) (episode TvEpisode) {
	env.Db.Where("uuid = ?", uuid).Find(&episode)
	env.Db.Where("uuid = ? AND user_id = ?", uuid, userID).Find(&episode.PlayState)
	env.Db.Model(&episode).Association("EpisodeFiles").Find(&episode.EpisodeFiles)
	return episode
}
