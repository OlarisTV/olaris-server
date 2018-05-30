package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type LibraryManager struct {
	ctx *MetadataContext
}

func NewLibraryManager(ctx *MetadataContext) *LibraryManager {
	return &LibraryManager{ctx: ctx}
}

func (self *LibraryManager) UpdateMD(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		fmt.Println("Updating meta-data for movies")
		self.UpdateMovieMD(library)
	case MediaTypeSeries:
		fmt.Println("Updating meta-data for TV")
		self.UpdateTvMD(library)
	}
}
func (self *LibraryManager) UpdateEpisodeMD() error {
	episodes := []TvEpisode{}
	self.ctx.Db.Where("tmdb_id = ?", 0).Find(&episodes)
	for _, episode := range episodes {
		var season TvSeason
		var tv TvSeries
		self.ctx.Db.Where("id = ?", episode.TvSeasonID).Find(&season)
		self.ctx.Db.Where("id = ?", season.TvSeriesID).Find(&tv)
		fmt.Printf("Grabbing metadata for episode %s for series '%s'\n", episode.EpisodeNum, tv.Name)
		episodeInt, err := strconv.ParseInt(episode.EpisodeNum, 10, 32)
		if err != nil {
			fmt.Println("Could not parse season:", err)
		}
		fullEpisode, err := self.ctx.Tmdb.GetTvEpisodeInfo(tv.TmdbID, season.SeasonNumber, int(episodeInt), nil)
		if err == nil {
			episode.AirDate = fullEpisode.AirDate
			episode.Name = fullEpisode.Name
			episode.TmdbID = fullEpisode.ID
			episode.Overview = fullEpisode.Overview
			episode.StillPath = fullEpisode.StillPath
			self.ctx.Db.Save(&episode)
		} else {
			fmt.Println("Could not grab episode information")

		}
	}
	return nil
}

func (self *LibraryManager) UpdateSeasonMD() error {
	seasons := []TvSeason{}
	self.ctx.Db.Where("tmdb_id = ?", 0).Find(&seasons)
	for _, season := range seasons {
		var tv TvSeries
		self.ctx.Db.Where("id = ?", season.TvSeriesID).Find(&tv)

		fmt.Printf("Grabbing meta-data for season %d of series '%s'\n", season.SeasonNumber, tv.Name)
		fullSeason, err := self.ctx.Tmdb.GetTvSeasonInfo(tv.TmdbID, season.SeasonNumber, nil)
		if err == nil {
			season.AirDate = fullSeason.AirDate
			season.Overview = fullSeason.Overview
			season.Name = fullSeason.Name
			season.TmdbID = fullSeason.ID
			season.PosterPath = fullSeason.PosterPath
			self.ctx.Db.Save(&season)
		} else {
			fmt.Println("Could not grab seasonal information")
		}
	}
	return nil
}

func (self *LibraryManager) UpdateTvMD(library *Library) error {
	series := []TvSeries{}
	self.ctx.Db.Where("tmdb_id = ?", 0).Find(&series)
	for _, serie := range series {
		fmt.Println("Looking up meta-data for series:", serie.Name)
		var options = make(map[string]string)
		searchRes, err := self.ctx.Tmdb.SearchTv(serie.Name, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			fmt.Println("Found Series that matches, using first result and doing deepscan.")
			tv := searchRes.Results[0] // Take the first result for now
			fullTv, err := self.ctx.Tmdb.GetTvInfo(tv.ID, nil)
			if err == nil {
				serie.Overview = fullTv.Overview
				serie.Status = fullTv.Status
				serie.Type = fullTv.Type
			} else {
				fmt.Println("Could not get full results, only adding search results. Error:", err)
			}
			serie.TmdbID = tv.ID
			serie.FirstAirDate = tv.FirstAirDate
			serie.OriginalName = tv.OriginalName
			serie.BackdropPath = tv.BackdropPath
			serie.PosterPath = tv.PosterPath
			self.ctx.Db.Save(&serie)
		}
	}

	self.UpdateSeasonMD()
	self.UpdateEpisodeMD()
	//episodes := []TvEpisode{}
	//self.ctx.Db.Where("tmdb_id = ? AND library_id = ?", 0, library.ID).Find(&episodes)
	//for _, episode := range episodes {
	//}

	return nil
}

func (self *LibraryManager) UpdateMovieMD(library *Library) error {
	movies := []MovieItem{}
	self.ctx.Db.Where("tmdb_id = ? AND library_id = ?", 0, library.ID).Find(&movies)
	for _, movie := range movies {
		fmt.Printf("Attempting to fetch metadata for '%s'\n", movie.Title)
		var options = make(map[string]string)
		if movie.Year > 0 {
			options["year"] = movie.YearAsString()
		}
		searchRes, err := self.ctx.Tmdb.SearchMovie(movie.Title, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			fmt.Println("Found movie that matches, using first result and doing deepscan.")
			mov := searchRes.Results[0] // Take the first result for now
			fullMov, err := self.ctx.Tmdb.GetMovieInfo(mov.ID, nil)
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
			self.ctx.Db.Save(&movie)
		}

	}
	return nil
}

func (self *LibraryManager) Probe(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		fmt.Println("Probing for movies")
		self.ProbeMovies(library)
	case MediaTypeSeries:
		fmt.Println("Probing for series")
		self.ProbeSeries(library)
	}
}

func (self *LibraryManager) ProbeSeries(library *Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {
		if supportedExtensions[filepath.Ext(walkPath)] {
			count := 0
			self.ctx.Db.Where("file_path= ?", walkPath).Find(&TvEpisode{}).Count(&count)
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

				seriesRe := regexp.MustCompile("^(.*)S(\\d{2})E(\\d{2})")
				res := seriesRe.FindStringSubmatch(fileInfo.Name())

				if len(res) > 2 {
					title := Sanitize(res[1])
					season := res[2]
					episode := res[3]
					fmt.Printf("Found '%s' season %s episode %s\n", title, season, episode)
					mi := MediaItem{
						FileName:  fileInfo.Name(),
						FilePath:  walkPath,
						Size:      fileInfo.Size(),
						Title:     title,
						LibraryID: library.ID,
					}
					var tv TvSeries
					var tvs TvSeason

					seasonInt, err := strconv.ParseInt(season, 10, 32)
					if err != nil {
						fmt.Println("Could not parse season:", err)
					}

					self.ctx.Db.FirstOrCreate(&tv, TvSeries{Name: title})
					self.ctx.Db.FirstOrCreate(&tvs, TvSeason{TvSeriesID: tv.ID, Name: title, SeasonNumber: int(seasonInt)})

					ep := TvEpisode{MediaItem: mi, SeasonNum: season, EpisodeNum: episode, TvSeasonID: tvs.ID}
					self.ctx.Db.Create(&ep)
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
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
			self.ctx.Db.Where("file_path= ?", walkPath).Find(&MovieItem{}).Count(&count)
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
				self.ctx.Db.Create(&movie)
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
	self.ctx.Db.Find(&libraries)
	return libraries
}

func (self *LibraryManager) AddLibrary(name string, filePath string) {
	fmt.Printf("Add library '%s' with path '%s'", name, filePath)
	lib := Library{Name: name, FilePath: filePath}
	self.ctx.Db.Create(&lib)
}

func (self *LibraryManager) ActivateAll() {
	for _, lib := range self.AllLibraries() {
		fmt.Println("Scanning library:", lib.Name, lib.FilePath)
		self.Probe(&lib)

		fmt.Println("Updating metadata for library:", lib.Name)
		self.UpdateMD(&lib)
	}
}
