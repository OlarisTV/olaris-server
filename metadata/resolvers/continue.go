package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
	"sort"
)

func (r *Resolver) UpNext(ctx context.Context) *[]*RecentlyAddedResolver {
	userID := helpers.GetUserID(ctx)
	sortables := []sortable{}

	for _, movie := range db.UpNextMovies(userID) {
		sortables = append(sortables, movie)
	}

	for _, ep := range db.UpNextEpisodes(userID) {
		sortables = append(sortables, ep)

	}
	sort.Sort(ByCreationDate(sortables))

	l := []*RecentlyAddedResolver{}

	for _, item := range sortables {
		if res, ok := item.(*db.TvEpisode); ok {
			l = append(l, &RecentlyAddedResolver{r: &EpisodeResolver{r: *res}})
		}
		if res, ok := item.(*db.Movie); ok {
			l = append(l, &RecentlyAddedResolver{r: &MovieResolver{r: *res}})
		}
	}

	return &l
}

// 1. Collect latest 5 play_states
// 2. Loop through all
// 3. If finished == false, return item as 'continue play'
// 4. If finished == true and movie = false try to figure out the next episode
// 5. Next episode is episodes by series ordered by season+episode number and take the index of the last + 1
// select ((season_num * 100)+ episode_num) as height  from tv_episodes order by height DESC
/*
select tv_series.name, play_states.playtime, tv_series.id as series_id, play_states.finished, (tv_episodes.season_num*100)+tv_episodes.episode_num as height from play_states
inner join tv_episodes ON tv_episodes.ID = play_states.owner_id AND play_states.owner_type = "tv_episodes"
inner join tv_seasons on tv_seasons.id = tv_episodes.tv_season_id
inner join tv_series on tv_series.id = tv_seasons.tv_series_id
GROUP BY tv_series.id
ORDER BY height DESC
*/
