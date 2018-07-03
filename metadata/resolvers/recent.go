package resolvers

import (
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"sort"
)

type NextUpResolver RecentlyAddedResolver

type RecentlyAddedResolver struct {
	r interface{}
}
type sortable interface {
	TimeStamp() int64
}

type ByCreationDate []sortable

func (a ByCreationDate) Len() int           { return len(a) }
func (a ByCreationDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCreationDate) Less(i, j int) bool { return a[i].TimeStamp() > a[j].TimeStamp() }

func (r *Resolver) RecentlyAdded() *[]*RecentlyAddedResolver {
	sortables := []sortable{}

	for _, movie := range db.RecentlyAddedMovies() {
		sortables = append(sortables, movie)
	}

	for _, ep := range db.RecentlyAddedEpisodes() {
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

func (r *RecentlyAddedResolver) ToMovie() (*MovieResolver, bool) {
	res, ok := r.r.(*MovieResolver)
	return res, ok
}

func (r *RecentlyAddedResolver) ToEpisode() (*EpisodeResolver, bool) {
	res, ok := r.r.(*EpisodeResolver)
	return res, ok
}
