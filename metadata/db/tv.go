package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

type BaseItem struct {
	UUIDable
	Overview     string
	BackdropPath string
	PosterPath   string
}

type Series struct {
	BaseItem
	gorm.Model
	Name         string
	FirstAirDate string
	FirstAirYear uint64
	OriginalName string
	Status       string
	TmdbID       int
	Type         string
	Seasons      []*Season
}

type Season struct {
	BaseItem
	gorm.Model
	Name         string
	AirDate      string
	SeasonNumber int
	Series       *Series
	SeriesID     uint
	TmdbID       int
	Episodes     []*Episode
}

type Episode struct {
	gorm.Model
	BaseItem
	Name         string
	SeasonNum    int
	EpisodeNum   int
	SeasonID     uint
	TmdbID       int
	AirDate      string
	StillPath    string
	Season       *Season
	PlayState    PlayState `gorm:"polymorphic:Owner;"`
	EpisodeFiles []EpisodeFile
}

func (self *Episode) TimeStamp() int64 {
	return self.CreatedAt.Unix()
}

type EpisodeFile struct {
	gorm.Model
	MediaItem
	EpisodeID uint
	Episode   *Episode
	Streams   []Stream `gorm:"polymorphic:Owner;"`
}

func (self *EpisodeFile) IsSingleFile() bool {
	count := 0
	env.Db.Model(&EpisodeFile{}).Where("episode_id = ?", self.EpisodeID).Count(&count)
	if count <= 1 {
		return true
	} else {
		return false
	}
}

func (self *EpisodeFile) DeleteSelfAndMD() {
	// Delete all stream information
	env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'episode_files'", &self.ID)

	var episode Episode
	env.Db.First(&episode, self.EpisodeID)

	if self.IsSingleFile() {
		// Delete all PlayState information
		env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'episode_files'", self.EpisodeID)

		// Delete Episode
		env.Db.Delete(&episode)

		count := 0
		var season Season
		env.Db.First(&season, episode.SeasonID)

		env.Db.Model(Episode{}).Where("season_id = ?", season.ID).Count(&count)

		fmt.Println(count)
		// If there are no more episodes to this season, delete the season.
		if count == 0 {
			env.Db.Delete(Season{}, "id = ?", episode.SeasonID)
		}

		// If there are no more seasons to this series, delete it.
		count = 0
		env.Db.Model(Season{}).Where("series_id = ?", season.SeriesID).Count(&count)
		if count == 0 {
			env.Db.Delete(Series{}, "id = ?", season.SeriesID)
		}
	}

	// Delete all file information
	env.Db.Delete(&self)

}

func CollectEpisodeData(episodes []Episode, userID uint) {
	for i, _ := range episodes {
		env.Db.Model(episodes[i]).Preload("Streams").Association("EpisodeFiles").Find(&episodes[i].EpisodeFiles)
		env.Db.Model(episodes[i]).Where("user_id = ? AND owner_id = ? and owner_type =?", userID, episodes[i].ID, "tv_episodes").First(&episodes[i].PlayState)
	}
}

func FindAllSeries() (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("tmdb_id != 0").Find(&series)
	return series
}

func SearchSeriesByTitle(userID uint, name string) (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("name LIKE ?", "%"+name+"%").Find(&series)
	return series
}

func FindSeriesByUUID(uuid *string) (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("uuid = ?", uuid).Find(&series)
	return series
}

func FindSeasonsForSeries(seriesID uint) (seasons []Season) {
	env.Db.Preload("Episodes.EpisodeFiles.Streams").Where("series_id = ?", seriesID).Find(&seasons)
	return seasons
}

func FindEpisodesForSeason(seasonID uint, userID uint) (episodes []Episode) {
	env.Db.Preload("EpisodeFiles.Streams").Where("season_id = ?", seasonID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}

func FindEpisodesInLibrary(libraryID uint, userID uint) (episodes []Episode) {
	env.Db.Where("library_id =?", libraryID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}

func FindSeasonByUUID(uuid *string) (season Season) {
	env.Db.Where("uuid = ?", uuid).Find(&season)
	return season
}

func FindEpisodeByUUID(uuid *string, userID uint) (episode *Episode) {
	var episodes []Episode
	env.Db.Where("uuid = ?", uuid).First(&episodes)
	if len(episodes) == 1 {
		episode = &episodes[0]
		env.Db.Model(&episode).Preload("Streams").Association("EpisodeFiles").Find(&episode.EpisodeFiles)
		env.Db.Model(&episode).Where("user_id = ? AND owner_id = ? and owner_type =?", userID, episode.ID, "tv_episodes").First(&episode.PlayState)
	}
	return episode
}

func FindAllEpisodeFiles() (files []EpisodeFile) {
	env.Db.Find(&files)

	return files
}

func DeleteEpisodesFromLibrary(libraryID uint) {
	files := []EpisodeFile{}
	env.Db.Where("library_id = ?", libraryID).Find(&files)
	for _, file := range files {
		file.DeleteSelfAndMD()
	}
}
