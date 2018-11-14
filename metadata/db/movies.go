package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
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
	TmdbID        int
	ReleaseDate   string
	OriginalTitle string
	ImdbID        string
	MovieFiles    []MovieFile
	PlayState     PlayState `gorm:"polymorphic:Owner;"`
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

// DeleteSelfAndMD removes this file and any metadata involved for the movie.
func (file *MovieFile) DeleteSelfAndMD() {
	// Delete all stream information since it's only for this file
	db.Unscoped().Delete(Stream{}, "owner_id = ? AND owner_type = 'movies'", &file.ID)

	db.Where("id = ?", file.MovieID).Find(&file.Movie)

	if file.IsSingleFile() {
		// TODO: Figure out if we can use gorm associations for this
		db.Unscoped().Delete(PlayState{}, "owner_id = ? AND owner_type = 'movies'", file.MovieID)

		// Delete movie
		db.Unscoped().Delete(&file.Movie)
	}

	// Delete all file information
	db.Unscoped().Delete(&file)

}

// String returns a nice overview of the given movie file.
func (file *MovieFile) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", file.Title, file.Year, file.FilePath)
}

// YearAsString returns the year, as a string....
func (movie *Movie) YearAsString() string {
	return strconv.FormatUint(movie.Year, 10)
}

// TimeStamp returns a unix timestamp.
func (movie *Movie) TimeStamp() int64 {
	return movie.CreatedAt.Unix()
}

// FindAllMovieFiles Returns all movies, even once that could not be identified.
func FindAllMovieFiles() (movies []MovieFile) {
	db.Find(&movies)

	return movies
}

// CollectMovieInfo ensures that all relevant information for a movie is loaded
// this can include stream information (audio/video/subtitle tracks) and personalized playstate information.
func CollectMovieInfo(movies []Movie, userID uint) {
	// Can't use 'movie' in range here as it won't modify the original object
	for i := range movies {
		db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
		// TODO(Maran): We should be able to use Gorm's build in polymorphic has_ony query to somehow do this
		db.Model(movies[i]).Where("user_id = ? AND owner_id = ? and owner_type = ?", userID, movies[i].ID, "movies").First(&movies[i].PlayState)
	}
}

// FindAllUnidentifiedMovies retrives all movies without an tmdb_id in the database.
func FindAllUnidentifiedMovies() (movies []Movie) {
	db.Where("tmdb_id = ?", 0).Find(&movies)
	log.Debugln("Collecting unidentified movies", len(movies))

	return movies
}

// FindMoviesForMDRefresh finds all movies, including unidentified ones.
func FindMoviesForMDRefresh() (movies []Movie) {
	db.Find(&movies)
	return movies
}

// FindAllMovies finds all identified movies including all associated information like streams and files.
func FindAllMovies(userID uint) (movies []Movie) {
	db.Where("tmdb_id != 0").Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

// FindMovieByUUID finds the movie specified by the given uuid.
func FindMovieByUUID(uuid *string, userID uint) (movies []Movie) {
	db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

// SearchMovieByTitle search movies by title.
func SearchMovieByTitle(userID uint, name string) (movies []Movie) {
	db.Where("original_title LIKE ?", "%"+name+"%").Find(&movies)
	CollectMovieInfo(movies, userID)
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

// FindMoviesInLibrary finds all movies in the given library.
func FindMoviesInLibrary(libraryID uint, userID uint) (movies []Movie) {
	db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	CollectMovieInfo(movies, userID)

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

// UpdateMovie updates a movie in the database.
func UpdateMovie(movie *Movie) {
	//TODO: This is persisting everything including files and streams, perhaps we can do it more selectively to lower db activity.
	db.Save(movie)
}

// FirstOrCreateMovie returns the first instance or writes a movie to the db.
func FirstOrCreateMovie(movie *Movie, movieDef Movie) {
	db.FirstOrCreate(movie, movieDef)
}

// UpdateMovieFile updates a movieFile in the database.
func UpdateMovieFile(movie *Movie) {
	db.Save(movie)
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

type mergeResult struct {
	TmdbID  uint
	ID      uint
	Counter uint
}

// MergeDuplicateMovies should merge duplicate movies into a singular movie with movie files associated.
func MergeDuplicateMovies() int {
	log.Debugln("Checking for duplicate movies that can be merged.")

	var merging []mergeResult
	rows, err := db.Raw("SELECT tmdb_id,id, count(*) as counter FROM movies WHERE tmdb_id != 0 GROUP BY tmdb_id HAVING counter > 1").Rows()
	if err != nil {
		fmt.Println(err)
	}
	for rows.Next() {
		var res mergeResult
		db.ScanRows(rows, &res)
		merging = append(merging, res)
	}
	rows.Close()

	for _, res := range merging {
		log.WithFields(log.Fields{"tmdb_id": res.TmdbID}).Debugln("Merging movies based on tmdb_id.")

		// We might want to ensure this always works, I'm not sure how deterministic the order of SQLite is.
		db.Exec("UPDATE movie_files SET movie_id=(SELECT id FROM movies WHERE tmdb_id = ? LIMIT 1) WHERE movie_id = ?", res.TmdbID, res.ID)
		db.Exec("DELETE FROM movies WHERE id = ?", res.ID)
	}
	return 0
}
