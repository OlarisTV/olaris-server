package resolvers

import "strings"

type MovieSort string

// Maps MovieSort enum names to their database column names
var _movieSortToString = map[MovieSort]string{
	"title":       "title",
	"name":        "original_title",
	"releasedate": "release_date",
}

func (ms MovieSort) toLower() MovieSort {
	return MovieSort(strings.ToLower(string(ms)))
}

func (ms *MovieSort) ToString() string {
	return _movieSortToString[ms.toLower()]
}
