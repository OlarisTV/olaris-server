package db

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
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

func CollectMovieInfo(movies []Movie, userID uint) {
	// Can't use 'movie' in range here as it won't modify the original object
	// TODO(Maran): We might want to see if we can make these queries smarter somehow
	for i, _ := range movies {
		env.Db.Model(movies[i]).Association("MovieFiles").Find(&movies[i].MovieFiles)
		env.Db.Where("uuid = ? AND user_id = ?", movies[i].UUID, userID).Find(&movies[i].PlayState)
	}
}

func FindAllMovies(ctx context.Context) (movies []Movie) {
	env.Db.Where("tmdb_id != 0").Find(&movies)
	CollectMovieInfo(movies, helpers.GetUserID(ctx))

	return movies
}
func FindMovieWithUUID(ctx context.Context, uuid *string) (movies []Movie) {
	env.Db.Where("tmdb_id != 0 AND uuid = ?", uuid).Find(&movies)
	CollectMovieInfo(movies, helpers.GetUserID(ctx))

	return movies
}

func FindMoviesInLibrary(ctx context.Context, libraryID uint) (movies []Movie) {
	env.Db.Where("library_id = ? AND tmdb_id != 0", libraryID).Find(&movies)
	CollectMovieInfo(movies, helpers.GetUserID(ctx))

	return movies
}
