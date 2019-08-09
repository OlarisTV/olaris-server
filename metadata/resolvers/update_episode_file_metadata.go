package resolvers

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"path/filepath"
	"strings"
)

// UpdateEpisodeFileMetadataInput is a request
type UpdateEpisodeFileMetadataInput struct {
	EpisodeFileUUID *string
	SeriesUUID      *string
	TmdbID          int32
}

// UpdateEpisodeFileMetadataPayloadResolver is the payload
type UpdateEpisodeFileMetadataPayloadResolver struct {
	error error
}

// UpdateEpisodeFileMetadata handles the updateMediaItemMetadata mutation
func (r *Resolver) UpdateEpisodeFileMetadata(
	ctx context.Context,
	args *struct {
		Input UpdateEpisodeFileMetadataInput
	},
) *UpdateEpisodeFileMetadataPayloadResolver {

	var episodeFiles []*db.EpisodeFile
	var err error
	if args.Input.EpisodeFileUUID != nil {
		episodeFile, err := db.FindEpisodeFileByUUID(*args.Input.EpisodeFileUUID)
		if err != nil {
			return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
		}
		episodeFiles = append(episodeFiles, episodeFile)
	} else if args.Input.SeriesUUID != nil {
		episodeFiles, err = findEpisodeFilesForSeries(*args.Input.SeriesUUID)
		if err != nil {
			return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
		}
	} else {
		return &UpdateEpisodeFileMetadataPayloadResolver{
			error: errors.New("Neither EpisodeFile nor Series UUID given"),
		}
	}

	var oldEpisodes []*db.Episode
	for _, episodeFile := range episodeFiles {
		e, _ := db.FindEpisodeByID(episodeFile.EpisodeID)
		oldEpisodes = append(oldEpisodes, e)
	}

	tmdbAgent := r.env.MetadataRetrievalAgent

	for _, episodeFile := range episodeFiles {
		name := strings.TrimSuffix(episodeFile.FileName, filepath.Ext(episodeFile.FileName))
		parsedInfo := parsers.ParseSerieName(name)

		if parsedInfo.SeasonNum == 0 || parsedInfo.EpisodeNum == 0 {
			return &UpdateEpisodeFileMetadataPayloadResolver{
				error: fmt.Errorf(
					"Failed to parse Episode/Season number from filename %s",
					episodeFile.FileName)}

		}

		episode, err := managers.GetOrCreateEpisodeByTmdbID(
			int(args.Input.TmdbID), parsedInfo.SeasonNum, parsedInfo.EpisodeNum,
			tmdbAgent,
			nil, // TODO(Leon Handreke): How do we get the subscriber here.
		)
		if err != nil {
			return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
		}

		episodeFile.Episode = episode
		episodeFile.EpisodeID = episode.ID
		db.SaveEpisodeFile(episodeFile)

		// Garbage collect previously associated Episode objects from DB
		for _, oldEpisode := range oldEpisodes {
			// Refresh the episode with the updates above
			oldEpisode, err = db.FindEpisodeByID(oldEpisode.ID)
			if err != nil {
				return &UpdateEpisodeFileMetadataPayloadResolver{
					error: errors.Wrap(
						err,
						"Failed to refresh previously associated Episode")}
			}
			managers.GarbageCollectEpisodeIfRequired(oldEpisode)

		}
	}

	return &UpdateEpisodeFileMetadataPayloadResolver{}
}

func findEpisodeFilesForSeries(uuid string) ([]*db.EpisodeFile, error) {
	series, err := db.FindSeriesByUUID(uuid)
	if err != nil {
		errors.Wrap(err, "Failed to find EpisodeFiles for series")
	}

	var episodeFiles []*db.EpisodeFile
	for _, season := range series.Seasons {
		for _, episode := range season.Episodes {
			for _, episodeFile := range episode.EpisodeFiles {
				episodeFiles = append(episodeFiles, &episodeFile)
			}
		}
	}
	return episodeFiles, nil
}

// Error returns error.
func (r *UpdateEpisodeFileMetadataPayloadResolver) Error() *ErrorResolver {
	if r.error != nil {
		return CreateErrResolver(r.error)
	}
	return nil
}
