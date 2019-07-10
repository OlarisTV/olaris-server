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

	// Currently, we'll only use this call to retag MovieFiles without a movie,
	// but implementing it as an update can't hurt either.
	movie, err := db.FindMovieForMovieFile(movieFile)
	// TODO(Leon Handreke): We don't pipe through the DB error here, maybe this is not a
	// RecordNotFound after all. For now, we just ignore errors here.
	if err != nil {
		movie = &db.Movie{MovieFiles: []db.MovieFile{*movieFile}}
	}

	tmdbAgent := r.env.MetadataRetrievalAgent
	if err := tmdbAgent.UpdateMovieMetadataFromTmdbID(movie, int(args.Input.TmdbID)); err != nil {
		return &UpdateMovieFileMetadataPayloadResolver{error: err}
	}

	db.SaveMovie(movie)

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
