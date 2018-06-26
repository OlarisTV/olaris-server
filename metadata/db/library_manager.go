package db

import (
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/fsnotify/fsnotify"
	"gitlab.com/bytesized/bytesized-streaming/metadata/helpers"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var supportedExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".mov": true,
	".avi": true,
}

const (
	MediaTypeMovie = iota
	MediaTypeSeries
	MediaTypeMusic
	MediaTypeOtherMovie
)

type LibraryManager struct {
	pool    *tunny.Pool
	watcher *fsnotify.Watcher
}

type EpisodePayload struct {
	series  TvSeries
	season  TvSeason
	episode TvEpisode
}

func NewLibraryManager(watcher *fsnotify.Watcher) *LibraryManager {
	manager := LibraryManager{}
	if watcher != nil {
		manager.watcher = watcher
	}
	// The MovieDB currently has a 40 requests per 10 seconds limit. Assuming every request takes a second then four workers is probably ideal.
	manager.pool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		fmt.Println("Starting worker")
		ep, ok := payload.(EpisodePayload)
		if ok {
			fmt.Printf("Worker in '%s', S%dE%s\n", ep.series.Name, ep.season.SeasonNumber, ep.episode.EpisodeNum)
			err := manager.UpdateEpisodeMD(ep.series, ep.season, ep.episode)
			if err != nil {
				fmt.Println("GOT AN ERROR UPDATING EPISODE")
			}
		}
		fmt.Println("Ending worker")
		return nil
	})

	return &manager
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

func (self *LibraryManager) UpdateEpisodeMD(tv TvSeries, season TvSeason, episode TvEpisode) error {
	fmt.Printf("Grabbing metadata for episode %s for series '%s'\n", episode.EpisodeNum, tv.Name)
	episodeInt, err := strconv.ParseInt(episode.EpisodeNum, 10, 32)
	if err != nil {
		fmt.Println("Could not parse season:", err)
	}
	fullEpisode, err := env.Tmdb.GetTvEpisodeInfo(tv.TmdbID, season.SeasonNumber, int(episodeInt), nil)
	if err == nil {
		if fullEpisode != nil {
			episode.SetUUID()
			episode.AirDate = fullEpisode.AirDate
			episode.Name = fullEpisode.Name
			episode.TmdbID = fullEpisode.ID
			episode.Overview = fullEpisode.Overview
			episode.StillPath = fullEpisode.StillPath
			obj := env.Db.Save(&episode)
			return obj.Error
		}
		return nil
	} else {
		fmt.Println("Could not grab episode information:", err)
		return err
	}
}

func (self *LibraryManager) UpdateEpisodesMD() error {
	episodes := []TvEpisode{}
	env.Db.Where("tmdb_id = ?", 0).Find(&episodes)
	for i := range episodes {
		go func(episode *TvEpisode) {
			var season TvSeason
			var tv TvSeries
			env.Db.Where("id = ?", episode.TvSeasonID).Find(&season)
			env.Db.Where("id = ?", season.TvSeriesID).Find(&tv)
			self.pool.Process(EpisodePayload{season: season, series: tv, episode: *episode})
		}(&episodes[i])
	}
	return nil
}

func (self *LibraryManager) UpdateSeasonMD() error {
	seasons := []TvSeason{}
	env.Db.Where("tmdb_id = ?", 0).Find(&seasons)
	for _, season := range seasons {
		var tv TvSeries
		env.Db.Where("id = ?", season.TvSeriesID).Find(&tv)

		fmt.Printf("Grabbing meta-data for season %d of series '%s'\n", season.SeasonNumber, tv.Name)
		fullSeason, err := env.Tmdb.GetTvSeasonInfo(tv.TmdbID, season.SeasonNumber, nil)
		if err == nil {
			season.SetUUID()
			season.AirDate = fullSeason.AirDate
			season.Overview = fullSeason.Overview
			season.Name = fullSeason.Name
			season.TmdbID = fullSeason.ID
			season.PosterPath = fullSeason.PosterPath
			env.Db.Save(&season)
		} else {
			fmt.Println("Could not grab seasonal information")
		}
	}
	return nil
}

func (self *LibraryManager) UpdateTvMD(library *Library) error {
	series := []TvSeries{}
	env.Db.Where("tmdb_id = ?", 0).Find(&series)
	for _, serie := range series {
		fmt.Println("Looking up meta-data for series:", serie.Name)
		var options = make(map[string]string)
		if serie.FirstAirYear != 0 {
			options["first_air_date_year"] = strconv.FormatUint(serie.FirstAirYear, 10)
		}
		searchRes, err := env.Tmdb.SearchTv(serie.Name, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			fmt.Println("Found Series that matches, using first result and doing deepscan.")
			tv := searchRes.Results[0] // Take the first result for now
			fullTv, err := env.Tmdb.GetTvInfo(tv.ID, nil)
			if err == nil {
				serie.Overview = fullTv.Overview
				serie.Status = fullTv.Status
				serie.Type = fullTv.Type
			} else {
				fmt.Println("Could not get full results, only adding search results. Error:", err)
			}
			serie.SetUUID()
			serie.TmdbID = tv.ID
			serie.FirstAirDate = tv.FirstAirDate
			serie.OriginalName = tv.OriginalName
			serie.BackdropPath = tv.BackdropPath
			serie.PosterPath = tv.PosterPath
			env.Db.Save(&serie)
		}
	}

	self.UpdateSeasonMD()
	self.UpdateEpisodesMD()

	return nil
}

func (self *LibraryManager) UpdateMovieMD(library *Library) error {
	movies := []Movie{}
	// Consider removing the library here as metadata is no longer tied to one library
	env.Db.Where("tmdb_id = ?", 0).Find(&movies)
	for _, movie := range movies {
		fmt.Printf("Attempting to fetch metadata for '%s'\n", movie.Title)
		var options = make(map[string]string)
		if movie.Year > 0 {
			options["year"] = movie.YearAsString()
		}
		searchRes, err := env.Tmdb.SearchMovie(movie.Title, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			fmt.Println("Found movie that matches, using first result and doing deepscan.")
			mov := searchRes.Results[0] // Take the first result for now
			fullMov, err := env.Tmdb.GetMovieInfo(mov.ID, nil)
			if err == nil {
				movie.Overview = fullMov.Overview
				movie.ImdbID = fullMov.ImdbID
			} else {
				fmt.Println("Could not get full results, only adding search results. Error:", err)
			}
			movie.SetUUID()
			movie.TmdbID = mov.ID
			movie.ReleaseDate = mov.ReleaseDate
			movie.OriginalTitle = mov.OriginalTitle
			movie.BackdropPath = mov.BackdropPath
			movie.PosterPath = mov.PosterPath
			env.Db.Save(&movie)
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
			env.Db.Where("file_path= ?", walkPath).Find(&EpisodeFile{}).Count(&count)
			if count == 0 {
				self.ProbeFile(library, walkPath)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
}
func (self *LibraryManager) AddWatcher(filePath string) {
	err := self.watcher.Add(filePath)
	if err != nil {
		fmt.Println("FSNOTIFY FAILURE:", err)
	}
}

func (self *LibraryManager) ProbeFile(library *Library, filePath string) error {
	fmt.Println("Scanning file:", filePath)
	var title string
	var year uint64
	fileInfo, err := os.Stat(filePath)

	if err != nil {
		// This catches broken symlinks
		if _, ok := err.(*os.PathError); ok {
			fmt.Println("Got an error while statting file:", err)
			return nil
		}
		return err
	}
	switch kind := library.Kind; kind {
	case MediaTypeSeries:
		var year string
		fileName := fileInfo.Name()
		// First figure out if there is a year in there, and if so parse it out.
		yearRegex := regexp.MustCompile("([\\[\\(]?((?:19[0-9]|20[01])[0-9])[\\]\\)]?)")
		ress := yearRegex.FindStringSubmatch(fileInfo.Name())
		if len(ress) > 1 {
			year = ress[2]
			fmt.Println(year)
			fileName = strings.Replace(fileName, ress[1], "", -1)
		}
		// Find out episode numbers
		seriesRe := regexp.MustCompile("^(.*)S(\\d{2})E(\\d{2})")
		res := seriesRe.FindStringSubmatch(fileName)

		if len(res) > 2 {
			yearInt, err := strconv.ParseUint(res[2], 10, 32)
			if err != nil {
				fmt.Println("Could not parse year:", err)
			}

			title := helpers.Sanitize(res[1])
			season := res[2]
			episode := res[3]
			fmt.Printf("Found '%s' season %s episode %s\n", title, season, episode)
			mi := MediaItem{
				FileName:  fileInfo.Name(),
				FilePath:  filePath,
				Size:      fileInfo.Size(),
				Title:     title,
				LibraryID: library.ID,
				Year:      yearInt,
			}
			mi.SetUUID()
			var tv TvSeries
			var tvs TvSeason

			seasonInt, err := strconv.ParseInt(season, 10, 32)
			if err != nil {
				fmt.Println("Could not parse season:", err)
			}

			env.Db.FirstOrCreate(&tv, TvSeries{Name: title})
			newSeason := TvSeason{TvSeriesID: tv.ID, SeasonNumber: int(seasonInt)}
			env.Db.FirstOrCreate(&tvs, newSeason)
			fmt.Println("Found/created:", tvs)

			ep := TvEpisode{SeasonNum: season, EpisodeNum: episode, TvSeasonID: tvs.ID}
			env.Db.FirstOrCreate(&ep, ep)

			epFile := EpisodeFile{MediaItem: mi, TvEpisodeID: ep.ID}

			env.Db.FirstOrCreate(&epFile, epFile)
		}
	case MediaTypeMovie:

		movieRe := regexp.MustCompile("(.*)\\((\\d{4})\\)")
		res := movieRe.FindStringSubmatch(fileInfo.Name())

		if len(res) > 1 {
			title = helpers.Sanitize(res[1])
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
			title, yearStr = helpers.HeavySanitize(name)
			year, err = strconv.ParseUint(yearStr, 10, 32)
			if err != nil {
				fmt.Println("Could not parse year:", err)
			}
			fmt.Println("attempted to find some stuff", title, year)
		}

		// Create a movie stub so the metadata can get to work on it after probing
		movie := Movie{Title: title, Year: year}
		env.Db.FirstOrCreate(&movie, movie)

		mi := MediaItem{
			FileName:  fileInfo.Name(),
			FilePath:  filePath,
			Size:      fileInfo.Size(),
			Title:     title,
			Year:      year,
			LibraryID: library.ID,
		}
		mi.SetUUID()

		movieFile := MovieFile{MediaItem: mi, MovieID: movie.ID}

		fmt.Println(movieFile.String())
		env.Db.FirstOrCreate(&movieFile, movieFile)

	}
	return nil
}

func (self *LibraryManager) ProbeMovies(library *Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(walkPath)] {
			self.AddWatcher(walkPath)
			self.AddWatcher(filepath.Dir(walkPath))

			count := 0
			env.Db.Where("file_path= ?", walkPath).Find(&MovieFile{}).Count(&count)
			if count == 0 {
				self.ProbeFile(library, walkPath)
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

func (self *LibraryManager) RefreshAll() {
	for _, lib := range AllLibraries() {
		self.AddWatcher(lib.FilePath)

		fmt.Println("Scanning library:", lib.Name, lib.FilePath)
		self.Probe(&lib)

		fmt.Println("Updating metadata for library:", lib.Name)
		self.UpdateMD(&lib)
	}
}
