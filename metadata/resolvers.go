package metadata

import (
	"fmt"
	"github.com/graph-gophers/graphql-go"
	"github.com/jinzhu/gorm"
)

var SchemaTxt = `
	schema {
		query: Query
		mutation: Mutation
	}
	# The query type, represents all of the entry points into our object graph
	type Query {
		movies(): [Movie]!
		libraries(): [Library]!  }

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

func InitSchema(db *gorm.DB) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, &Resolver{db: db})
	return Schema
}

type Resolver struct {
	db *gorm.DB
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
			fmt.Println("Adding movie:", movie.Title)
			l = append(l, &mov)
		}
	}
	return l
}

type movieResolver struct {
	r MovieItem
}

func (r *movieResolver) Title() string {
	fmt.Println(r.r)
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
