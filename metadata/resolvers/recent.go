package resolvers

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
	"sort"
)

// MediaItemResolver is a resolver around media types.
type MediaItemResolver struct {
	r interface{}
}

// ToMovie tries to convert media to Movie
func (r *MediaItemResolver) ToMovie() (*MovieResolver, bool) {
	res, ok := r.r.(*MovieResolver)
	return res, ok
}

// ToEpisode tries to convert media to Episode
func (r *MediaItemResolver) ToEpisode() (*EpisodeResolver, bool) {
	res, ok := r.r.(*EpisodeResolver)
	return res, ok
}

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

func (a ByUpdatedAt) Len() int           { return len(a) }
func (a ByUpdatedAt) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByUpdatedAt) Less(i, j int) bool { return a[i].UpdatedAtTimeStamp() > a[j].UpdatedAtTimeStamp() }

// RecentlyAdded returns recently added media content.
func (r *Resolver) RecentlyAdded() *[]*MediaItemResolver {
	sortables := []sortable{}

	for _, movie := range db.RecentlyAddedMovies() {
		sortables = append(sortables, movie)
	}

	for _, ep := range db.RecentlyAddedEpisodes() {
		sortables = append(sortables, ep)

	}
	sort.Sort(ByCreationDate(sortables))

	l := []*MediaItemResolver{}

	for _, item := range sortables {
		if res, ok := item.(*db.Episode); ok {
			l = append(l, &MediaItemResolver{r: &EpisodeResolver{r: newEpisode(res, 0)}})
		}
		if res, ok := item.(*db.Movie); ok {
			l = append(l, &MediaItemResolver{r: &MovieResolver{r: *res}})
		}
	}

	return &l
}
