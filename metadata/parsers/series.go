package parsers

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/helpers"
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
	log.Debugf("Parsing filename '%s' for episode information.", fileName)
	var err error
	var psi = ParsedSeriesInfo{}

	yearResult := yearRegex.FindStringSubmatch(fileName)
	if len(yearResult) > 1 {
		yearString := yearResult[2]
		log.Debugf("Found release year '%s'", yearString)
		// Remove Year data from original fileName
		fileName = strings.Replace(fileName, yearResult[1], "", -1)
		psi.Year, err = strconv.ParseUint(yearString, 10, 32)
		if err != nil {
			log.Warnln("Could not convert year to uint:", err)
		}
		log.Debugf("Removed year from episode information, resulting in '%s'", fileName)
	}

	// Find out episode numbers
	res := seriesRegex.FindStringSubmatch(fileName)
	if len(res) < 3 {
		// Fall back to some rarer used formats like 03x03 for season/episode
		res = seriesFallbackRegex.FindStringSubmatch(fileName)
	}

	// We expect a title, a season and an episode
	if len(res) > 2 {
		psi.SeasonNum, err = strconv.Atoi(res[2])
		if err != nil {
			log.Warnln("Could not convert season to uint:", err)
		}
		psi.EpisodeNum, err = strconv.Atoi(res[3])
		if err != nil {
			log.Warnln("Could not convert episode to uint:", err)
		}
		psi.Title = helpers.Sanitize(res[1])
	} else {
		psi.Title = helpers.Sanitize(fileName)
	}
	log.Debugf("Done parsing found season '%d' episode '%d' for series '%s'", psi.SeasonNum, psi.EpisodeNum, psi.Title)

	return &psi
}
