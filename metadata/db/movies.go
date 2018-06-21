package db

import (
	"fmt"
	"github.com/jinzhu/gorm"
)

type MovieItem struct {
	gorm.Model
	MediaItem
	TmdbID        int
	ReleaseDate   string
	OriginalTitle string
	ImdbID        string
}

func (self *MovieItem) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}

func FindAllMovies() (movies []MovieItem) {
	ctx.Db.Where("tmdb_id != 0").Find(&movies)
	return movies
}
func FindMovieWithUUID(uuid *string) (movies []MovieItem) {
	ctx.Db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	return movies
}

func FindMoviesInLibrary(libraryID uint) (movies []MovieItem) {
	ctx.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)

	return movies
}
