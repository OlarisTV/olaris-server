package metadata

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/helpers/levenshtein"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
	"math"
	"path/filepath"
	"strings"
	"sync"
)

type TmdbEpisodeKey struct {
	TmdbSeriesID  int
	SeasonNumber  int
	EpisodeNumber int
}

const xattrNameTVSeriesTMDBID = "user.olaris.v1.tv.tmdb.series.id"
const xattrNameTVSeasonNumber = "user.olaris.v1.tv.tmdb.season.number"
const xattrNameTVEpisodeNumber = "user.olaris.v1.tv.tmdb.episode.number"

func (m *MetadataManager) getEpisodeLock(episodeID uint) *sync.RWMutex {
	v, _ := m.episodeLock.LoadOrStore(episodeID, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

func (m *MetadataManager) getSeasonLock(seasonID uint) *sync.RWMutex {
	v, _ := m.seasonLock.LoadOrStore(seasonID, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

func (m *MetadataManager) getSeriesLock(seriesID uint) *sync.RWMutex {
	v, _ := m.seriesLock.LoadOrStore(seriesID, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

// ForceSeriesMetadataUpdate refreshes all data from the agent and updates the database record.
// TODO(Leon Handreke): Use a pool here and make it explicit in the documentation and function
//  name that we're only queueing these updates.
func (m *MetadataManager) ForceSeriesMetadataUpdate() {
	series, err := db.FindAllSeries(nil)
	if err != nil {

		log.WithField("error", err.Error()).
			Error("Failed to get series for forced metadata update")
	}
	for _, series := range series {
		m.UpdateSeriesMD(series)
		for _, season := range db.FindSeasonsForSeries(series.ID) {
			m.UpdateSeasonMD(&season)
			for _, episode := range db.FindEpisodesForSeason(season.ID) {
				m.UpdateEpisodeMD(&episode)
			}
		}
	}
}

// UpdateSeriesMD loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (m *MetadataManager) UpdateSeriesMD(series *db.Series) error {
	log.WithFields(log.Fields{"name": series.Name}).
		Println("Refreshing metadata for series.")
	m.agent.UpdateSeriesMD(series, series.TmdbID)
	db.SaveSeries(series)
	return nil
}

// UpdateEpisodeMD updates the database record with the latest data from the agent
func (m *MetadataManager) UpdateEpisodeMD(ep *db.Episode) error {
	if err := m.agent.UpdateEpisodeMD(ep,
		ep.GetSeries().TmdbID, ep.GetSeason().SeasonNumber, ep.EpisodeNum); err != nil {
		return err
	}
	if err := db.SaveEpisode(ep); err != nil {
		return err
	}

	return nil
}

// UpdateSeasonMD updates the database record with the latest data from the agent
func (m *MetadataManager) UpdateSeasonMD(season *db.Season) error {
	if err := m.agent.UpdateSeasonMD(
		season, season.GetSeries().TmdbID, season.SeasonNumber); err != nil {
		return err
	}
	if err := db.SaveSeason(season); err != nil {
		return err
	}
	return nil
}

// Attempt to parse a filename and determine the three values
// that uniquely identify the episode (on TMDB)
func (m *MetadataManager) getEpisodeKeyFromFilename(
	episodeFile *db.EpisodeFile) (*TmdbEpisodeKey, error) {

	name := strings.TrimSuffix(episodeFile.FilePath, filepath.Ext(episodeFile.FileName))
	parsedInfo := parsers.ParseSeriesName(name)

	// Find a series for this Episode
	var options = make(map[string]string)
	if parsedInfo.Year != "" {
		options["first_air_date_year"] = parsedInfo.Year
	}
	searchRes, err := m.agent.TmdbSearchTv(parsedInfo.Title, options)
	if err != nil {
		return nil, err
	}
	if len(searchRes.Results) == 0 {
		log.WithFields(log.Fields{
			"title": parsedInfo.Title,
			"year":  parsedInfo.Year,
		}).Warnln("Could not find match based on parsed title and given year.")

		return nil, errors.New("Could not find match in TMDB ID for given filename")
	}

	var bestDistance = math.MaxInt32
	// We use the index here because the type is really long.
	var bestResultIdx int
	for i, r := range searchRes.Results {
		d := levenshtein.ComputeDistance(parsedInfo.Title, r.Name)
		if d < bestDistance {
			bestDistance = d
			bestResultIdx = i
		}
	}
	seriesInfo := searchRes.Results[bestResultIdx]

	return &TmdbEpisodeKey{TmdbSeriesID: seriesInfo.ID, SeasonNumber: parsedInfo.SeasonNum, EpisodeNumber: parsedInfo.EpisodeNum}, nil

}

// Attempt to read the season/episode information from the file's xattrs
// The bool return value indicates whether xattr information was present on the file
func (m *MetadataManager) getEpisodeKeyFromXattr(
	episodeFile *db.EpisodeFile) (*TmdbEpisodeKey, bool, error) {

	p, err := filesystem.ParseFileLocator(episodeFile.GetFilePath())
	if err != nil {
		return nil, false, err
	}

	xattrNames := []string{xattrNameTVSeriesTMDBID, xattrNameTVSeasonNumber, xattrNameTVEpisodeNumber}
	xattrTmdbIDs, err := helpers.GetXattrInts(p.Path, xattrNames)
	if err != nil {
		log.Debugln("No Xattr data found for ", p.Path, err)
		return &TmdbEpisodeKey{}, false, nil
	}

	return &TmdbEpisodeKey{
		TmdbSeriesID:  xattrTmdbIDs[xattrNameTVSeriesTMDBID],
		SeasonNumber:  xattrTmdbIDs[xattrNameTVSeasonNumber],
		EpisodeNumber: xattrTmdbIDs[xattrNameTVEpisodeNumber],
	}, true, nil
}

func (m *MetadataManager) getEpisodeKey(episodeFile *db.EpisodeFile) (*TmdbEpisodeKey, error) {
	episodeKey, xattrInfoFound, err := m.getEpisodeKeyFromXattr(episodeFile)
	if err != nil {
		return nil, err
	}
	if xattrInfoFound {
		log.Debugln(
			"read xattr for TMDB series ID", episodeKey.TmdbSeriesID,
			"season", episodeKey.SeasonNumber,
			"episode", episodeKey.EpisodeNumber,
			"from filename", episodeFile.FileName)
		return episodeKey, nil
	}

	return m.getEpisodeKeyFromFilename(episodeFile)
}

// GetOrCreateEpisodeForEpisodeFile tries to create an Episode object by parsing the filename of the
// given EpisodeFile and looking it up in TMDB. It associates the EpisodeFile with the new Model.
// If no matching episode can be found in TMDB, it returns an error.
func (m *MetadataManager) GetOrCreateEpisodeForEpisodeFile(
	episodeFile *db.EpisodeFile) (*db.Episode, error) {

	if episodeFile.EpisodeID != 0 {
		return db.FindEpisodeByID(episodeFile.EpisodeID)
	}

	episodeKey, err := m.getEpisodeKey(episodeFile)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get episode key from file %s", episodeFile.FilePath)
	}

	episode, err := m.GetOrCreateEpisodeByTmdbID(
		episodeKey.TmdbSeriesID, episodeKey.SeasonNumber, episodeKey.EpisodeNumber)
	if err != nil {
		return nil, err
	}

	episodeFile.Episode = episode
	episodeFile.EpisodeID = episode.ID
	db.SaveEpisodeFile(episodeFile)

	episode.EpisodeFiles = []db.EpisodeFile{*episodeFile}

	return episode, nil
}

// GetOrCreateEpisodeByTmdbID gets or creates an Episode object in the database,
// populating it with the details of the episode indicated by the TMDB ID.
func (m *MetadataManager) GetOrCreateEpisodeByTmdbID(
	seriesTmdbID int, seasonNum int, episodeNum int) (*db.Episode, error) {

	season, err := m.getOrCreateSeasonByTmdbID(seriesTmdbID, seasonNum)
	if err != nil {
		return nil, err
	}

	// Lock so that we don't create the same episode twice
	// TODO(Leon Handreke): More fine-grained locking
	m.seriesCreationMutex.Lock()
	defer m.seriesCreationMutex.Unlock()

	episode, err := db.FindEpisodeByNumber(season, episodeNum)
	if err == nil {
		return episode, nil
	}

	episode = &db.Episode{Season: season, SeasonID: season.ID, EpisodeNum: episodeNum}
	if err := m.UpdateEpisodeMD(episode); err != nil {
		return nil, err
	}

	if m.Subscriber != nil {
		m.Subscriber.EpisodeAdded(episode)
	}

	return episode, nil
}

func (m *MetadataManager) getOrCreateSeriesByTmdbID(
	seriesTmdbID int) (*db.Series, error) {

	// Lock so that we don't create the same series twice
	m.seriesCreationMutex.Lock()
	defer m.seriesCreationMutex.Unlock()

	series, err := db.FindSeriesByTmdbID(seriesTmdbID)
	if err == nil {
		return series, nil
	}

	series = &db.Series{BaseItem: db.BaseItem{TmdbID: seriesTmdbID}}
	if err := m.UpdateSeriesMD(series); err != nil {
		return nil, err
	}

	if m.Subscriber != nil {
		m.Subscriber.SeriesAdded(series)
	}

	return series, nil
}

func (m *MetadataManager) getOrCreateSeasonByTmdbID(
	seriesTmdbID int, seasonNum int) (*db.Season, error) {

	series, err := m.getOrCreateSeriesByTmdbID(seriesTmdbID)
	if err != nil {
		return nil, err
	}

	// Lock so that we don't create the same series twice
	m.seriesCreationMutex.Lock()
	defer m.seriesCreationMutex.Unlock()

	season, err := db.FindSeasonBySeasonNumber(series, seasonNum)
	if err == nil {
		return season, nil
	}

	season = &db.Season{Series: series, SeriesID: series.ID, SeasonNumber: seasonNum}
	if err := m.UpdateSeasonMD(season); err != nil {
		return nil, err
	}

	if m.Subscriber != nil {
		m.Subscriber.SeasonAdded(season)
	}

	return season, nil
}

func (m *MetadataManager) GarbageCollectAllEpisodes() error {
	// TODO(Leon Handreke): We actually only need the ID here.
	episodes, err := db.FindAllEpisodes()
	if err != nil {
		return errors.Wrap(err, "Failed to get all Episodes")
	}
	for _, episode := range episodes {
		m.GarbageCollectEpisodeIfRequired(episode.ID)
	}
	return nil
}

// GarbageCollectEpisodeIfRequired deletes an Episode and its associated Season/Series objects if
// required if no more EpisodeFiles associated with them remain.
func (m *MetadataManager) GarbageCollectEpisodeIfRequired(episodeID uint) error {
	log.Debugln("Garbage collecting episode", episodeID)

	m.getEpisodeLock(episodeID).Lock()
	defer m.getEpisodeLock(episodeID).Unlock()

	episode, err := db.FindEpisodeByID(episodeID)
	if err != nil {
		return errors.Wrap(err, "Failed to refresh episode")
	}

	if len(episode.EpisodeFiles) > 0 {
		return nil
	}

	if err := db.DeleteEpisode(episode.ID); err != nil {
		return errors.Wrap(err, "Failed to delete Episode")
	}

	m.getSeasonLock(episode.SeasonID).Lock()
	defer m.getSeasonLock(episode.SeasonID).Unlock()
	// Garbage collect season
	season, err := db.FindSeason(episode.SeasonID)
	if err != nil {
		return err
	}
	if len(season.Episodes) > 0 {
		return nil
	}
	if err := db.DeleteSeason(season.ID); err != nil {
		return errors.Wrap(err, "Failed to delete Season")
	}

	m.getSeriesLock(season.SeriesID).Lock()
	defer m.getSeriesLock(season.SeriesID).Unlock()
	// Garbage collect series
	series, err := db.FindSeries(season.SeriesID)
	if err != nil {
		return err
	}
	if len(series.Seasons) > 0 {
		return nil
	}
	if err := db.DeleteSeries(series.ID); err != nil {
		return errors.Wrap(err, "Failed to delete Series")
	}

	return nil
}
