package metadata

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// LibrarySubscriber is an interface for a receiver for the various events that the
// MetadataManager can emit.
type LibrarySubscriber interface {
	MovieAdded(*db.Movie)
	EpisodeAdded(*db.Episode)
	SeriesAdded(*db.Series)
	SeasonAdded(*db.Season)
}
