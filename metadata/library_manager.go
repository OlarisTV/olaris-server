package metadata

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/ryanbradynd05/go-tmdb"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type LibraryManager struct {
	db   *gorm.DB
	tmdb *tmdb.TMDb
}

func NewLibraryManager(db *gorm.DB, tmdb *tmdb.TMDb) *LibraryManager {
	return &LibraryManager{db: db, tmdb: tmdb}
}

func (self *LibraryManager) UpdateMD(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		fmt.Println("Updating meta-data for movies")
		self.UpdateMovieMD(library)
	}
}

func (self *LibraryManager) UpdateMovieMD(library *Library) error {
	movies := []MovieItem{}
	self.db.Where("tmdb_id = ? AND library_id = ?", 0, library.ID).Find(&movies)
	for _, movie := range movies {
		fmt.Printf("Attempting to fetch metadata for '%s'\n", movie.Title)
		var options = make(map[string]string)
		if movie.Year > 0 {
			options["year"] = movie.YearAsString()
		}
		searchRes, err := self.tmdb.SearchMovie(movie.Title, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			fmt.Println("Found movie that matches, using first result and doing deepscan.")
			mov := searchRes.Results[0] // Take the first result for now
			fullMov, err := self.tmdb.GetMovieInfo(mov.ID, nil)
			if err == nil {
				movie.Overview = fullMov.Overview
				movie.ImdbID = fullMov.ImdbID
			} else {
				fmt.Println("Could not get full results, only adding search results. Error:", err)
			}
			movie.TmdbID = mov.ID
			movie.ReleaseDate = mov.ReleaseDate
			movie.OriginalTitle = mov.OriginalTitle
			movie.BackdropPath = mov.BackdropPath
			movie.PosterPath = mov.PosterPath
			self.db.Save(movie)
		}

	}
	return nil
}

func (self *LibraryManager) Probe(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		fmt.Println("Probing for movies")
		self.ProbeMovies(library)
	}
}
func (self *LibraryManager) ProbeMovies(library *Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {
		var title string
		var year uint64

		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(walkPath)] {
			count := 0
			self.db.Where("file_path= ?", walkPath).Find(&MovieItem{}).Count(&count)
			if count == 0 {
				fileInfo, err := os.Stat(walkPath)

				if err != nil {
					// This catches broken symlinks
					if _, ok := err.(*os.PathError); ok {
						fmt.Println("Got an error while statting file:", err)
						return nil
					}
					return err
				}

				movieRe := regexp.MustCompile("(.*)\\((\\d{4})\\)")
				res := movieRe.FindStringSubmatch(fileInfo.Name())

				if len(res) > 1 {
					title = Sanitize(res[1])
				}
				if len(res) > 2 {
					year, err = strconv.ParseUint(res[2], 10, 32)
					if err != nil {
						fmt.Println("Could not parse year:", err)
					}
				}

				if title == "" {
					basename := fileInfo.Name()
					name := strings.TrimSuffix(basename, filepath.Ext(basename))
					fmt.Println("Could not parse title for:")
					fmt.Println("Trying heavy sanitizing")
					var yearStr string
					title, yearStr = HeavySanitize(name)
					year, err = strconv.ParseUint(yearStr, 10, 32)
					if err != nil {
						fmt.Println("Could not parse year:", err)
					}
					fmt.Println("attempted to find some stuff", title, year)
				}

				mi := MediaItem{
					FileName:  fileInfo.Name(),
					FilePath:  walkPath,
					Size:      fileInfo.Size(),
					Title:     title,
					Year:      year,
					LibraryID: library.ID,
				}
				movie := MovieItem{MediaItem: mi}
				fmt.Println(movie.String())
				self.db.Create(&movie)
			} else {
				fmt.Printf("Path '%s' already exists in library.\n", walkPath)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		return

	}
}
func (self *LibraryManager) AllLibraries() []Library {
	var libraries []Library
	self.db.Find(&libraries)
	return libraries
}

func (self *LibraryManager) AddLibrary(name string, filePath string) {
	fmt.Printf("Add library '%s' with path '%s'", name, filePath)
	lib := Library{Name: name, FilePath: filePath}
	self.db.Create(&lib)
}

func (self *LibraryManager) ActivateAll() {
	for _, lib := range self.AllLibraries() {
		fmt.Println("Scanning library:", lib.Name, lib.FilePath)
		self.Probe(&lib)

		fmt.Println("Updating metadata for library:", lib.Name)
		self.UpdateMD(&lib)
	}
}
