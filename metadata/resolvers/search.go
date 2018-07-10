package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
)

type SearchItemResolver struct {
	r interface{}
}

func (r *SearchItemResolver) ToMovie() (*MovieResolver, bool) {
	res, ok := r.r.(*MovieResolver)
	return res, ok
}
func (r *SearchItemResolver) ToSeries() (*SeriesResolver, bool) {
	res, ok := r.r.(*SeriesResolver)
	return res, ok
}

type SearchArgs struct {
	Name string
}

func (r *Resolver) Search(ctx context.Context, args *SearchArgs) *[]*SearchItemResolver {
	userID := helpers.GetUserID(ctx)
	var l []*SearchItemResolver

	for _, movie := range db.SearchMovieByTitle(userID, args.Name) {
		l = append(l, &SearchItemResolver{r: &MovieResolver{r: movie}})
	}
	for _, serie := range db.SearchSeriesByTitle(userID, args.Name) {
		l = append(l, &SearchItemResolver{r: &SeriesResolver{r: Series{Series: serie}}})
	}

	return &l
}
