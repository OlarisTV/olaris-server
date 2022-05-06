package resolvers

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-rename/identify"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// UpdateEpisodeFileMetadataInput is a request
type UpdateEpisodeFileMetadataInput struct {
	EpisodeFileUUID *[]*string
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
	var err error
	err = ifAdmin(ctx)
	if err != nil {
		return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
	}

	var episodeFiles []*db.EpisodeFile
	var episodeFileUUIDs []*string

	if args.Input.EpisodeFileUUID != nil {
		episodeFileUUIDs = *args.Input.EpisodeFileUUID
	}

	if len(episodeFileUUIDs) > 0 {
		for _, e := range episodeFileUUIDs {
			episodeFile, err := db.FindEpisodeFileByUUID(*e)
			if err != nil {
				return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
			}
			episodeFiles = append(episodeFiles, episodeFile)
		}
	} else if args.Input.SeriesUUID != nil {
		log.Debugln("Received SeriesUUID")
		episodeFiles, err = findEpisodeFilesForSeries(*args.Input.SeriesUUID)
		if err != nil {
			return &UpdateEpisodeFileMetadataPayloadResolver{error: err}
		}
	} else {
		return &UpdateEpisodeFileMetadataPayloadResolver{
			error: errors.New("Neither EpisodeFile nor Series UUID given"),
		}
	}

	updateEpisodeFileMetadataGroup := sync.WaitGroup{}
	updateEpisodeFileMetadataGroup.Add(len(episodeFiles))

	for _, v := range episodeFiles {
		go func(episodeFile *db.EpisodeFile) {
			defer updateEpisodeFileMetadataGroup.Done()
			opts := identify.GetDefaultOptions()
			opts.ForceSeries = true

			parsedInfo := identify.NewParsedFile(episodeFile.FileName, opts)

			if parsedInfo.SeasonNum() == 0 || parsedInfo.EpisodeNum() == 0 {
				log.Warnln(
					"Failed to parse Episode/Season number from filename:",
					episodeFile.FileName)
				return
			}

			// TODO(Leon Handreke): Make the handling for figuring out whether the episode
			// actually exists more explicit. Right now, it's in the err clause + continue.
			episode, err := r.env.MetadataManager.GetOrCreateEpisodeByTmdbID(
				int(args.Input.TmdbID), parsedInfo.SeasonNum(), parsedInfo.EpisodeNum())
			if err != nil {
				log.Warnln("Failed to create episode: ", err.Error())
				return
			}

			// Remember previous episode ID so we can maybe garbage collect it.
			oldEpisodeID := episodeFile.EpisodeID

			episodeFile.Episode = episode
			episodeFile.EpisodeID = episode.ID
			db.SaveEpisodeFile(episodeFile)

			go r.env.MetadataManager.GarbageCollectEpisodeIfRequired(oldEpisodeID)
		}(v)
	}
	// TODO(Leon Handreke): Have at least a spinner, better proper progress reporting.
	//  See https://gitlab.com/olaris/olaris-react/issues/33
	// updateEpisodeFileMetadataGroup.Wait()

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
