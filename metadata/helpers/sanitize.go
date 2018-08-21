// Package helpers has a collection of helpers that help process metadata details.
package helpers

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Sanitize prepares a filename for parsing. It tries to remove dots and other weird characters to make parsing easier.
func Sanitize(filename string) string {
	// Remove extension
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	filename = strings.Replace(filename, ".", " ", -1)
	filename = strings.Replace(filename, "’", "'", -1)
	filename = strings.Trim(filename, " ")
	filename = strings.Trim(filename, " -")
	return filename
}

// HeavySanitize mangles a filename even further removing often found extra characters in filenames.
func HeavySanitize(title string) (string, string) {
	var year string
	yearReg := regexp.MustCompile("(\\d{4})")
	title = strings.Replace(title, "4k", "", -1)
	title = strings.Replace(title, "1080p", "", -1)
	title = strings.Replace(title, "720p", "", -1)

	res := yearReg.FindStringSubmatch(title)
	if len(res) > 1 {
		year = res[1]
		title = strings.Replace(title, year, "", -1)
		title = Sanitize(title)
	}

	return title, year
}
