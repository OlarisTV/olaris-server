package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strconv"
)

type MovieFile struct {
	MediaItem
	Movie   Movie
	MovieID uint
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
	PlayState     PlayState `gorm:"polymorphic:Playstae;"`
}

func (self *MovieFile) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}

func (self *Movie) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}

func FindAllMovies() (movies []Movie) {
	ctx.Db.Where("tmdb_id != 0").Find(&movies)
	// Can't use 'movie' in range here as it won't modify the original object
	// TODO(Maran): DRY this up
	for i, _ := range movies {
		ctx.Db.Model(movies[i]).Association("MovieFiles").Find(&movies[i].MovieFiles)
		ctx.Db.Where("uuid = ?", movies[i].UUID).Find(&movies[i].PlayState)
	}
	return movies
}
func FindMovieWithUUID(uuid *string) (movies []Movie) {
	ctx.Db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	// TODO(Maran): DRY this up
	for i, _ := range movies {
		ctx.Db.Model(movies[i]).Association("MovieFiles").Find(&movies[i].MovieFiles)
		ctx.Db.Where("uuid = ?", movies[i].UUID).Find(&movies[i].PlayState)
	}
	return movies
}

func FindMoviesInLibrary(libraryID uint) (movies []Movie) {
	ctx.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	// TODO(Maran): DRY this up
	for i, _ := range movies {
		ctx.Db.Model(movies[i]).Association("MovieFiles").Find(&movies[i].MovieFiles)
		ctx.Db.Where("uuid = ?", movies[i].UUID).Find(&movies[i].PlayState)
	}

	return movies
}
