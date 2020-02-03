package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/filesystem"
	"strconv"
)

// MovieFile is used to store fileinformation about a movie.
type MovieFile struct {
	gorm.Model
	MediaItem
	Movie   Movie
	MovieID uint
	Streams []Stream `gorm:"polymorphic:Owner;"`
}

// Movie is used to store movie metadata information.
type Movie struct {
	gorm.Model
	BaseItem
	Title         string
	Year          uint64
	ReleaseDate   string
	OriginalTitle string
	ImdbID        string
	MovieFiles    []MovieFile
}

// LogFields defines some standard items to log in debug messages.
func (movie *Movie) LogFields() log.Fields {
	return log.Fields{"title": movie.OriginalTitle, "tmdbId": movie.TmdbID}
}

// IsSingleFile returns true if this is the only file for this movie.
func (file *MovieFile) IsSingleFile() bool {
	count := 0
	db.Model(&MovieFile{}).Where("movie_id = ?", file.MovieID).Count(&count)
	if count <= 1 {
		return true
	}
	return false
}

// GetFileName is a wrapper for the MediaFile interface
func (file MovieFile) GetFileName() string {
	return file.FileName
}

// GetFilePath is a wrapper for the MediaFile interface
func (file MovieFile) GetFilePath() string {
	return file.FilePath
}

// GetLibrary is a wrapper for the MediaFile interface
func (file MovieFile) GetLibrary() *Library {
	var library Library
	db.Model(&file).Related(&library)
	return &library
}

// GetStreams returns all streams for this file
func (file MovieFile) GetStreams() []Stream {
	return file.Streams
}

// DeleteSelfAndMD removes this file and any metadata involved for the movie.
func (file MovieFile) DeleteSelfAndMD() {
	log.WithFields(log.Fields{
		"path": file.FilePath,
	}).Println("Removing file and metadata")

	// Delete all stream information since it's only for this file
	db.Unscoped().Delete(Stream{}, "owner_id = ? AND owner_type = 'movies'", &file.ID)

	db.Where("id = ?", file.MovieID).Find(&file.Movie)

	if file.IsSingleFile() {
		// TODO: Figure out if we can use gorm associations for this
		db.Unscoped().Delete(PlayState{}, "media_uuid = ?", file.UUID)

		// Delete movie
		if file.MovieID != 0 {
			db.Unscoped().Delete(&file.Movie)
		}
	}

	// Delete all file information
	db.Unscoped().Delete(&file)

}

// String returns a nice overview of the given movie file.
func (file *MovieFile) String() string {
	return fmt.Sprintf("MovieFile Path:%s", file.FilePath)
}

// YearAsString returns the year, as a string....
func (movie *Movie) YearAsString() string {
	return strconv.FormatUint(movie.Year, 10)
}

// TimeStamp returns a unix timestamp.
func (movie *Movie) TimeStamp() int64 {
	return movie.CreatedAt.Unix()
}

// UpdatedAtTimeStamp returns a unix timestamp.
func (movie *Movie) UpdatedAtTimeStamp() int64 {
	return movie.UpdatedAt.Unix()
}

// FindAllMovieFiles Returns all movies, even once that could not be identified.
func FindAllMovieFiles() (movies []MovieFile) {
	db.Preload("Library").Find(&movies)

	return movies
}

// FindMovieFileByPath Returns the first MovieFile matching the provided filePath,
// regardless of whether the MovieFile is local or remote.
func FindMovieFileByPath(filePath filesystem.Node) (*MovieFile, error) {
	var movieFile MovieFile
	if err := db.Where("file_path = ?", filePath.FileLocator().String()).
		First(&movieFile).Error; err != nil {
		return nil, err
	}
	return &movieFile, nil
}

// QueryDetails specify arguments for media queries
type QueryDetails struct {
	UserID uint
	Offset int
	Limit  int
}

// CollectMovieInfo ensures that all relevant information for a movie is loaded
// this can include stream information (audio/video/subtitle tracks) and personalized playstate information.
func CollectMovieInfo(movie *Movie) {
	db.Model(movie).
		Preload("Streams").
		Association("MovieFiles").
		Find(&movie.MovieFiles)
}

// FindAllUnidentifiedMovieFilesInLibrary retrieves all movies without an tmdb_id in the database.
func FindAllUnidentifiedMovieFilesInLibrary(libraryID uint) ([]*MovieFile, error) {
	var files []*MovieFile
	if err := db.
		Find(&files, "movie_id = 0 AND library_id = ?", libraryID).
		Error; err != nil {
		return nil, err
	}
	return files, nil
}

// FindAllUnidentifiedMovieFiles find all MovieFiles without an associated Movie
func FindAllUnidentifiedMovieFiles(qd QueryDetails) ([]MovieFile, error) {
	var movieFiles []MovieFile

	query := db.
		Find(&movieFiles, "movie_id = 0").
		Offset(qd.Offset).Limit(qd.Limit)
	if err := query.Error; err != nil {
		return []MovieFile{},
			errors.Wrap(err, "Failed to find unidentified movie files")
	}
	return movieFiles, nil
}

// FindMoviesForMDRefresh finds all movies, including unidentified ones.
func FindMoviesForMDRefresh() (movies []Movie) {
	db.Find(&movies)
	return movies
}

// FindAllMovies finds all identified movies including all associated information like streams and files.
func FindAllMovies(qd *QueryDetails) (movies []Movie) {
	q := db
	if qd != nil {
		q = q.Limit(qd.Limit).Offset(qd.Offset)
	}
	q = q.Find(&movies)
	for _, movie := range movies {
		CollectMovieInfo(&movie)
	}

	return movies
}

// FindMovieByUUID finds the movie specified by the given uuid.
func FindMovieByUUID(uuid string) (*Movie, error) {
	return findMovie("uuid = ?", uuid)
}

// FindMovieByTmdbID finds the movie specified by the given TMDB ID
func FindMovieByTmdbID(tmdbID int) (*Movie, error) {
	return findMovie("tmdb_id = ?", tmdbID)
}

// FindMovieByID finds the movie specified by the given ID.
func FindMovieByID(id uint) (*Movie, error) {
	return findMovie("id = ?", id)
}

func findMovie(where ...interface{}) (*Movie, error) {
	var movie Movie
	if err := db.Take(&movie, where...).Error; err != nil {
		return nil, err
	}
	CollectMovieInfo(&movie)
	return &movie, nil
}

// SearchMovieByTitle search movies by title.
func SearchMovieByTitle(name string) (movies []Movie) {
	db.Where("original_title LIKE ?", "%"+name+"%").Find(&movies)
	for _, movie := range movies {
		CollectMovieInfo(&movie)
	}
	return movies
}

// DeleteMoviesFromLibrary removes all movies from the given library.
func DeleteMoviesFromLibrary(libraryID uint) {
	files := []MovieFile{}
	db.Where("library_id = ?", libraryID).Find(&files)
	for _, file := range files {
		file.DeleteSelfAndMD()
	}
}

// FindMoviesInLibrary finds movies that have files in a certain library
func FindMoviesInLibrary(libraryID uint) (movies []Movie) {
	var files []MovieFile
	db.Preload("Movie").Where("library_id = ?", libraryID).Find(&files)
	for _, f := range files {
		movies = append(movies, f.Movie)
	}
	return movies

}

// FindMovieFilesInLibrary finds all movies in the given library.
func FindMovieFilesInLibrary(libraryID uint) (movies []MovieFile) {
	db.Where("library_id =?", libraryID).Find(&movies)

	return movies
}

// FindMovieFilesInLibraryByLocator finds all movie files in the
// provided library under the locator's path
func FindMovieFilesInLibraryByLocator(libraryID uint, locator filesystem.FileLocator) (movies []MovieFile) {
	db.Where("library_id =?", libraryID).
		Where("file_path LIKE ?", fmt.Sprintf("%s%%", locator)).
		Find(&movies)

	return movies
}

// CreateMovieFile persists a moviefile in the database.
func CreateMovieFile(movie *MovieFile) {
	db.Create(movie)
}

// CreateMovie persists a movie in the database.
func CreateMovie(movie *Movie) {
	db.Create(movie)
}

// SaveMovie updates a movie in the database.
func SaveMovie(movie *Movie) error {
	//TODO: This is persisting everything including files and streams, perhaps we can do it more selectively to lower db activity.
	return db.Save(movie).Error
}

// DeleteMovie deletes the movie from the database
func DeleteMovie(movie *Movie) {
	//TODO: This is persisting everything including files and streams, perhaps we can do it more selectively to lower db activity.
	db.Delete(movie)
}

// SaveMovieFile saves a MovieFile
func SaveMovieFile(movieFile *MovieFile) {
	db.Save(movieFile)
}

// FirstMovie gets the first movie out of the database (used in tests).
func FirstMovie() Movie {
	var movie Movie
	db.First(&movie)
	return movie
}

// MovieFileExists checks whether there already is a EpisodeFile present with the given path.
func MovieFileExists(filePath string) bool {
	count := 0
	db.Where("file_path= ?", filePath).Find(&MovieFile{}).Count(&count)
	if count == 0 {
		return false
	}
	return true
}

// FindMovieFileByUUID finds a specific movie based on it's UUID
func FindMovieFileByUUID(uuid string) (*MovieFile, error) {
	var movieFile MovieFile
	if err := db.First(&movieFile, "uuid = ?", uuid).Error; err != nil {
		return nil, err
	}
	return &movieFile, nil
}

// FindMovieForMovieFile accepts a movieFile and returns the movie
func FindMovieForMovieFile(movieFile *MovieFile) (*Movie, error) {
	var movie Movie
	if err := db.Model(movieFile).Related(&movie).Error; err != nil {
		return nil, err
	}
	CollectMovieInfo(&movie)
	return &movie, nil
}

// FindFilesForMovieUUID finds all movieFiles for the associated movie UUIDs
func FindFilesForMovieUUID(uuid string) (movieFiles []*MovieFile) {
	db.Joins("JOIN movies ON movies.id = movie_files.movie_id").Where("movies.uuid = ?", uuid).Find(&movieFiles)
	return movieFiles
}

// FindStreamsForMovieFileUUID finds all movieStreams for the associated movieFile UUIDs
func FindStreamsForMovieFileUUID(uuid string) (streams []*Stream) {
	db.Joins("JOIN movie_files ON movie_files.id = streams.owner_id AND owner_type = 'movie_files'").Where("movie_files.uuid = ?", uuid).Find(&streams)
	return streams
}
