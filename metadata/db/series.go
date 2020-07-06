package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
)

// Series holds metadata information about series.
type Series struct {
	BaseItem
	gorm.Model
	Name         string
	FirstAirDate string
	FirstAirYear uint64
	OriginalName string
	Status       string
	Type         string
	Seasons      []*Season
}

// Season holds metadata information about seasons.
type Season struct {
	BaseItem
	gorm.Model
	Name         string
	AirDate      string
	SeasonNumber int
	Series       *Series
	SeriesID     uint
	Episodes     []*Episode
}

// GetSeries get the associated series to this eason
func (s *Season) GetSeries() *Series {
	var series Series
	db.Model(&s).Related(&series)
	return &series
}

// Episode holds metadata information about episodes.
type Episode struct {
	gorm.Model
	BaseItem
	Name         string
	SeasonNum    int
	EpisodeNum   int
	SeasonID     uint
	AirDate      string
	StillPath    string
	Season       *Season
	EpisodeFiles []EpisodeFile
}

// TimeStamp returns a unix timestamp for the given episode.
func (ep *Episode) TimeStamp() int64 {
	return ep.CreatedAt.Unix()
}

// UpdatedAtTimeStamp returns a unix timestamp for the given episode.
func (ep *Episode) UpdatedAtTimeStamp() int64 {
	return ep.UpdatedAt.Unix()
}

// GetSeason returns the associated season
func (ep *Episode) GetSeason() *Season {
	var season Season
	db.Model(ep).Related(&season)
	return &season
}

// GetSeries returns the associated series through season
func (ep *Episode) GetSeries() *Series {
	var series *Series
	season := ep.GetSeason()
	series = season.GetSeries()
	return series
}

// EpisodeFile holds filesystem information about a given episode file.
type EpisodeFile struct {
	gorm.Model
	MediaItem
	EpisodeID uint
	Episode   *Episode
	Streams   []Stream `gorm:"polymorphic:Owner;"`
}

// GetStreams returns all streams for this file
func (file EpisodeFile) GetStreams() []Stream {
	return file.Streams
}

// IsSingleFile returns true if this is the only file for the given episode.
func (file *EpisodeFile) IsSingleFile() bool {
	count := 0
	db.Model(&EpisodeFile{}).Where("episode_id = ?", file.EpisodeID).Count(&count)
	if count <= 1 {
		return true
	}
	return false
}

// GetFileName is a wrapper for the MediaFile interface
func (file EpisodeFile) GetFileName() string {
	return file.FileName
}

// GetFilePath is a wrapper for the MediaFile interface
func (file EpisodeFile) GetFilePath() string {
	return file.FilePath
}

// GetLibrary is a wrapper for the MediaFile interface
func (file EpisodeFile) GetLibrary() *Library {
	var library Library
	db.Model(&file).Related(&library)
	return &library
}

// DeleteWithStreams deletes the episode file and any stale metadata information that might have resulted.
func (file EpisodeFile) DeleteWithStreams() {
	log.WithFields(log.Fields{
		"path": file.FilePath,
	}).Println("Removing file and metadata")

	// Delete all stream information
	db.Unscoped().Delete(Stream{}, "owner_id = ? AND owner_type = 'episode_files'", &file.ID)
	// Delete all file information
	db.Unscoped().Delete(&file)

}

type countResult struct {
	Count uint
}

// UnwatchedEpisodesInSeriesCount retrieves the amount of unwatched episodes in a given series.
func UnwatchedEpisodesInSeriesCount(seriesID uint, userID uint) uint {
	var res countResult
	db.Raw("SELECT COUNT(*) as count FROM episodes WHERE season_id IN(SELECT id FROM seasons WHERE series_id = ?) AND uuid NOT IN(SELECT media_uuid FROM play_states WHERE finished = true AND user_id = ? AND media_uuid IN(SELECT uuid FROM episodes WHERE season_id IN(SELECT id FROM seasons WHERE series_id = ?)))", seriesID, userID, seriesID).Scan(&res)
	return res.Count
}

// UnwatchedEpisodesInSeasonCount retrieves the amount of unwatched episodes in a given season.
func UnwatchedEpisodesInSeasonCount(seasonID uint, userID uint) uint {
	var res countResult
	db.Raw("select count(*) as count from episodes where season_id = ? "+
		"AND uuid NOT IN (SELECT media_uuid FROM play_states WHERE finished = true "+
		"AND user_id = ? AND media_uuid IN( SELECT uuid FROM episodes WHERE season_id = ?))", seasonID, userID,
		seasonID).Scan(&res)
	return res.Count
}

// FindSeriesForMDRefresh finds all series, including unidentified ones.
func FindSeriesForMDRefresh() (series []Series) {
	db.Find(&series)
	return series
}

// FindAllSeries retrieves all identified series from the db.
func FindAllSeries(qd *QueryDetails) ([]*Series, error) {
	var series []*Series
	q := db
	if qd != nil {
		q = q.Offset(qd.Offset).Limit(qd.Limit)

	}
	if err := q.
		Preload("Seasons.Episodes.EpisodeFiles.Streams").
		Find(&series).
		Error; err != nil {
		return nil, err
	}
	return series, nil
}

// SearchSeriesByTitle searches for series based on their name.
func SearchSeriesByTitle(name string) (series []Series) {
	db.Preload("Seasons.Episodes.EpisodeFiles.Streams").
		Where("name LIKE ?", "%"+name+"%").
		Find(&series)
	return series
}

// FindSeriesByUUID retrives a serie based on it's UUID.
func FindSeriesByUUID(uuid string) (*Series, error) {
	return findSeries("uuid = ?", uuid)
}

// FindSeriesByTmdbID retrives a serie based on its TMDB ID
func FindSeriesByTmdbID(tmdbID int) (*Series, error) {
	return findSeries("tmdb_id = ?", tmdbID)
}

// FindSeries finds a series by it's ID
func FindSeries(seriesID uint) (*Series, error) {
	return findSeries("id = ?", seriesID)
}

func findSeries(where ...interface{}) (*Series, error) {
	var series Series

	// We return a singular item in an array so we can use the same GraphQL query we probably want to split this.
	if err := db.Preload("Seasons.Episodes.EpisodeFiles.Streams").
		Take(&series, where...).Error; err != nil {
		return nil, err
	}
	return &series, nil
}

// FindSeasonsForSeries retrieves all season for the given series based on it's UUID.
func FindSeasonsForSeries(seriesID uint) (seasons []Season) {
	db.Preload("Episodes.EpisodeFiles.Streams").Where("series_id = ?", seriesID).Find(&seasons)
	return seasons
}

// FindEpisodesForSeason finds all episodes for the given season ID.
func FindEpisodesForSeason(seasonID uint) (episodes []Episode) {
	db.
		Preload("EpisodeFiles.Streams").
		Where("season_id = ?", seasonID).
		Find(&episodes)

	return episodes
}

// FindSeriesInLibrary finds all series belonging to an EpisodeFile in a given library.
func FindSeriesInLibrary(libraryID uint) (series []Series) {
	db.Raw("SELECT series.* FROM episode_files JOIN episodes ON episodes.id = episode_files.id JOIN seasons ON seasons.id = episodes.season_id JOIN series ON series.id = seasons.series_id WHERE library_id = ? GROUP BY series.tmdb_id", libraryID).Scan(&series)
	return series
}

// FindEpisodeFilesInLibrary finds all episode files in the given library.
func FindEpisodeFilesInLibrary(libraryID uint) ([]EpisodeFile, error) {
	var episodeFiles []EpisodeFile
	err := db.Find(&episodeFiles, "library_id =?", libraryID).Error
	return episodeFiles, err
}

// FindEpisodeFilesInLibraryByLocator finds all episode files in the given library
// under the given locator's path
func FindEpisodeFilesInLibraryByLocator(libraryID uint, locator filesystem.FileLocator) (episodes []EpisodeFile) {
	db.Where("library_id =?", libraryID).
		Where("file_path LIKE ?", fmt.Sprintf("%s%%", locator)).
		Find(&episodes)
	return episodes
}

// FindEpisodeFileByPath Returns the first EpisodeFile matching the provided filePath,
// regardless of whether the EpisodeFile is local or remote.
func FindEpisodeFileByPath(filePath filesystem.Node) (*EpisodeFile, error) {
	var episodeFile EpisodeFile

	if err := db.Where("file_path = ?", filePath.FileLocator().String()).
		First(&episodeFile).Error; err != nil {
		return nil, err
	}
	return &episodeFile, nil
}

// FindEpisodesInLibrary returns all episodes in the given library.
func FindEpisodesInLibrary(libraryID uint) (episodes []Episode) {
	var files []EpisodeFile
	db.Preload("Episode").Where("library_id = ?", libraryID).Find(&files)
	for _, e := range files {
		episodes = append(episodes, *e.Episode)
	}

	return episodes
}

// FindEpisodeFileByUUID finds an EpisodeFile by UUID
func FindEpisodeFileByUUID(uuid string) (*EpisodeFile, error) {
	return findEpisodeFile("uuid = ?", uuid)
}

func findEpisodeFile(where ...interface{}) (*EpisodeFile, error) {
	var episodeFile EpisodeFile
	if err := db.
		Preload("Streams").
		Take(&episodeFile, where...).Error; err != nil {
		return nil, err
	}
	return &episodeFile, nil
}

// FindAllSeasons returns all seasons
func FindAllSeasons() (seasons []Season) {
	db.Find(&seasons)
	return seasons
}

// FindSeasonByUUID finds the season based on it's UUID.
func FindSeasonByUUID(uuid string) (*Season, error) {
	return findSeason("uuid = ?", uuid)
}

// FindSeason finds a season by it's ID
func FindSeason(seasonID uint) (*Season, error) {
	return findSeason("id = ?", seasonID)
}

// FindSeasonBySeasonNumber gets a season by Series/season number
func FindSeasonBySeasonNumber(series *Series, seasonNum int) (*Season, error) {
	return findSeason("series_id = ? AND season_number = ?", series.ID, seasonNum)
}

func findSeason(where ...interface{}) (*Season, error) {
	var season Season

	// We return a singular item in an array so we can use the same GraphQL query we probably want to split this.
	if err := db.
		Preload("Episodes.EpisodeFiles.Streams").
		Preload("Series").
		Take(&season, where...).Error; err != nil {
		return nil, err
	}
	return &season, nil
}

// FindEpisodeByUUID finds a episode based on it's UUID.
func FindEpisodeByUUID(uuid string) (*Episode, error) {
	return findEpisode("uuid = ?", uuid)
}

// FindEpisodeByID finds a episode based on its ID
func FindEpisodeByID(id uint) (*Episode, error) {
	return findEpisode("id = ?", id)
}

// FindEpisodeByNumber finds an Episode by Season/episode number
func FindEpisodeByNumber(season *Season, episodeNum int) (*Episode, error) {
	return findEpisode("season_id = ? AND episode_num = ?", season.ID, episodeNum)
}

func findEpisode(where ...interface{}) (*Episode, error) {
	var episode Episode

	// We return a singular item in an array so we can use the same GraphQL query we probably want to split this.
	if err := db.
		Preload("EpisodeFiles.Streams").
		Take(&episode, where...).Error; err != nil {
		return nil, err
	}
	return &episode, nil
}

func FindAllEpisodes() ([]*Episode, error) {
	var episodes []*Episode
	if err := db.
		Preload("EpisodeFiles.Streams").
		Find(&episodes).Error; err != nil {
		return nil, err
	}
	return episodes, nil
}

// FindAllEpisodeFiles retrieves all episodefiles from the db.
func FindAllEpisodeFiles() (files []EpisodeFile) {
	db.Preload("Library").Find(&files)

	return files
}

// EpisodeFileExists checks whether there already is a EpisodeFile present with the given path.
func EpisodeFileExists(filePath string) bool {
	count := 0
	db.Where("file_path= ?", filePath).Find(&EpisodeFile{}).Count(&count)
	if count == 0 {
		return false
	}
	return true
}

// CreateSeries persists a series in the database.
func CreateSeries(series *Series) {
	db.Create(series)
}

// SaveSeries updates a series in the database.
func SaveSeries(series *Series) error {
	if err := db.Save(series).Error; err != nil {
		return errors.Wrapf(err, "Failed to save Series %s", series.UUID)
	}
	return nil
}

// SaveSeason updates a season in the database.
func SaveSeason(season *Season) error {
	if err := db.Save(season).Error; err != nil {
		return errors.Wrapf(err, "Failed to save Season %s", season.UUID)
	}
	return nil
}

// SaveEpisode updates an episode in the database.
func SaveEpisode(episode *Episode) error {
	if err := db.Save(episode).Error; err != nil {
		return errors.Wrapf(err, "Failed to save Episode %s", episode.UUID)
	}
	return nil
}

// DeleteEpisode deletes an Episode
func DeleteEpisode(episodeID uint) error {
	return db.Unscoped().Delete(&Episode{}, "id = ?", episodeID).Error
}

// DeleteSeason deletes a Season
func DeleteSeason(seasonID uint) error {
	return db.Unscoped().Delete(&Season{}, "id = ?", seasonID).Error
}

// DeleteSeries deletes a Series
func DeleteSeries(seriesID uint) error {
	return db.Unscoped().Delete(&Series{}, "id = ?", seriesID).Error
}

// SaveEpisodeFile updates an episodeFile in the database.
func SaveEpisodeFile(episodeFile *EpisodeFile) error {
	return db.Save(episodeFile).Error
}

// CreateEpisode writes an episode to the db.
func CreateEpisode(episode *Episode) {
	db.Create(episode)
}

// ItemsWithMissingMetadata fetches series with missing metadata.
func ItemsWithMissingMetadata() []string {
	var uuids []string

	// We can probably optimise this by only initiating strings somehow
	var episodes []Episode
	db.Select("uuid").Where("still_path = ''").Find(&episodes)
	for _, episode := range episodes {
		uuids = append(uuids, episode.UUID)
	}

	var movies []Movie
	db.Select("uuid").Where("poster_path = ''").Or("backdrop_path = ''").Find(&movies)
	for _, movie := range movies {
		uuids = append(uuids, movie.UUID)
	}

	var seasons []Season
	db.Select("uuid").Where("poster_path = ''").Find(&seasons)
	for _, season := range seasons {
		uuids = append(uuids, season.UUID)
	}

	var series []Series
	db.Select("uuid").Where("poster_path = ''").Or("backdrop_path = ''").Find(&series)
	for _, s := range series {
		uuids = append(uuids, s.UUID)
	}

	return uuids
}

// FindAllUnidentifiedEpisodeFiles find all EpisodeFiles without an associated Episode
func FindAllUnidentifiedEpisodeFiles(qd *QueryDetails) ([]EpisodeFile, error) {
	var episodeFiles []EpisodeFile

	query := db
	if qd != nil {
		query = query.Offset(qd.Offset).Limit(qd.Limit)
	}

	query = query.Find(&episodeFiles, "episode_id = 0")

	if err := query.Error; err != nil {
		return []EpisodeFile{},
			errors.Wrap(err, "Failed to find unidentified episode files")
	}
	return episodeFiles, nil
}

// FindAllUnidentifiedEpisodeFilesInLibrary find all EpisodeFiles without an associated Episode in a library
func FindAllUnidentifiedEpisodeFilesInLibrary(libraryID uint) ([]*EpisodeFile, error) {
	var episodeFiles []*EpisodeFile

	query := db.Find(&episodeFiles, "episode_id = 0 AND library_id = ?", libraryID)

	if err := query.Error; err != nil {
		return nil,
			errors.Wrap(err, "Failed to find unidentified episode files")
	}
	return episodeFiles, nil
}
