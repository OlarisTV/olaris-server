package parsers

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/helpers"
	"regexp"
	"strconv"
)

// ParsedMovieInfo holds extracted information from the given filename.
type ParsedMovieInfo struct {
	Year  uint64
	Title string
}

var movieRe = regexp.MustCompile("(.*)\\((\\d{4})\\)")

// ParseMovieName attempts to parse a filename looking for movie information.
func ParseMovieName(fileName string) *ParsedMovieInfo {
	log.Debugf("Parsing filename '%s' for movie information.", fileName)
	psi := ParsedMovieInfo{}
	var err error

	res := movieRe.FindStringSubmatch(fileName)

	if len(res) > 1 {
		psi.Title = helpers.Sanitize(res[1])
	}

	// Year was also found
	if len(res) > 2 {
		psi.Year, err = strconv.ParseUint(res[2], 10, 32)
		if err != nil {
			log.Warnln("Could not convert year to uint:", err)
		}
		log.Debugf("Found release year '%v'.", psi.Year)
	}

	if psi.Title == "" {
		log.Warnln("Could not parse title, doing some heavy sanitizing and trying it again.")
		var yearStr string
		psi.Title, yearStr = helpers.HeavySanitize(fileName)
		if yearStr != "" {
			psi.Year, err = strconv.ParseUint(yearStr, 10, 32)
			if err != nil {
				log.Warnln("Could not convert year to uint:", err)
			}
		}
	}
	log.Debugf("Parsed title is '%s'.", psi.Title)
	return &psi
}
