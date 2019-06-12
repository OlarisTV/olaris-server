// Package parsers has a collection of parsers that can extract useful information out of filenames.
package parsers

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/metadata/helpers"
	"regexp"
	"strconv"
	"strings"
)

// ParsedMovieInfo holds extracted information from the given filename.
type ParsedMovieInfo struct {
	Year  uint64
	Title string
}

var movieRe = regexp.MustCompile("(.*)\\((\\d{4})\\)")

// ParseMovieName attempts to parse a filename looking for movie information.
func ParseMovieName(fileName string) *ParsedMovieInfo {
	log.WithFields(log.Fields{"filename": fileName}).Debugln("Parsing file for movie information.")
	psi := ParsedMovieInfo{}
	var err error
	var year string

	res := movieRe.FindStringSubmatch(fileName)

	if len(res) > 1 {
		psi.Title = helpers.Sanitize(res[1])
	} else {
		psi.Title, year = helpers.HeavySanitize(helpers.Sanitize(fileName))
	}

	// Year was also found
	if len(res) > 2 {
		year = res[2]
	}

	if year != "" {
		psi.Year, err = strconv.ParseUint(year, 10, 32)
		if err != nil {
			log.Warnln("Could not convert year to uint:", err)
		}
		log.WithFields(log.Fields{"year": psi.Year}).Debugln("Found release year.")
	}

	if psi.Title == "" {
		log.WithFields(log.Fields{
			"filename": fileName,
		}).Warnln("Could not parse title, doing some heavy sanitizing and trying it again.")
		var yearStr string
		psi.Title, yearStr = helpers.HeavySanitize(fileName)
		if yearStr != "" {
			psi.Year, err = strconv.ParseUint(yearStr, 10, 32)
			if err != nil {
				log.Warnln("Could not convert year to uint:", err)
			}
		}
	}
	// Remove everything with at least two spaces and everything after, normally this is just garbage.
	psi.Title = regexp.MustCompile("\\s{2,}.*").ReplaceAllString(psi.Title, "")
	psi.Title = strings.Trim(psi.Title, " ")

	log.WithFields(log.Fields{"title": psi.Title}).Debugln("Done parsing title.")
	return &psi
}
