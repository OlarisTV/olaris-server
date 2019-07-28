package resolvers

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// SearchItemResolver wrapper arounds search items.
type SearchItemResolver struct {
	r interface{}
}

// ToMovie parses content to a movie.
func (r *SearchItemResolver) ToMovie() (*MovieResolver, bool) {
	res, ok := r.r.(*MovieResolver)
	return res, ok
}

// ToSeries parses content to a serie.
func (r *SearchItemResolver) ToSeries() (*SeriesResolver, bool) {
	res, ok := r.r.(*SeriesResolver)
	return res, ok
}

type searchArgs struct {
	Name string
}

// Search searches for media content.
func (r *Resolver) Search(args *searchArgs) *[]*SearchItemResolver {
	var l []*SearchItemResolver

	for _, movie := range db.SearchMovieByTitle(args.Name) {
		l = append(l, &SearchItemResolver{r: &MovieResolver{r: movie}})
	}
	for _, serie := range db.SearchSeriesByTitle(args.Name) {
		l = append(l, &SearchItemResolver{r: &SeriesResolver{serie}})
	}

	return &l
}
