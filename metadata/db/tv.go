package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// BaseItem holds information that is shared between various mediatypes.
type BaseItem struct {
	UUIDable
	Overview     string
	BackdropPath string
	PosterPath   string
}

// Series holds metadata information about series.
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

// Season holds metadata information about seasons.
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

// Episode holds metadata information about episodes.
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

// TimeStamp returns a unix timestamp for the given episode.
func (ep *Episode) TimeStamp() int64 {
	return ep.CreatedAt.Unix()
}

// EpisodeFile holds filesystem information about a given episode file.
type EpisodeFile struct {
	gorm.Model
	MediaItem
	EpisodeID uint
	Episode   *Episode
	Streams   []Stream `gorm:"polymorphic:Owner;"`
}

// IsSingleFile returns true if this is the only file for the given episode.
func (file *EpisodeFile) IsSingleFile() bool {
	count := 0
	env.Db.Model(&EpisodeFile{}).Where("episode_id = ?", file.EpisodeID).Count(&count)
	if count <= 1 {
		return true
	}
	return false
}

// DeleteSelfAndMD deletes the episode file and any stale metadata information that might have resulted.
func (file *EpisodeFile) DeleteSelfAndMD() {
	// Delete all stream information
	env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'episode_files'", &file.ID)

	var episode Episode
	env.Db.First(&episode, file.EpisodeID)

	if file.IsSingleFile() {
		// Delete all PlayState information
		env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'episode_files'", file.EpisodeID)

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
	env.Db.Delete(&file)

}

// CollectEpisodeData collects all relevant information for the given episode such as streams and playstates.
func CollectEpisodeData(episodes []Episode, userID uint) {
	for i := range episodes {
		env.Db.Model(episodes[i]).Preload("Streams").Association("EpisodeFiles").Find(&episodes[i].EpisodeFiles)
		env.Db.Model(episodes[i]).Where("user_id = ? AND owner_id = ? and owner_type =?", userID, episodes[i].ID, "tv_episodes").First(&episodes[i].PlayState)
	}
}

// FindAllSeries retrieves all identified series from the db.
func FindAllSeries() (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("tmdb_id != 0").Find(&series)
	return series
}

// SearchSeriesByTitle searches for series based on their name.
func SearchSeriesByTitle(userID uint, name string) (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("name LIKE ?", "%"+name+"%").Find(&series)
	return series
}

// FindSeriesByUUID retrives a serie based on it's UUID.
func FindSeriesByUUID(uuid *string) (series []Series) {
	env.Db.Preload("Seasons.Episodes.EpisodeFiles.Streams").Where("uuid = ?", uuid).Find(&series)
	return series
}

// FindSeasonsForSeries retrieves all season for the given series based on it's UUID.
func FindSeasonsForSeries(seriesID uint) (seasons []Season) {
	env.Db.Preload("Episodes.EpisodeFiles.Streams").Where("series_id = ?", seriesID).Find(&seasons)
	return seasons
}

// FindEpisodesForSeason finds all episodes for the given season UUID.
func FindEpisodesForSeason(seasonID uint, userID uint) (episodes []Episode) {
	env.Db.Preload("EpisodeFiles.Streams").Where("season_id = ?", seasonID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}

// FindEpisodesInLibrary returns all episodes in the given library.
func FindEpisodesInLibrary(libraryID uint, userID uint) (episodes []Episode) {
	env.Db.Where("library_id =?", libraryID).Find(&episodes)
	CollectEpisodeData(episodes, userID)

	return episodes
}

// FindSeasonByUUID finds the season based on it's UUID.
func FindSeasonByUUID(uuid *string) (season Season) {
	env.Db.Where("uuid = ?", uuid).Find(&season)
	return season
}

// FindEpisodeByUUID finds a episode based on it's UUID.
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

// FindAllEpisodeFiles retrieves all episodefiles from the db.
func FindAllEpisodeFiles() (files []EpisodeFile) {
	env.Db.Find(&files)

	return files
}

// DeleteEpisodesFromLibrary deletes all episodes from the given library.
func DeleteEpisodesFromLibrary(libraryID uint) {
	files := []EpisodeFile{}
	env.Db.Where("library_id = ?", libraryID).Find(&files)
	for _, file := range files {
		file.DeleteSelfAndMD()
	}
}
