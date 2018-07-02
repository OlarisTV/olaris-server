package parsers

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
	"regexp"
	"strconv"
)

type ParsedMovieInfo struct {
	Year  uint64
	Title string
}

var movieRe = regexp.MustCompile("(.*)\\((\\d{4})\\)")

func ParseMovieName(fileName string) *ParsedMovieInfo {
	psi := ParsedMovieInfo{}
	var err error

	res := movieRe.FindStringSubmatch(fileName)

	// No year has been parsed
	if len(res) > 1 {
		psi.Title = helpers.Sanitize(res[1])
	}

	// Year was also found
	if len(res) > 2 {
		psi.Year, err = strconv.ParseUint(res[2], 10, 32)
		if err != nil {
			fmt.Println("Could not convert year to i:", err)
		}
	}

	if psi.Title == "" {
		fmt.Println("Could not parse title, going to try some heavy lifting")
		var yearStr string
		psi.Title, yearStr = helpers.HeavySanitize(fileName)
		psi.Year, err = strconv.ParseUint(yearStr, 10, 32)
		if err != nil {
			fmt.Println("Could not convert year:", err)
		}
	}
	return &psi
}
