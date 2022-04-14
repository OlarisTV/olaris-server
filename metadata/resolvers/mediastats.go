package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// MediaStats returns some stats about the media in your server
func (r *Resolver) MediaStats(ctx context.Context) *MediaStatsResolver {
	return &MediaStatsResolver{}
}

type MediaStatsResolver struct{}

func (r *MediaStatsResolver) MovieCount() (int32, error) {
	return db.GetMovieCount()
}

func (r *MediaStatsResolver) SeriesCount() (int32, error) {
	return db.GetSeriesCount()
}

func (r *MediaStatsResolver) SeasonCount() (int32, error) {
	return db.GetSeasonCount()
}

func (r *MediaStatsResolver) EpisodeCount() (int32, error) {
	return db.GetEpisodeCount()
}
