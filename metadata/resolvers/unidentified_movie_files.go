package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type unidentifiedMovieFilesArgs struct {
	Offset *int32
	Limit  *int32
}

// UnidentifiedMovieFiles return unidentified movie files
func (r *Resolver) UnidentifiedMovieFiles(
	ctx context.Context,
	args *unidentifiedMovieFilesArgs) []*MovieFileResolver {

	movieFiles, err := db.FindAllUnidentifiedMovieFiles(
		buildDatabaseQueryDetails(args.Offset, args.Limit))
	if err != nil {
		return []*MovieFileResolver{}
	}

	var res []*MovieFileResolver
	for _, movieFile := range movieFiles {
		res = append(res, &MovieFileResolver{r: movieFile})
	}
	return res
}
