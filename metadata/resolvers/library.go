package resolvers

import (
	"context"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
)

type Library struct {
	db.Library
	Movies   []*MovieResolver
	Episodes []*EpisodeResolver
}

func (r *Resolver) Libraries(ctx context.Context) []*LibraryResolver {
	userID := helpers.GetUserID(ctx)
	var l []*LibraryResolver
	libraries := db.AllLibraries()
	for _, library := range libraries {
		list := Library{library, nil, nil}
		var mr []*MovieResolver
		for _, movie := range db.FindMoviesInLibrary(library.ID, userID) {
			if movie.Title != "" {
				mov := MovieResolver{r: movie}
				mr = append(mr, &mov)
			}
		}
		list.Movies = mr

		for _, episode := range db.FindEpisodesInLibrary(library.ID, userID) {
			list.Episodes = append(list.Episodes, &EpisodeResolver{r: episode})
		}

		lib := LibraryResolver{r: list}
		l = append(l, &lib)
	}
	return l
}
