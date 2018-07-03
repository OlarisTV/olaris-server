package db

import (
	"github.com/jinzhu/gorm"
)

type PlayState struct {
	gorm.Model
	UUIDable
	UserID    uint
	Finished  bool
	Playtime  float64
	OwnerID   uint
	OwnerType string
}
type LatestEpResult struct {
	EpisodeId  int
	SeasonId   int
	SeriesId   int
	EpisodeNum int
	SeasonNum  int
	Playtime   int
	Finished   bool
	Height     int
}

func UpNextMovies(userID uint) (movies []*Movie) {
	env.Db.Joins("JOIN play_states on play_states.owner_id = movies.id").Where("play_states.finished = 0 and play_states.owner_type = 'movies'").Find(&movies)
	return movies
}

func UpNextEpisodes(userID uint) []*Episode {
	result := []LatestEpResult{}
	eps := []*Episode{}
	env.Db.Raw("select episodes.id as episode_id, episodes.episode_num as episode_num, episodes.season_num, seasons.id as season_id, play_states.playtime, series.id as series_id, play_states.finished, (episodes.season_num*100)+episodes.episode_num as height from play_states"+
		" inner join episodes ON episodes.ID = play_states.owner_id AND play_states.owner_type = 'episodes'"+
		" inner join seasons on seasons.id = episodes.season_id"+
		" inner join series on series.id = seasons.series_id"+
		" where play_states.user_id = ?"+
		" GROUP BY series.id"+
		" ORDER BY height DESC", userID).Scan(&result)
	// TODO(Maran): I'm not 100% the order_by height here is being used 'before' the grouping, if not then we might not always pick the latest episode
	for _, r := range result {
		if r.Finished == false {
			ep := Episode{}
			env.Db.Where("ID = ?", r.EpisodeId).First(&ep)
			eps = append(eps, &ep)
		} else {
			result := LatestEpResult{}
			env.Db.Raw("select episodes.id as episode_id, series.id as series_id"+
				" from episodes"+
				" join series ON series.id = seasons.series_id"+
				" join seasons on seasons.id = episodes.season_id"+
				" where season_num >= ? AND episode_num > ?  AND series_id = ?"+
				" order by season_num ASC, episode_num ASC LIMIT 1", r.SeasonNum, r.EpisodeNum, r.SeriesId).Scan(&result)
			ep := Episode{}
			env.Db.Where("ID = ?", result.EpisodeId).First(&ep)
			eps = append(eps, &ep)
		}
	}
	return eps
}

func LatestPlayStates(limit uint, userID uint) []PlayState {
	var pss []PlayState
	env.Db.Order("updated_at DESC").Where("user_id = ?", userID).Limit(limit).Find(&pss)
	return pss
}

func CreatePlayState(userID uint, uuid string, finished bool, playtime float64) bool {
	var ps PlayState
	env.Db.FirstOrInit(&ps, PlayState{UUIDable: UUIDable{uuid}, UserID: userID})
	ps.Finished = finished
	ps.Playtime = playtime
	env.Db.Save(&ps)

	count := 0
	var movie Movie
	var episode Episode

	env.Db.Where("uuid = ?", uuid).Find(&movie).Count(&count)
	if count > 0 {
		movie.PlayState = ps
		env.Db.Save(&movie)
		return true
	}

	count = 0
	env.Db.Where("uuid = ?", uuid).Find(&episode).Count(&count)
	if count > 0 {
		episode.PlayState = ps
		env.Db.Save(&episode)
		return true
	}

	return false
}
