package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
	"sort"
)

func (r *Resolver) UpNext(ctx context.Context) *[]*MediaItemResolver {
	userID := helpers.GetUserID(ctx)
	sortables := []sortable{}

	for _, movie := range db.UpNextMovies(userID) {
		sortables = append(sortables, movie)
	}

	for _, ep := range db.UpNextEpisodes(userID) {
		sortables = append(sortables, ep)

	}
	sort.Sort(ByCreationDate(sortables))

	l := []*MediaItemResolver{}

	for _, item := range sortables {
		if res, ok := item.(*db.Episode); ok {
			l = append(l, &MediaItemResolver{r: &EpisodeResolver{r: *res}})
		}
		if res, ok := item.(*db.Movie); ok {
			l = append(l, &MediaItemResolver{r: &MovieResolver{r: *res}})
		}
	}

	return &l
}
