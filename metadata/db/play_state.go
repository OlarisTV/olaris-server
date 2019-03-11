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
	db.Joins("JOIN play_states ON play_states.owner_id = movies.id").Where("play_states.finished = 0 AND play_states.owner_type = 'movies'").Where("play_states.user_id = ?", userID).Find(&movies)
	for i := range movies {
		db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
	}
	return movies
}

// UpNextEpisodes returns a list of episodes that are up for viewing next. If you recently finished episode 5 of series Y and episode 6 is unwatched it should return this episode.
func UpNextEpisodes(userID uint) []*Episode {
	result := []latestEpResult{}
	eps := []*Episode{}
	db.Raw("select episodes.id as episode_id, episodes.episode_num as episode_num, episodes.season_num, seasons.id as season_id, play_states.playtime, series.id as series_id, play_states.finished, (episodes.season_num*100)+episodes.episode_num as height from play_states"+
		" inner join episodes ON episodes.ID = play_states.owner_id AND play_states.owner_type = 'episodes'"+
		" inner join seasons on seasons.id = episodes.season_id"+
		" inner join series on series.id = seasons.series_id"+
		" where play_states.user_id = ?"+
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
			db.Raw("select episodes.id as episode_id, series.id as series_id"+
				" from episodes"+
				" join series ON series.id = seasons.series_id"+
				" join seasons on seasons.id = episodes.season_id"+
				" where season_num >= ? AND episode_num > ?  AND series_id = ?"+
				" order by season_num ASC, episode_num ASC LIMIT 1", r.SeasonNum, r.EpisodeNum, r.SeriesID).Scan(&result)
			ep := Episode{}
			db.Where("ID = ?", result.EpisodeID).First(&ep)
			eps = append(eps, &ep)
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

// CreatePlayState creates new playstate for the given user and media content.
func CreatePlayState(userID uint, mediaUUID string, finished bool, playtime float64) *PlayState {
	count := 0
	var movie Movie
	var episode Episode
	var ps PlayState
	log.WithFields(log.Fields{"mediaUUID": mediaUUID}).Debugln("Looking for media to update playstate.")

	db.Where("uuid = ?", mediaUUID).Find(&movie).Count(&count)
	if count > 0 {
		db.FirstOrInit(&ps, PlayState{OwnerID: movie.ID, UserID: userID, OwnerType: "movies"})
		ps.Finished = finished
		ps.Playtime = playtime
		log.WithFields(log.Fields{"type": "movie", "playtime": ps.Playtime, "finished": ps.Finished}).Debugln("Updating playstate.")
		db.Save(&ps)

		movie.PlayState = ps
		db.Save(&movie)
		return &ps
	}

	count = 0
	db.Where("uuid = ?", mediaUUID).Find(&episode).Count(&count)
	if count > 0 {
		db.FirstOrInit(&ps, PlayState{OwnerID: episode.ID, UserID: userID, OwnerType: "episodes"})
		ps.Finished = finished
		ps.Playtime = playtime
		episode.PlayState = ps
		log.WithFields(log.Fields{"type": "episode", "playtime": ps.Playtime, "finished": ps.Finished}).Debugln("Updating playstate.")
		db.Save(&episode)
		return &ps
	}

	log.WithFields(log.Fields{"mediaUUID": mediaUUID}).Debugln("Could not find media to update.")
	return &ps

}
