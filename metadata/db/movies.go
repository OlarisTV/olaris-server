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

func (movie *Movie) logFields() log.Fields {
	return log.Fields{"title": movie.OriginalTitle, "tmdbId": movie.TmdbID}
}

// IsSingleFile returns true if this is the only file for this movie.
func (file *MovieFile) IsSingleFile() bool {
	count := 0
	env.Db.Model(&MovieFile{}).Where("movie_id = ?", file.MovieID).Count(&count)
	if count <= 1 {
		return true
	}
	return false
}

// DeleteSelfAndMD removes this file and any metadata involved for the movie.
func (file *MovieFile) DeleteSelfAndMD() {
	// Delete all stream information since it's only for this file
	env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'movies'", &file.ID)

	env.Db.Where("id = ?", file.MovieID).Find(&file.Movie)

	if file.IsSingleFile() {
		// Delete all PlayState information
		env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'movies'", file.MovieID)

		// Delete movie
		env.Db.Delete(&file.Movie)
	}

	// Delete all file information
	env.Db.Delete(&file)

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
	env.Db.Find(&movies)

	return movies
}

// CollectMovieInfo ensures that all relevant information for a movie is loaded
// this can include stream information (audio/video/subtitle tracks) and personalized playstate information.
func CollectMovieInfo(movies []Movie, userID uint) {
	// Can't use 'movie' in range here as it won't modify the original object
	for i := range movies {
		env.Db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
		// TODO(Maran): We should be able to use Gorm's build in polymorphic has_ony query to somehow do this
		env.Db.Model(movies[i]).Where("user_id = ? AND owner_id = ? and owner_type =?", userID, movies[i].ID, "movies").First(&movies[i].PlayState)
	}
}

// FindAllMovies finds all identified movies.
func FindAllMovies(userID uint) (movies []Movie) {
	env.Db.Where("tmdb_id != 0").Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

// FindMovieWithUUID finds the movie specified by the given uuid.
func FindMovieWithUUID(uuid *string, userID uint) (movies []Movie) {
	env.Db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

// SearchMovieByTitle search movies by title.
func SearchMovieByTitle(userID uint, name string) (movies []Movie) {
	env.Db.Where("original_title LIKE ?", "%"+name+"%").Find(&movies)
	CollectMovieInfo(movies, userID)
	return movies
}

// DeleteMoviesFromLibrary removes all movies from the given library.
func DeleteMoviesFromLibrary(libraryID uint) {
	files := []MovieFile{}
	env.Db.Where("library_id = ?", libraryID).Find(&files)
	for _, file := range files {
		file.DeleteSelfAndMD()
	}
}

// FindMoviesInLibrary finds all movies in the given library.
func FindMoviesInLibrary(libraryID uint, userID uint) (movies []Movie) {
	env.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}
