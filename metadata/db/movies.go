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
	PlayState     PlayState `gorm:"polymorphic:Playstate;"`
}

func (self *MovieFile) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}

func (self *Movie) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}

func CollectMovieInfo(movies []Movie, userID uint) {
	// Can't use 'movie' in range here as it won't modify the original object
	// TODO(Maran): We might want to see if we can make these queries smarter somehow
	for i, _ := range movies {
		env.Db.Model(movies[i]).Preload("Streams").Association("MovieFiles").Find(&movies[i].MovieFiles)
		env.Db.Where("uuid = ? AND user_id = ?", movies[i].UUID, userID).Find(&movies[i].PlayState)
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

func FindMoviesInLibrary(libraryID uint, userID uint) (movies []Movie) {
	env.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	CollectMovieInfo(movies, userID)

	return movies
}
