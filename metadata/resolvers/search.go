package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/auth"
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
func (r *Resolver) Search(ctx context.Context, args *searchArgs) *[]*SearchItemResolver {
	userID, _ := auth.UserID(ctx)
	var l []*SearchItemResolver

	for _, movie := range db.SearchMovieByTitle(userID, args.Name) {
		l = append(l, &SearchItemResolver{r: &MovieResolver{r: movie}})
	}
	for _, serie := range db.SearchSeriesByTitle(userID, args.Name) {
		l = append(l, &SearchItemResolver{r: &SeriesResolver{newSeries(&serie, userID)}})
	}

	return &l
}
