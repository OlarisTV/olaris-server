package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
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
	userID := GetUserID(ctx)
	var l []*SearchItemResolver

	for _, movie := range db.SearchMovieByTitle(userID, args.Name) {
		l = append(l, &SearchItemResolver{r: &MovieResolver{r: movie}})
	}
	for _, serie := range db.SearchSeriesByTitle(userID, args.Name) {
		s := CreateSeriesResolver(serie, userID)
		l = append(l, &SearchItemResolver{r: s})
	}

	return &l
}
