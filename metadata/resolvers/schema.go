package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
)

var SchemaTxt = `
	schema {
		query: Query
		mutation: Mutation
	}
	# The query type, represents all of the entry points into our object graph
	type Query {
		movies(uuid: String): [Movie]!
		libraries(): [Library]!
		tvseries(uuid: String): [TvSeries]!
		tvseason(uuid: String!): Season!
		tvepisode(uuid: String!): Episode!
		users(): [User]!
	}

	type Mutation {
		# Add a library to scan
		createLibrary(name: String!, file_path: String!, kind: Int!): LibRes!
		createUser(login: String!, password: String!, admin: Boolean!): CreateUserResponse!
		createPlayState(uuid: String!, finished: Boolean!, playtime: Float!): CreatePSResponse!
		createStreamingTicket(uuid: String!): CreateSTResponse!
	}

	interface LibRes {
		library: Library!
		error: Error
	}

	interface CreateUserResponse {
		user: User!
		error: Error
	}

	interface CreatePSResponse {
		success: Boolean!
	}

	interface CreateSTResponse {
		error: Error
		# Path with a JWT that will stream your file.
		streamingPath: String!
	}

	interface Error {
		message: String!
		hasError: Boolean!
	}

	interface User {
		login: String!
		admin: Boolean!
	}

	interface PlayState {
		finished: Boolean!
		playtime: Float!
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
		uuid: String!
	}

	interface Season {
		name: String!
		overview: String!
		season_number: Int!
		air_date: String!
		poster_path: String!
		tmdb_id: Int!
		episodes: [Episode]!
		uuid: String!
	}

	interface Episode {
		name: String!
		overview: String!
		still_path: String!
		air_date: String!
		episode_number: String!
		tmdb_id: Int!
		uuid: String!
		files: [EpisodeFile]!
		play_state: PlayState
	}

	interface EpisodeFile {
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
		uuid: String!
	}

	# A movie file
	interface Movie {
		# Title of the movie
		title: String!
		# Official Title
		original_title: String!
		# Release year
		year: String!
		# Short description of the movie
		overview: String!
		# IMDB ID
		imdb_id: String!
		# TMDB ID
		tmdb_id: Int!
		# ID to retrieve backdrop
		backdrop_path: String!
		# ID to retrieve poster
		poster_path: String!
		uuid: String!
		files: [MovieFile]!
		play_state: PlayState
	}

	interface MovieFile {
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
		# Library ID
		library_id: Int!
		uuid: String!
	}

`

func InitSchema(env *db.MetadataContext) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, &Resolver{env: env})
	return Schema
}
