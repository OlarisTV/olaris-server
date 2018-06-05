package resolvers

import (
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

type Library struct {
	db.Library
	Movies   []*MovieResolver
	Episodes []*EpisodeResolver
}

func (r *Resolver) Libraries() []*LibraryResolver {
	var l []*LibraryResolver
	libraries := db.AllLibraries()
	for _, library := range libraries {
		list := Library{library, nil, nil}
		var movies []db.MovieItem
		var mr []*MovieResolver
		r.ctx.Db.Where("library_id = ?", library.ID).Find(&movies)
		for _, movie := range movies {
			if movie.Title != "" {
				mov := MovieResolver{r: movie}
				mr = append(mr, &mov)
			}
		}
		list.Movies = mr

		var episodes []db.TvEpisode
		r.ctx.Db.Where("library_id =?", library.ID).Find(&episodes)
		for _, episode := range episodes {
			list.Episodes = append(list.Episodes, &EpisodeResolver{r: episode})
		}

		lib := LibraryResolver{r: list}
		l = append(l, &lib)
	}
	return l
}
