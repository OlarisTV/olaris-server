package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// NearbyEpisodesQueryArgs contains the arguments that can be passed into the
// nearbyEpisodes query
type NearbyEpisodesQueryArgs struct {
	Uuid          string
	PreviousLimit int32
	NextLimit     int32
}

// NearbyEpisodes returns the next "x" episodes before and after the episode
// identified by the provided UUID.
func (r *Resolver) NearbyEpisodes(_ context.Context, args *NearbyEpisodesQueryArgs) *NearbyEpisodesResolver {
	return &NearbyEpisodesResolver{args: args}
}

// NearbyEpisodesResolver resolves the episodes directly before and after any
// given episode, identified by its UUID.
type NearbyEpisodesResolver struct {
	args *NearbyEpisodesQueryArgs
}

// Previous returns the previous n episodes before the one identified by the
// given UUID.
func (r *NearbyEpisodesResolver) Previous() ([]*EpisodeResolver, error) {
	episodes, err := db.GetPreviousEpisodes(r.args.Uuid, r.args.PreviousLimit)

	episodeResolvers := make([]*EpisodeResolver, len(episodes))
	for i, episode := range episodes {
		episodeResolvers[i] = &EpisodeResolver{r: episode}
	}

	return episodeResolvers, err
}

// Next returns the next n episodes before the one identified by the given UUID.
func (r *NearbyEpisodesResolver) Next() ([]*EpisodeResolver, error) {
	episodes, err := db.GetNextEpisodes(r.args.Uuid, r.args.NextLimit)

	episodeResolvers := make([]*EpisodeResolver, len(episodes))
	for i, episode := range episodes {
		episodeResolvers[i] = &EpisodeResolver{r: episode}
	}

	return episodeResolvers, err
}
