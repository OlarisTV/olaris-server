package metadata

import (
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jinzhu/gorm"
	"net/http"
)

var SchemaTxt = `
	schema {
		query: Query
		mutation: Mutation
	}
	# The query type, represents all of the entry points into our object graph
	type Query {
		movies(): [Movie]!
		libraries(): [Library]!
		tvseries(): [TvSeries]!
	}

	type Mutation {
		# Add a library to scan
		createLibrary(name: String!, file_path: String!): LibRes!
	}

	interface LibRes {
		library: Library!
		error: Error
	}

	interface Error {
		message: String!
		hasError: Boolean!
	}

	# A media library
	interface Library {
		# Library Type (0 - movies)
		kind: Int!
		# Human readable name of the Library
		name: String!
		# Path that this library manages
		file_path: String!
		movies: [Movie]!
		episodes: [Episode]!
	}

	interface TvSeries {
		name: String!
		overview: String!
		first_air_date: String!
		status: String!
		seasons: [Season]!
		backdrop_path: String!
		poster_path: String!
		tmdb_id: Int!
		type: String!
	}

	interface Season {
		name: String!
		overview: String!
		season_number: Int!
		air_date: String!
		poster_path: String!
		tmdb_id: Int!
		episodes: [Episode]!
	}

	interface Episode {
		name: String!
		overview: String!
		still_path: String!
		air_date: String!
		tmdb_id: Int!
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
	}

	# A movie file
	interface Movie {
		# Title of the movie
		title: String!
		# Official Title
		original_title: String!
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
		# Release year
		year: String!
		# Library ID
		library_id: Int!
		# Short description of the movie
		overview: String!
		# IMDB ID
		imdb_id: String!
		# TMDB ID
		tmdb_id: Int!
	}
`

func InitSchema(ctx *MetadataContext) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, &Resolver{db: ctx.Db, ctx: ctx})
	return Schema
}

type Resolver struct {
	ctx *MetadataContext
	db  *gorm.DB
}

func (r *Resolver) Libraries() []*libraryResolver {
	var l []*libraryResolver
	var libraries []Library
	r.db.Find(&libraries)
	for _, library := range libraries {
		var movies []MovieItem
		var mr []*movieResolver
		r.db.Where("library_id = ?", library.ID).Find(&movies)
		for _, movie := range movies {
			if movie.Title != "" {
				mov := movieResolver{r: movie}
				mr = append(mr, &mov)
			}
		}
		library.Movies = mr

		var episodes []TvEpisode
		r.ctx.Db.Where("library_id =?", library.ID).Find(&episodes)
		for _, episode := range episodes {
			library.Episodes = append(library.Episodes, &episodeResolver{r: episode})
		}

		lib := libraryResolver{r: library}
		l = append(l, &lib)
	}
	return l
}

func (r *Resolver) Movies() []*movieResolver {
	var l []*movieResolver
	var movies []MovieItem
	r.db.Find(&movies)
	for _, movie := range movies {
		if movie.Title != "" {
			mov := movieResolver{r: movie}
			l = append(l, &mov)
		}
	}
	return l
}

func (r *Resolver) TvSeries() []*tvSeriesResolver {
	var resolvers []*tvSeriesResolver
	var series []TvSeries
	r.ctx.Db.Find(&series)
	for _, serie := range series {
		var seasons []TvSeason
		fmt.Println("Looking for seasons for:", serie.ID)
		r.ctx.Db.Where("tv_series_id = ?", serie.ID).Find(&seasons)
		for _, season := range seasons {
			var episodes []TvEpisode
			r.ctx.Db.Where("tv_season_id = ?", season.ID).Find(&episodes)
			for _, episode := range episodes {
				season.EpisodeResolvers = append(season.EpisodeResolvers, &episodeResolver{r: episode})
			}
			serie.SeasonResolvers = append(serie.SeasonResolvers, &seasonResolver{r: season})
		}
		resolvers = append(resolvers, &tvSeriesResolver{r: serie})

	}
	return resolvers
}

type tvSeriesResolver struct {
	r TvSeries
}

func (r *tvSeriesResolver) Name() string {
	return r.r.Name
}
func (r *tvSeriesResolver) Overview() string {
	return r.r.Overview
}
func (r *tvSeriesResolver) FirstAirDate() string {
	return r.r.FirstAirDate
}
func (r *tvSeriesResolver) Status() string {
	return r.r.Status
}
func (r *tvSeriesResolver) Type() string {
	return r.r.Type
}
func (r *tvSeriesResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *tvSeriesResolver) BackdropPath() string {
	return r.r.BackdropPath
}
func (r *tvSeriesResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}
func (r *tvSeriesResolver) Seasons() []*seasonResolver {
	return r.r.SeasonResolvers
}

type seasonResolver struct {
	r TvSeason
}

func (r *seasonResolver) Name() string {
	return r.r.Name
}

func (r *seasonResolver) Overview() string {
	return r.r.Overview
}
func (r *seasonResolver) AirDate() string {
	return r.r.AirDate
}
func (r *seasonResolver) PosterPath() string {
	return r.r.PosterPath
}
func (r *seasonResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

func (r *seasonResolver) SeasonNumber() int32 {
	return int32(r.r.SeasonNumber)
}
func (r *seasonResolver) Episodes() []*episodeResolver {
	return r.r.EpisodeResolvers
}

type episodeResolver struct {
	r TvEpisode
}

func (r *episodeResolver) Name() string {
	return r.r.Name
}

func (r *episodeResolver) Overview() string {
	return r.r.Overview
}
func (r *episodeResolver) AirDate() string {
	return r.r.AirDate
}
func (r *episodeResolver) StillPath() string {
	return r.r.StillPath
}
func (r *episodeResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}
func (r *episodeResolver) FilePath() string {
	return r.r.FilePath
}
func (r *episodeResolver) FileName() string {
	return r.r.FileName
}

type movieResolver struct {
	r MovieItem
}

func (r *movieResolver) Title() string {
	return r.r.Title
}
func (r *movieResolver) OriginalTitle() string {
	return r.r.OriginalTitle
}
func (r *movieResolver) FilePath() string {
	return r.r.FilePath
}
func (r *movieResolver) FileName() string {
	return r.r.FileName
}
func (r *movieResolver) Year() string {
	return r.r.YearAsString()
}
func (r *movieResolver) Overview() string {
	return r.r.Overview
}
func (r *movieResolver) ImdbID() string {
	return r.r.ImdbID
}
func (r *movieResolver) TmdbID() int32 {
	return int32(r.r.TmdbID)
}

// Will this be a problem if we ever run out of the 32int space?
func (r *movieResolver) LibraryID() int32 {
	return int32(r.r.LibraryID)
}

type libraryResolver struct {
	r Library
}

func (r *libraryResolver) Name() string {
	return r.r.Name
}
func (r *libraryResolver) Movies() []*movieResolver {
	return r.r.Movies
}
func (r *libraryResolver) Episodes() []*episodeResolver {
	return r.r.Episodes
}
func (r *libraryResolver) FilePath() string {
	return r.r.FilePath
}
func (r *libraryResolver) Kind() int32 {
	return 0
}

func (r *Resolver) CreateLibrary(args *struct {
	Name     string
	FilePath string
}) *libResResolv {
	library := Library{
		Name:     args.Name,
		FilePath: args.FilePath,
	}
	r.db.Create(&library)
	r.ctx.RefreshChan <- 1
	libRes := LibRes{Error: &errorResolver{Error{hasError: false}}, Library: &libraryResolver{library}}
	return &libResResolv{libRes}
}

type libResResolv struct {
	r LibRes
}

func (r *libResResolv) Library() *libraryResolver {
	return r.r.Library
}
func (r *libResResolv) Error() *errorResolver {
	return r.r.Error
}

type errorResolver struct {
	r Error
}

type LibRes struct {
	Error   *errorResolver
	Library *libraryResolver
}

type Error struct {
	message  string
	hasError bool
}

func (r *errorResolver) Message() string {
	return r.r.message
}
func (r *errorResolver) HasError() bool {
	return r.r.hasError
}

//mutation AddLibrary($name: String!, $file_path: String!) {
//	createLibrary(name: $name, file_path: $file_path){
//    library{
//      name
//    }
//    error{
//      hasError
//      message
//    }
//  }
//}

func GraphiQLHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(graphiQLpage)
}

func NewRelayHandler(ctx *MetadataContext) *relay.Handler {
	schema := InitSchema(ctx)
	return &relay.Handler{Schema: schema}
}
