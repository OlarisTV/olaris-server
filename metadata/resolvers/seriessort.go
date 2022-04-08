package resolvers

import "strings"

type SeriesSort string

// Maps SeriesSort enum names to their database column names
var _seriesSortToString = map[SeriesSort]string{
	"name":         "name",
	"originalname": "original_name",
	"firstairdate": "first_air_date",
}

func (ss SeriesSort) toLower() SeriesSort {
	return SeriesSort(strings.ToLower(string(ss)))
}

func (ss *SeriesSort) ToString() string {
	return _seriesSortToString[ss.toLower()]
}
