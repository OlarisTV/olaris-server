package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// NextEpisodesQueryArgs contains the arguments that can be passed into the
// nextEpisodes query
type NextEpisodesQueryArgs struct {
	Uuid  string
	Limit *int32
}

// NextEpisodes returns the next n episodes after the one identified by the
// provided UUID. If no limit is provided, it will return all of them.
func (r *Resolver) NextEpisodes(_ context.Context, args *NextEpisodesQueryArgs) ([]*EpisodeResolver, error) {
	episodes, err := db.GetNextEpisodes(args.Uuid, args.Limit)

	episodeResolvers := make([]*EpisodeResolver, len(episodes))
	for i, episode := range episodes {
		episodeResolvers[i] = &EpisodeResolver{r: episode}
	}

	return episodeResolvers, err
}
