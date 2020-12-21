package db

import (
	"github.com/jinzhu/gorm"
)

// PlayState holds status information about media files, it keeps track of progress and whether or not the content has been viewed
// to completion.
type PlayState struct {
	gorm.Model
	UUIDable
	UserID    uint `gorm:"unique_index:idx_unique_play_state_per_media"`
	Finished  bool
	Playtime  float64
	MediaUUID string `gorm:"unique_index:idx_unique_play_state_per_media"`
}

// latestEpResult holds information about the episode that is up next for the given user.
type latestEpResult struct {
	EpisodeID    int
	SeasonID     int
	SeriesID     int
	EpisodeNum   int
	SeasonNumber int
	Playtime     int
	Finished     bool
	Height       int
}
type uniqueSeries struct {
	SeriesID int
}

// UpNextMovies returns a list of movies that are recently added and not watched yet.
func UpNextMovies(userID uint) (movies []*Movie) {
	db.Select("movies.*, play_states.*").
		Order("play_states.updated_at DESC").
		Joins("JOIN play_states ON play_states.media_uuid = movies.uuid").
		Where("play_states.finished = false").
		Where("play_states.user_id = ?", userID).
		Find(&movies)
	for i := range movies {
		db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
	}
	return movies
}

// UpNextEpisodes returns a list of episodes that are up for viewing next. If you recently finished episode 5 of series Y and episode 6 is unwatched it should return this episode.
func UpNextEpisodes(userID uint) []*Episode {
	result := []latestEpResult{}
	res := []uniqueSeries{}
	eps := []*Episode{}
	db.Raw("SELECT DISTINCT(seasons.series_id) FROM play_states "+
		"INNER JOIN episodes ON episodes.uuid = play_states.media_uuid "+
		"INNER JOIN seasons on seasons.id = episodes.season_id "+
		"WHERE play_states.user_id = ? "+
		"GROUP BY seasons.series_id, play_states.updated_at", userID).Scan(&res)

	for _, series := range res {
		db.Raw("SELECT seasons.series_id, seasons.id as season_id,episode_num, seasons.season_number, play_states.finished, max((seasons.season_number*100)+episodes.episode_num) as height, episodes.id as episode_id, episodes.uuid FROM play_states "+
			"INNER JOIN episodes ON episodes.uuid = play_states.media_uuid "+
			"INNER JOIN seasons on seasons.id = episodes.season_id "+
			"WHERE series_id = ? AND play_states.user_id = ? "+
			"GROUP BY seasons.series_id, episodes.id, seasons.id, play_states.finished "+
			"ORDER BY height DESC "+
			"LIMIT 1", series.SeriesID, userID).Scan(&result)

		r := result[0]

		if r.Finished == false {
			ep := Episode{}
			db.Where("ID = ?", r.EpisodeID).First(&ep)
			eps = append(eps, &ep)
		} else {
			result := latestEpResult{}
			db.Raw("SELECT episodes.id AS episode_id, series.id AS series_id, seasons.season_number"+
				" FROM episodes"+
				" JOIN seasons ON seasons.id = episodes.season_id"+
				" JOIN series ON series.id = seasons.series_id"+
				" WHERE seasons.season_number = ? AND episode_num > ? AND series_id = ?"+
				" ORDER BY season_number ASC, episode_num ASC LIMIT 1", r.SeasonNumber, r.EpisodeNum, r.SeriesID).Scan(&result)
			ep := Episode{}
			if result.EpisodeID != 0 {
				db.Where("ID = ?", result.EpisodeID).First(&ep)
				eps = append(eps, &ep)
			} else {
				// It appears there a no more episode left in this season, let's try the next.
				db.Raw("SELECT episodes.id AS episode_id, series.id AS series_id, seasons.season_number"+
					" FROM episodes"+
					" JOIN seasons ON seasons.id = episodes.season_id"+
					" JOIN series ON series.id = seasons.series_id"+
					" WHERE season_number > ? AND episode_num > 0 AND series_id = ?"+
					" ORDER BY season_number ASC, episode_num ASC LIMIT 1", r.SeasonNumber, r.SeriesID).Scan(&result)
				if result.EpisodeID != 0 {
					db.Where("ID = ?", result.EpisodeID).First(&ep)
					eps = append(eps, &ep)
				}
			}
		}
	}
	for i := range eps {
		db.Model(eps[i]).Preload("Streams").Association("EpisodeFiles").Find(&eps[i].EpisodeFiles)
	}
	return eps
}

// LatestPlayStates returns playstates for content recently played for the given user.
func LatestPlayStates(limit uint, userID uint) []PlayState {
	var pss []PlayState
	db.Order("updated_at DESC").Where("user_id = ?", userID).Limit(limit).Find(&pss)
	return pss
}

func SavePlayState(playState *PlayState) error {
	var updatedPlayState PlayState
	// Upsert the given PlayState. The WHERE clause uniquely identifies the
	// PlayState due to the UNIQUE index on media_uuid/user_id.
	return db.
		Where(&PlayState{MediaUUID: playState.MediaUUID, UserID: playState.UserID}).
		Assign(playState).
		FirstOrCreate(&updatedPlayState).
		Error
}

func DeletePlayState(mediaUUID string, userID uint) error {
	return db.Unscoped().Delete(PlayState{}, "media_uuid = ? AND user_id = ?", mediaUUID, userID).Error
}

// FindPlayState finds a playstate
func FindPlayState(mediaUUID string, userID uint) (*PlayState, error) {
	if userID == 0 {
		// We panic in this case, we currently don't have a usecase for this and doing it
		// is a serious privacy risk.
		panic("Trying to query for all PlayStates")
	}

	var playState PlayState
	if err := db.First(&playState, &PlayState{
		MediaUUID: mediaUUID,
		UserID:    userID,
	}).
		Error; err != nil {

		return nil, err
	}
	return &playState, nil
}
