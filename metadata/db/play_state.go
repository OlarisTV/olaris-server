package db

import (
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// PlayState holds status information about media files, it keeps track of progress and whether or not the content has been viewed
// to completion.
type PlayState struct {
	gorm.Model
	UUIDable
	UserID    uint
	Finished  bool
	Playtime  float64
	OwnerID   uint
	OwnerType string
}

// latestEpResult holds information about the episode that is up next for the given user.
type latestEpResult struct {
	EpisodeID  int
	SeasonID   int
	SeriesID   int
	EpisodeNum int
	SeasonNum  int
	Playtime   int
	Finished   bool
	Height     int
}

// UpNextMovies returns a list of movies that are recently added and not watched yet.
func UpNextMovies(userID uint) (movies []*Movie) {
	db.Select("movies.*, play_states.*").Order("play_states.updated_at DESC").Joins("JOIN play_states ON play_states.owner_id = movies.id").Where("play_states.finished = 0 AND play_states.owner_type = 'movies'").Where("play_states.user_id = ?", userID).Preload("PlayState").Find(&movies)
	for i := range movies {
		db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
	}
	return movies
}

// UpNextEpisodes returns a list of episodes that are up for viewing next. If you recently finished episode 5 of series Y and episode 6 is unwatched it should return this episode.
func UpNextEpisodes(userID uint) []*Episode {
	result := []latestEpResult{}
	eps := []*Episode{}
	db.Raw("SELECT episodes.id AS episode_id, episodes.episode_num AS episode_num, episodes.season_num, seasons.id AS season_id, play_states.playtime, series.id AS series_id, play_states.finished, max((episodes.season_num*100)+episodes.episode_num) AS height FROM play_states"+
		" INNER JOIN episodes ON episodes.ID = play_states.owner_id AND play_states.owner_type = 'episodes'"+
		" INNER JOIN seasons ON seasons.id = episodes.season_id"+
		" INNER join series ON series.id = seasons.series_id"+
		" WHERE play_states.user_id = ?"+
		" GROUP BY series.id"+
		" ORDER BY height DESC", userID).Scan(&result)
	// I'm not 100% the order_by height here is being used 'before' the grouping, if not then we might not always pick the latest episode
	for _, r := range result {
		if r.Finished == false {
			ep := Episode{}
			db.Where("ID = ?", r.EpisodeID).First(&ep)
			eps = append(eps, &ep)
		} else {
			result := latestEpResult{}
			db.Raw("SELECT episodes.id AS episode_id, series.id AS series_id"+
				" FROM episodes"+
				" JOIN series ON series.id = seasons.series_id"+
				" JOIN seasons ON seasons.id = episodes.season_id"+
				" WHERE season_num = ? AND episode_num > ? AND series_id = ?"+
				" ORDER BY season_num ASC, episode_num ASC LIMIT 1", r.SeasonNum, r.EpisodeNum, r.SeriesID).Scan(&result)
			ep := Episode{}
			if result.EpisodeID != 0 {
				db.Where("ID = ?", result.EpisodeID).First(&ep)
				eps = append(eps, &ep)
			} else {
				// It appears there a no more episode left in this season, let's try the next.
				db.Raw("SELECT episodes.id AS episode_id, series.id AS series_id"+
					" FROM episodes"+
					" JOIN series ON series.id = seasons.series_id"+
					" JOIN seasons ON seasons.id = episodes.season_id"+
					" WHERE season_num > ? AND episode_num > 0 AND series_id = ?"+
					" ORDER BY season_num ASC, episode_num ASC LIMIT 1", r.SeasonNum, r.EpisodeNum, r.SeriesID).Scan(&result)
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

func updatePS(contentType string, contentID uint, userID uint, finished bool, playtime float64) *PlayState {
	newPs := PlayState{OwnerID: contentID, UserID: userID, OwnerType: contentType}
	var ps PlayState
	db.FirstOrInit(&ps, newPs)

	// We set this here so we return a playstate that is reset to the react app
	ps.Finished = finished
	ps.Playtime = playtime

	// This seems to imply we marked something as 'unwatched', if that's the case just blow up the playstate all together.
	if finished == false && playtime == 0 && ps.ID != 0 {
		log.Debugln("Removing playstate")
		db.Unscoped().Delete(&ps)
	} else {
		log.WithFields(log.Fields{"type": contentType, "playtime": ps.Playtime, "finished": ps.Finished}).Debugln("Updating playstate.")
		db.Save(&ps)
	}
	return &ps
}

// CreatePlayState creates new playstate for the given user and media content.
func CreatePlayState(userID uint, mediaUUID string, finished bool, playtime float64) *PlayState {
	count := 0
	var movie Movie
	var episode Episode
	var ps PlayState
	log.WithFields(log.Fields{"mediaUUID": mediaUUID}).Debugln("Looking for media to update playstate.")

	db.Where("uuid = ?", mediaUUID).Find(&movie).Count(&count)
	if count > 0 {
		return updatePS("movies", movie.ID, userID, finished, playtime)
	}

	count = 0
	db.Where("uuid = ?", mediaUUID).Find(&episode).Count(&count)
	if count > 0 {
		return updatePS("episodes", episode.ID, userID, finished, playtime)
	}

	log.WithFields(log.Fields{"mediaUUID": mediaUUID}).Debugln("Could not find media to update.")
	return &ps

}
