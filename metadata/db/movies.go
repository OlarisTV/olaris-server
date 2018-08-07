package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strconv"
)

type MovieFile struct {
	gorm.Model
	MediaItem
	Movie   Movie
	MovieID uint
	Streams []Stream `gorm:"polymorphic:Owner;"`
}

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

func (self *MovieFile) IsSingleFile() bool {
	count := 0
	env.Db.Model(&MovieFile{}).Where("movie_id = ?", self.MovieID).Count(&count)
	if count <= 1 {
		return true
	} else {
		return false
	}
}

func (self *MovieFile) DeleteSelfAndMD() {
	// Delete all stream information since it's only for this file
	env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'movies'", &self.ID)

	env.Db.Where("id = ?", self.MovieID).Find(&self.Movie)

	if self.IsSingleFile() {
		// Delete all PlayState information
		env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'movies'", self.MovieID)

		// Delete movie
		env.Db.Delete(&self.Movie)
	}

	// Delete all file information
	env.Db.Delete(&self)

}

func (self *MovieFile) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}

func (self *Movie) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}
func (self *Movie) TimeStamp() int64 {
	return self.CreatedAt.Unix()
}

func FindAllMovieFiles() (movies []MovieFile) {
	env.Db.Find(&movies)

	return movies
}

func CollectMovieInfo(movies []Movie, userID uint) {
	// Can't use 'movie' in range here as it won't modify the original object
	for i, _ := range movies {
		env.Db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
		// TODO(Maran): We should be able to use Gorm's build in polymorphic has_ony query to somehow do this
		env.Db.Model(movies[i]).Where("user_id = ? AND owner_id = ? and owner_type =?", userID, movies[i].ID, "movies").First(&movies[i].PlayState)
	}
}

func FindAllMovies(userID uint) (movies []Movie) {
	env.Db.Where("tmdb_id != 0").Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

func FindMovieWithUUID(uuid *string, userID uint) (movies []Movie) {
	env.Db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}

func SearchMovieByTitle(userID uint, name string) (movies []Movie) {
	env.Db.Where("original_title LIKE ?", "%"+name+"%").Find(&movies)
	CollectMovieInfo(movies, userID)
	return movies
}

func DeleteMoviesFromLibrary(libraryID uint) {
	files := []MovieFile{}
	env.Db.Where("library_id = ?", libraryID).Find(&files)
	for _, file := range files {
		file.DeleteSelfAndMD()
	}
}

func FindMoviesInLibrary(libraryID uint, userID uint) (movies []Movie) {
	env.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}
