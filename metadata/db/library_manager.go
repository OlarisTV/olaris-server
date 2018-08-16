package db

import (
	"github.com/Jeffail/tunny"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata/parsers"
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

// Defines various mediatypes, only Movie and Series support atm.
const (
	MediaTypeMovie = iota
	MediaTypeSeries
	MediaTypeMusic
	MediaTypeOtherMovie
)

// LibraryManager manages all active libraries.
type LibraryManager struct {
	pool    *tunny.Pool
	watcher *fsnotify.Watcher
}

type episodePayload struct {
	series  Series
	season  Season
	episode Episode
}

// NewLibraryManager creates a new LibraryManager with a pool worker that can process episode information.
func NewLibraryManager(watcher *fsnotify.Watcher) *LibraryManager {
	manager := LibraryManager{}
	if watcher != nil {
		manager.watcher = watcher
	}
	// The MovieDB currently has a 40 requests per 10 seconds limit. Assuming every request takes a second then four workers is probably ideal.
	manager.pool = tunny.NewFunc(4, func(payload interface{}) interface{} {
		log.Debugln("Spawning agent worker.")
		ep, ok := payload.(episodePayload)
		if ok {
			log.Debugf("Worker in '%s', S%vE%v", ep.series.Name, ep.season.SeasonNumber, ep.episode.EpisodeNum)
			err := manager.UpdateEpisodeMD(ep.series, ep.season, ep.episode)
			if err != nil {
				log.Warnln("Got an error updating metadata for series:", err)
			}
		}
		log.Debugln("Agent worker finished.")
		return nil
	})

	return &manager
}

// UpdateMD looks for missing metadata information and attempts to retrieve it.
func (man *LibraryManager) UpdateMD(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		log.Println("Updating metadata for movies.")
		man.UpdateMovieMD(library)
	case MediaTypeSeries:
		log.Println("Updating metadata for TV.")
		man.UpdateTvMD(library)
	}
}

// UpdateEpisodeMD looks for missing episode metadata and attempts to retrieve it.
func (man *LibraryManager) UpdateEpisodeMD(tv Series, season Season, episode Episode) error {
	log.Debugf("Grabbing metadata for episode %v for series '%v'", episode.EpisodeNum, tv.Name)
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
	}
	log.Warnln("Could not grab episode information:", err)
	return err
}

// UpdateEpisodesMD loops over all episode with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateEpisodesMD() error {
	episodes := []Episode{}
	env.Db.Where("tmdb_id = ?", 0).Find(&episodes)
	for i := range episodes {
		go func(episode *Episode) {
			var season Season
			var tv Series
			env.Db.Where("id = ?", episode.SeasonID).Find(&season)
			env.Db.Where("id = ?", season.SeriesID).Find(&tv)
			man.pool.Process(episodePayload{season: season, series: tv, episode: *episode})
		}(&episodes[i])
	}
	return nil
}

// UpdateSeasonMD loops over all seasons with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateSeasonMD() error {
	seasons := []Season{}
	env.Db.Where("tmdb_id = ?", 0).Find(&seasons)
	for _, season := range seasons {
		var tv Series
		env.Db.Where("id = ?", season.SeriesID).Find(&tv)

		log.Debugf("Grabbing metadata for season %d of series '%s'", season.SeasonNumber, tv.Name)
		fullSeason, err := env.Tmdb.GetTvSeasonInfo(tv.TmdbID, season.SeasonNumber, nil)
		if err == nil {
			season.AirDate = fullSeason.AirDate
			season.Overview = fullSeason.Overview
			season.Name = fullSeason.Name
			season.TmdbID = fullSeason.ID
			season.PosterPath = fullSeason.PosterPath
			env.Db.Save(&season)
		} else {
			log.Warnln("Could not grab seasonal information:", err)
		}
	}
	return nil
}

// UpdateTvMD loops over all series with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateTvMD(library *Library) error {
	series := []Series{}
	env.Db.Where("tmdb_id = ?", 0).Find(&series)
	for _, serie := range series {
		log.Debugln("Looking up meta-data for series:", serie.Name)
		var options = make(map[string]string)
		if serie.FirstAirYear != 0 {
			options["first_air_date_year"] = strconv.FormatUint(serie.FirstAirYear, 10)
		}
		searchRes, err := env.Tmdb.SearchTv(serie.Name, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			log.Debugln("Found Series that matches, using first result and doing deepscan.")
			tv := searchRes.Results[0] // Take the first result for now
			fullTv, err := env.Tmdb.GetTvInfo(tv.ID, nil)
			if err == nil {
				serie.Overview = fullTv.Overview
				serie.Status = fullTv.Status
				serie.Type = fullTv.Type
			} else {
				log.Warnln("Could not get full results, only adding search results. Error:", err)
			}
			serie.TmdbID = tv.ID
			serie.FirstAirDate = tv.FirstAirDate
			serie.OriginalName = tv.OriginalName
			serie.BackdropPath = tv.BackdropPath
			serie.PosterPath = tv.PosterPath
			env.Db.Save(&serie)
		}
	}

	man.UpdateSeasonMD()
	man.UpdateEpisodesMD()

	return nil
}

// UpdateMovieMD loops over all movies with no tmdb information yet and attempts to retrieve the metadata.
func (man *LibraryManager) UpdateMovieMD(library *Library) error {
	movies := []Movie{}
	// Consider removing the library here as metadata is no longer tied to one library
	env.Db.Where("tmdb_id = ?", 0).Find(&movies)
	for _, movie := range movies {
		log.Printf("Attempting to fetch metadata for '%s' as no metadata exist yet.", movie.Title)
		var options = make(map[string]string)
		if movie.Year > 0 {
			options["year"] = movie.YearAsString()
		}
		searchRes, err := env.Tmdb.SearchMovie(movie.Title, options)

		if err != nil {
			return err
		}

		if len(searchRes.Results) > 0 {
			log.Debugln("Found movie that matches, using first result and doing deepscan.")
			mov := searchRes.Results[0] // Take the first result for now
			fullMov, err := env.Tmdb.GetMovieInfo(mov.ID, nil)
			if err == nil {
				movie.Overview = fullMov.Overview
				movie.ImdbID = fullMov.ImdbID
			} else {
				log.Warnln("Could not get full results, only adding search results. Error:", err)
			}
			movie.TmdbID = mov.ID
			movie.ReleaseDate = mov.ReleaseDate
			movie.OriginalTitle = mov.OriginalTitle
			movie.BackdropPath = mov.BackdropPath
			movie.PosterPath = mov.PosterPath
			env.Db.Save(&movie)
		} else {
			log.Warnln("Could not find any valid metadata for this file.")
		}

	}
	return nil
}

// Probe goes over the filesystem and parses filenames in the given library.
func (man *LibraryManager) Probe(library *Library) {
	switch kind := library.Kind; kind {
	case MediaTypeMovie:
		log.Println("Probing files for movie information.")
		man.ProbeMovies(library)
	case MediaTypeSeries:
		log.Println("Probing files for series information.")
		man.ProbeSeries(library)
	}
}

// ProbeSeries goes over the given library and attempts to get series information from filenames.
func (man *LibraryManager) ProbeSeries(library *Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {
		if supportedExtensions[filepath.Ext(walkPath)] {
			man.AddWatcher(walkPath)
			man.AddWatcher(filepath.Dir(walkPath))

			count := 0
			env.Db.Where("file_path= ?", walkPath).Find(&EpisodeFile{}).Count(&count)
			if count == 0 {
				man.ProbeFile(library, walkPath)
			}
		}
		return nil
	})
	if err != nil {
		log.Warnln("Error probing series:", err)
	}
}

// AddWatcher adds a fsnotify watcher to the given path.
func (man *LibraryManager) AddWatcher(filePath string) {
	err := man.watcher.Add(filePath)
	if err != nil {
		log.Warnln("FSNOTIFY FAILURE:", err)
	}
}

// ProbeFile goes over the given file and tries to attempt to find out more information based on the filename.
func (man *LibraryManager) ProbeFile(library *Library, filePath string) error {
	log.Debugln("Parsing filename:", filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		// This catches broken symlinks
		if _, ok := err.(*os.PathError); ok {
			log.Warnln("Got an error while statting file:", err)
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
			log.Warnf("Could not discover enough information about %s to add it to the library.", parsedInfo.Title)
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

// ProbeMovies goes over the given library and attempts to get movie information from filenames.
func (man *LibraryManager) ProbeMovies(library *Library) {
	err := filepath.Walk(library.FilePath, func(walkPath string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(walkPath)] {
			man.AddWatcher(walkPath)
			man.AddWatcher(filepath.Dir(walkPath))

			count := 0
			env.Db.Where("file_path= ?", walkPath).Find(&MovieFile{}).Count(&count)
			if count == 0 {
				man.ProbeFile(library, walkPath)
			} else {
				log.Debugf("File '%s' already exists in library, not adding again.", walkPath)
			}
		}

		return nil
	})

	if err != nil {
		log.Warnln("Could not probe all movies:", err)
		return
	}
}

// CleanUpMovieMD checks whether there is stale movie information in the database and removes it.
func (man *LibraryManager) CleanUpMovieMD(movieFile MovieFile) {
	// Delete all stream information
	env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'movies'", &movieFile.ID)

	env.Db.Where("id = ?", movieFile.MovieID).Find(&movieFile.Movie)

	// Delete all PlayState information
	env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'movies'", movieFile.MovieID)

	// Delete all file information
	env.Db.Delete(&movieFile)

	// Delete movie
	env.Db.Delete(&movieFile.Movie)
}

// CleanUpSeriesMD checks whether there is stale series information in the database and removes it.
func (man *LibraryManager) CleanUpSeriesMD(episodeFile EpisodeFile) {
	// Check there are no other copies of the episode around.
	if episodeFile.IsSingleFile() {
		// Delete all stream information
		env.Db.Delete(Stream{}, "owner_id = ? AND owner_type = 'episode_files'", &episodeFile.ID)

		// Delete all PlayState information
		env.Db.Delete(PlayState{}, "owner_id = ? AND owner_type = 'episode_files'", episodeFile.EpisodeID)

		var episode Episode
		env.Db.First(&episode, episodeFile.EpisodeID)

		// Delete all file information
		env.Db.Delete(&episodeFile)

		// Delete Episode
		env.Db.Delete(&episode)

		count := 0
		var season Season
		env.Db.First(&season, episode.SeasonID)

		env.Db.Model(Episode{}).Where("season_id = ?", season.ID).Count(&count)

		// If there are no more episodes to this season, delete the season.
		if count == 0 {
			env.Db.Delete(Season{}, "id = ?", episode.SeasonID)
		}

		// If there are no more seasons to this series, delete it.
		count = 0
		env.Db.Model(Season{}).Where("series_id = ?", season.SeriesID).Count(&count)
		if count == 0 {
			env.Db.Delete(Series{}, "id = ?", season.SeriesID)
		}
	}
}

// CheckRemovedFiles checks all files in the database to ensure they still exist, if not it attempts to remove the MD information from the db.
func (man *LibraryManager) CheckRemovedFiles() {
	log.Infoln("Checking filesystem to see if any files got removed since our last scan.")
	for _, movieFile := range FindAllMovieFiles() {
		log.Debugf("Checking path '%s'", movieFile.FilePath)
		if !helpers.FileExists(movieFile.FilePath) {
			log.Debugln("Missing file, cleaning up MD", movieFile.FileName)
			movieFile.DeleteSelfAndMD()
		}
	}

	for _, file := range FindAllEpisodeFiles() {
		log.Debugln("Checking if episode exists:", file.FileName)
		if !helpers.FileExists(file.FilePath) {
			log.Debugln("Missing file, cleaning up MD", file.FileName)
			file.DeleteSelfAndMD()
		}
	}
}

// RefreshAll rescans all files and attempts to find missing metadata information.
func (man *LibraryManager) RefreshAll() {
	for _, lib := range AllLibraries() {
		man.AddWatcher(lib.FilePath)

		man.CheckRemovedFiles()

		log.Printf("Scanning library folder '%s' (%s) for media files.", lib.Name, lib.FilePath)
		man.Probe(&lib)

		log.Printf("Updating metadata for library '%s'", lib.Name)
		man.UpdateMD(&lib)

	}
}
