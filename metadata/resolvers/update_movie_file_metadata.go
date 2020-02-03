package resolvers

import (
	"context"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// UpdateMovieFileMetadataInput is a request
type UpdateMovieFileMetadataInput struct {
	MovieFileUUID string
	TmdbID        int32
}

// UpdateMovieFileMetadataPayloadResolver is the payload
type UpdateMovieFileMetadataPayloadResolver struct {
	error     error
	mediaItem interface{}
}

// UpdateMovieFileMetadata handles the updateMediaItemMetadata mutation
func (r *Resolver) UpdateMovieFileMetadata(
	ctx context.Context,
	args *struct{ Input UpdateMovieFileMetadataInput },
) *UpdateMovieFileMetadataPayloadResolver {
	movieFile, err := db.FindMovieFileByUUID(args.Input.MovieFileUUID)
	if err != nil {
		return &UpdateMovieFileMetadataPayloadResolver{error: err}
	}

	oldMovie, err := db.FindMovieForMovieFile(movieFile)
	// If this MovieFile already has the correct Movie associated
	if err == nil && oldMovie.TmdbID == int(args.Input.TmdbID) {
		return &UpdateMovieFileMetadataPayloadResolver{mediaItem: oldMovie}
	}

	movie, err := r.env.MetadataManager.GetOrCreateMovieByTmdbID(int(args.Input.TmdbID))
	if err != nil {
		return &UpdateMovieFileMetadataPayloadResolver{error: err}
	}

	movieFile.Movie = *movie
	db.SaveMovieFile(movieFile)

	r.env.MetadataManager.GarbageCollectMovieIfRequired(oldMovie.ID)

	return &UpdateMovieFileMetadataPayloadResolver{mediaItem: movie}
}

// MediaItem returns the media item
func (r *UpdateMovieFileMetadataPayloadResolver) MediaItem() *MediaItemResolver {
	return &MediaItemResolver{r: r.mediaItem}
}

// Error returns error.
func (r *UpdateMovieFileMetadataPayloadResolver) Error() *ErrorResolver {
	if r.error != nil {
		return CreateErrResolver(r.error)
	}
	return nil
}
