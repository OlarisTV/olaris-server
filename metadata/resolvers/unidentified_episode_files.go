package resolvers

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
)

type unidentifiedEpisodeFilesArgs struct {
	Offset *int32
	Limit  *int32
}

// UnidentifiedEpisodeFiles returns unidentified episode files
func (r *Resolver) UnidentifiedEpisodeFiles(
	args *unidentifiedEpisodeFilesArgs) []*EpisodeFileResolver {

	qd := buildDatabaseQueryDetails(args.Offset, args.Limit)
	episodeFiles, err := db.FindAllUnidentifiedEpisodeFiles(&qd)
	if err != nil {
		return []*EpisodeFileResolver{}
	}

	var res []*EpisodeFileResolver
	for _, episodeFile := range episodeFiles {
		res = append(res, &EpisodeFileResolver{r: episodeFile})
	}
	return res
}
