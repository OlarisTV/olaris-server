package parsers

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
	"regexp"
	"strconv"
	"strings"
)

var yearRegex = regexp.MustCompile("([\\[\\(]?((?:19[0-9]|20[01])[0-9])[\\]\\)]?)")
var seriesRegex = regexp.MustCompile("^(.*)S(\\d{1,2})E(\\d{1,2})")
var seriesFallbackRegex = regexp.MustCompile("^(.*)(\\d{1,2})x(\\d{1,2})")

type ParsedSeriesInfo struct {
	Year       uint64
	Title      string
	EpisodeNum int
	SeasonNum  int
}

func ParseSerieName(fileName string) *ParsedSeriesInfo {
	var err error
	var psi = ParsedSeriesInfo{}

	yearResult := yearRegex.FindStringSubmatch(fileName)
	if len(yearResult) > 1 {
		yearString := yearResult[2]
		// Remove Year data from original fileName
		fileName = strings.Replace(fileName, yearResult[1], "", -1)
		psi.Year, err = strconv.ParseUint(yearString, 10, 32)
		if err != nil {
			fmt.Println("Could not convert year to int:", err)
		}
	}

	// Find out episode numbers
	res := seriesRegex.FindStringSubmatch(fileName)
	if len(res) < 3 {
		res = seriesFallbackRegex.FindStringSubmatch(fileName)
	}

	// We expect a title, a season and an episode
	if len(res) > 2 {
		psi.SeasonNum, err = strconv.Atoi(res[2])
		if err != nil {
			fmt.Println("Could not convert season to int:", err)
		}
		psi.EpisodeNum, err = strconv.Atoi(res[3])
		if err != nil {
			fmt.Println("Could not convert episode to int:", err)
		}
		psi.Title = helpers.Sanitize(res[1])
	} else {
		psi.Title = helpers.Sanitize(fileName)
	}

	return &psi
}
