package parsers

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/helpers"
)

var yearRegex = regexp.MustCompile("([\\[\\(]?((19|20)\\d{2})[\\]\\)]?)")
var seriesRegex = regexp.MustCompile("^(.*)S(\\d{1,2})E(\\d{1,2})")
var seriesFallbackRegex = regexp.MustCompile("^(.*)(\\d{1,2})x(\\d{1,2})")

var seasonRegex = regexp.MustCompile("[Ss](eason|)\\s?(\\d{1,3})")
var firstNumberRegex = regexp.MustCompile("[0-9]{1,3}")

// ParsedSeriesInfo holds extracted information from the given filename.
type ParsedSeriesInfo struct {
	Year       string
	Title      string
	EpisodeNum int
	SeasonNum  int
}

func (psi *ParsedSeriesInfo) logFields() log.Fields {
	return log.Fields{"year": psi.Year, "title": psi.Title, "episodeNum": psi.EpisodeNum, "seasonNum": psi.SeasonNum}
}

// ParseSeriesName attempts to parse a filename looking for episode/season information.
func ParseSeriesName(filePath string) *ParsedSeriesInfo {
	filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath))
	fileName := filepath.Base(filePath)
	log.WithFields(log.Fields{"filename": fileName}).Debugln("Parsing filename for episode information.")
	var err error
	var psi = ParsedSeriesInfo{}
	defer func(p *ParsedSeriesInfo) {
		log.WithFields(p.logFields()).Debugln("Done parsing episode.")
	}(&psi)

	yearResult := yearRegex.FindStringSubmatch(filePath)
	if len(yearResult) > 0 {
		yearString := yearResult[2]
		log.WithFields(log.Fields{"year": yearString}).Println("Found release year.")
		// Remove Year data from original filePath and fileName
		filePath = strings.Replace(filePath, yearResult[1], "", -1)
		fileName = strings.Replace(fileName, yearResult[1], "", -1)
		psi.Year = yearString
		if err != nil {
			log.WithError(err).Warnln("Could not convert year to uint")
		}
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
		return &psi
	}

	//We expect a folder structure of Series/Season/Episode.file
	fileParent := filepath.Base(filepath.Dir(filePath))
	seasonResult := seasonRegex.MatchString(strings.ToLower(fileParent))
	if !seasonResult {
		psi.Title = helpers.Sanitize(fileName)
		return &psi
	}
	seasonNumber := firstNumberRegex.FindAllString(fileParent, -1)
	psi.SeasonNum, err = strconv.Atoi(seasonNumber[0])
	if err != nil {
		log.WithError(err).Debugln("Could not convert season to uint: ")
	}

	episodeNumber := firstNumberRegex.FindAllString(filepath.Base(filePath), -1)
	if len(episodeNumber) < 1 {
		log.Warnf("could not find an episode number in file %s", fileName)
		psi.Title = helpers.Sanitize(fileName)
		return &psi
	}
	psi.EpisodeNum, err = strconv.Atoi(episodeNumber[0])
	if err != nil {
		log.WithError(err).Debugln("Could not convert episode to uint: ")
	}

	seriesName := filepath.Base(filepath.Dir(filepath.Dir(filePath)))
	psi.Title = helpers.Sanitize(seriesName)

	return &psi
}
