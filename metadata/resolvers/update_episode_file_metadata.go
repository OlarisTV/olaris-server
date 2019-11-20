package resolvers

import (
	"context"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/db"
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

	for _, episodeFile := range episodeFiles {
		name := strings.TrimSuffix(episodeFile.FileName, filepath.Ext(episodeFile.FileName))
		parsedInfo := parsers.ParseSerieName(name)

		if parsedInfo.SeasonNum == 0 || parsedInfo.EpisodeNum == 0 {
			log.Warnln(
				"Failed to parse Episode/Season number from filename:",
				episodeFile.FileName)
			continue
		}

		// TODO(Leon Handreke): Make the handling for figure out whether the episode
		// actually exists more explicit. Right now, it's in the err clause + continue.
		episode, err := r.env.MetadataManager.GetOrCreateEpisodeByTmdbID(
			int(args.Input.TmdbID), parsedInfo.SeasonNum, parsedInfo.EpisodeNum)
		if err != nil {
			log.Warnln("Failed to create episode: ", err.Error())
			continue
		}

		// Remember previous episode ID so we can maybe garbage collect it.
		oldEpisodeID := episodeFile.EpisodeID

		episodeFile.Episode = episode
		episodeFile.EpisodeID = episode.ID
		db.SaveEpisodeFile(episodeFile)

		// Garbage collect previously associated Episode objects from DB
		oldEpisode, err := db.FindEpisodeByID(oldEpisodeID)
		if err != nil {
			return &UpdateEpisodeFileMetadataPayloadResolver{
				error: errors.Wrap(
					err,
					"Failed to refresh previously associated Episode")}
		}
		if err := r.env.MetadataManager.GarbageCollectEpisodeIfRequired(oldEpisode); err != nil {
			log.Errorln("Failed to garbage collect old episode: ", err.Error())
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
