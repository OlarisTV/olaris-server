package managers

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// LibrarySubscriber is an interface that can implement the various notifications the libraryManager can give off
type LibrarySubscriber interface {
	MovieAdded(*db.Movie)
	EpisodeAdded(*db.Episode)
	SeriesAdded(*db.Series)
	SeasonAdded(*db.Season)
}
