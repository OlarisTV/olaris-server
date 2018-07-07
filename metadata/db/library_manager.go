package db

import (
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/fsnotify/fsnotify"
	"gitlab.com/bytesized/bytesized-streaming/metadata/parsers"
	"os"
	"path/filepath"
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
	series  Series
	season  Season
	episode Episode
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
			fmt.Printf("Worker in '%s', S%vE%v\n", ep.series.Name, ep.season.SeasonNumber, ep.episode.EpisodeNum)
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
func (self *LibraryManager) UpdateEpisodeMD(tv Series, season Season, episode Episode) error {
	fmt.Printf("Grabbing metadata for episode %v for series '%v'\n", episode.EpisodeNum, tv.Name)
	fullEpisode, err := env.Tmdb.GetTvEpisodeInfo(tv.TmdbID, season.SeasonNumber, episode.EpisodeNum, nil)
	if err == nil {
		if fullEpisode != nil {
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
	episodes := []Episode{}
	env.Db.Where("tmdb_id = ?", 0).Find(&episodes)
	for i := range episodes {
		go func(episode *Episode) {
			var season Season
			var tv Series
			env.Db.Where("id = ?", episode.SeasonID).Find(&season)
			env.Db.Where("id = ?", season.SeriesID).Find(&tv)
			self.pool.Process(EpisodePayload{season: season, series: tv, episode: *episode})
		}(&episodes[i])
	}
	return nil
}

func (self *LibraryManager) UpdateSeasonMD() error {
	seasons := []Season{}
	env.Db.Where("tmdb_id = ?", 0).Find(&seasons)
	for _, season := range seasons {
		var tv Series
		env.Db.Where("id = ?", season.SeriesID).Find(&tv)

		fmt.Printf("Grabbing meta-data for season %d of series '%s'\n", season.SeasonNumber, tv.Name)
		fullSeason, err := env.Tmdb.GetTvSeasonInfo(tv.TmdbID, season.SeasonNumber, nil)
		if err == nil {
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
	series := []Series{}
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
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		// This catches broken symlinks
		if _, ok := err.(*os.PathError); ok {
			fmt.Println("Got an error while statting file:", err)
			return nil
		}
		return err
	}
	basename := fileInfo.Name()
	name := strings.TrimSuffix(basename, filepath.Ext(basename))

	switch kind := library.Kind; kind {
	case MediaTypeSeries:
		parsedInfo := parsers.ParseSerieName(name)
		if parsedInfo.SeasonNum != 0 && parsedInfo.EpisodeNum != 0 {
			mi := MediaItem{
				FileName:  name,
				FilePath:  filePath,
				Size:      fileInfo.Size(),
				Title:     parsedInfo.Title,
				LibraryID: library.ID,
				Year:      parsedInfo.Year,
			}
			var tv Series
			var tvs Season

			env.Db.FirstOrCreate(&tv, Series{Name: parsedInfo.Title})
			newSeason := Season{SeriesID: tv.ID, SeasonNumber: parsedInfo.SeasonNum}
			env.Db.FirstOrCreate(&tvs, newSeason)

			ep := Episode{SeasonNum: parsedInfo.SeasonNum, EpisodeNum: parsedInfo.EpisodeNum, SeasonID: tvs.ID}
			env.Db.FirstOrCreate(&ep, ep)

			epFile := EpisodeFile{MediaItem: mi, EpisodeID: ep.ID}
			epFile.Streams = CollectStreams(filePath)

			// TODO(Maran) We might be adding double files in case it already exist
			env.Db.Save(&epFile)
		} else {
			fmt.Printf("Could not discover enough information about %s to add it to the library\n", parsedInfo.Title)
		}

	case MediaTypeMovie:
		mvi := parsers.ParseMovieName(name)
		// Create a movie stub so the metadata can get to work on it after probing
		movie := Movie{Title: mvi.Title, Year: mvi.Year}
		env.Db.FirstOrCreate(&movie, movie)

		mi := MediaItem{
			FileName:  name,
			FilePath:  filePath,
			Size:      fileInfo.Size(),
			Title:     mvi.Title,
			Year:      mvi.Year,
			LibraryID: library.ID,
		}

		movieFile := MovieFile{MediaItem: mi, MovieID: movie.ID}
		movieFile.Streams = CollectStreams(filePath)
		env.Db.Save(&movieFile)

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
