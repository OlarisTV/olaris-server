package metadata

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strconv"
)

type MediaType int

const (
	MediaTypeMovie = iota
	MediaTypeSeries
	MediaTypeMusic
	MediaTypeOtherMovie
)

type MediaItem struct {
	Title     string
	Year      uint64
	FileName  string
	FilePath  string
	Size      int64
	Library   Library
	LibraryID uint
}

func (self *MediaItem) YearAsString() string {
	return strconv.FormatUint(self.Year, 10)
}

type MovieItem struct {
	gorm.Model
	MediaItem
	TmdbID        int
	ReleaseDate   string
	OriginalTitle string
	Overview      string
	BackdropPath  string
	PosterPath    string
	ImdbID        string
}

func (self *MovieItem) String() string {
	return fmt.Sprintf("Movie: %s\nYear: %d\nPath:%s", self.Title, self.Year, self.FilePath)
}
