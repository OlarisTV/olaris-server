package metadata

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func Sanitize(title string) string {
	title = strings.Replace(title, ".", " ", -1)
	title = strings.Replace(title, "’", "'", -1)
	title = strings.Trim(title, " ")
	return title
}

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

func EnsurePath(pathName string) error {
	fmt.Printf("Ensuring folder %s exists.\n", pathName)
	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		fmt.Println("Path does not exist, creating", pathName)
		err = os.MkdirAll(pathName, 0755)
		if err != nil {
			fmt.Println("Could not create path.")
			return err
		}
	}
	return nil
}
