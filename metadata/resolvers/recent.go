package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"sort"
)

type sortable interface {
	TimeStamp() int64
	UpdatedAtTimeStamp() int64
}

// ByCreationDate is a sortable type to sort by creation date.
type ByCreationDate []sortable

func (a ByCreationDate) Len() int           { return len(a) }
func (a ByCreationDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCreationDate) Less(i, j int) bool { return a[i].TimeStamp() > a[j].TimeStamp() }

// ByUpdatedAt is a sortable type to sort by updated_at date.
type ByUpdatedAt []sortable

func (a ByUpdatedAt) Len() int      { return len(a) }
func (a ByUpdatedAt) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByUpdatedAt) Less(i, j int) bool {
	return a[i].UpdatedAtTimeStamp() > a[j].UpdatedAtTimeStamp()
}

// RecentlyAdded returns recently added media content.
func (r *Resolver) RecentlyAdded(ctx context.Context) *[]*MediaItemResolver {
	userID, _ := auth.UserID(ctx)
	sortables := []sortable{}

	for _, movie := range db.RecentlyAddedMovies(userID) {
		sortables = append(sortables, movie)
	}

	for _, ep := range db.RecentlyAddedEpisodes(userID) {
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
