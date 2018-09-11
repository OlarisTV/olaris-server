package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"gitlab.com/olaris/olaris-server/metadata/app"
)

// SchemaTxt defines the graphql schema.
var SchemaTxt = `
	schema {
		query: Query
		mutation: Mutation
	}

	union MediaItem = Movie | Episode
	union SearchItem = Movie | Series

	# The query type, represents all of the entry points into our object graph
	type Query {
		movies(uuid: String): [Movie]!
		libraries(): [Library]!
		series(uuid: String): [Series]!
		season(uuid: String): Season!
		episode(uuid: String): Episode
		users(): [User]!
		recentlyAdded(): [MediaItem]
		upNext(): [MediaItem]
		search(name: String!): [SearchItem]
		invites(): [Invite]
	}

	type Mutation {
		# Tell the application to index all the supported files in the given directory.
		# 'kind' can be 0 for movies and 1 for series.
		createLibrary(name: String!, filePath: String!, kind: Int!): LibraryResponse!

		# Delete a library and remove all collected metadata.
		deleteLibrary(id: Int!): LibraryResponse!

		# Create a invite code so a user can register on the server
		createUserInvite(): UserInviteResponse!

		# Create a playstate for the given media item
		createPlayState(uuid: String!, finished: Boolean!, playtime: Float!): CreatePSResponse!

		# Request permission to play a certain file
		createStreamingTicket(uuid: String!): CreateSTResponse!

		# Delete a user from the database, please note that the user will be able to username until the JWT expires.
		deleteUser(id: Int!): UserResponse!


	}


	type LibraryResponse {
		library: Library
		error: Error
	}

	type UserResponse{
		user: User
		error: Error
	}

	type UserInviteResponse {
		code: String!
		error: Error
	}

	type CreatePSResponse {
		success: Boolean!
	}

	type CreateSTResponse {
		error: Error
		# Path with a JWT that will stream your file.
		streamingPath: String!
		jwt: String!
	}

	type Error {
		message: String!
		hasError: Boolean!
	}

	type User {
		id: Int!
		username: String!
		admin: Boolean!
	}

	type PlayState {
		finished: Boolean!
		playtime: Float!
	}

	# A media library
	type Library {
		# Primary ID
		id: Int!

		# Library Type (0 - movies)
		kind: Int!

		# Human readable name of the Library
		name: String!

		# Path that this library manages
		filePath: String!

		movies: [Movie]!
		episodes: [Episode]!
	}

	type Series {
		name: String!
		overview: String!
		firstAirDate: String!
		status: String!
		seasons: [Season]!
		backdropPath: String!
		posterPath: String!
		tmdbID: Int!
		type: String!
		uuid: String!
		unwatchedEpisodesCount: Int!
	}

	type Season {
		name: String!
		overview: String!
		seasonNumber: Int!
		airDate: String!
		posterPath: String!
		tmdbID: Int!
		episodes: [Episode]!
		uuid: String!
		unwatchedEpisodesCount: Int!
		series: Series
	}

	type Episode {
		name: String!
		overview: String!
		stillPath: String!
		airDate: String!
		episodeNumber: Int!
		tmdbID: Int!
		uuid: String!
		files: [EpisodeFile]!
		playState: PlayState
		season: Season
	}

	type EpisodeFile {
		# Filename
		fileName: String!
		# Absolute path to the filesystem
		filePath: String!
		uuid: String!
		streams: [Stream]!
		# Total duration of the first video stream in seconds
		totalDuration: Float
	}

	type Stream {
	  # Name of the codec used for encoding
	  codecName: String
	  # Mimetype for the codec
	  codecMime: String
	  # Encoding profile used for codec
	  profile: String
	  # Stream bitrate (not file)
	  bitRate: Int
	  # Type of stream can be either 'video', 'audio' or 'subtitle'
	  streamType: String
	  # Language used for audio or subtitle types
	  language: String
	  # Title for audio and subtitle streams
	  title: String
	  # Title for audio and subtitle streams
	  resolution: String
	  # Total duration of the stream in seconds
	  totalDuration: Float
	}

	# A movie file
	type Movie {
		# Official title according to the MovieDB
		name: String!
		# Title based on parsed filename
		title: String!
		# Release year
		year: String!
		# Short description of the movie
		overview: String!
		# IMDB ID
		imdbID: String!
		# TMDB ID
		tmdbID: Int!
		# ID to retrieve backdrop
		backdropPath: String!
		# ID to retrieve poster
		posterPath: String!
		uuid: String!
		files: [MovieFile]!
		playState: PlayState
	}

	type MovieFile {
		# Filename
		fileName: String!
		# Absolute path to the filesystem
		filePath: String!
		libraryId: Int!
		uuid: String!
		# Stream information (subtitles / audio and video streams)
		streams: [Stream]!
		# Total duration of the first video stream in seconds
		totalDuration: Float
	}

	# Invite that can be used to allow other users access to your server.
	type Invite {
		code: String
		user: User
	}

`

// InitSchema inits the graphql schema.
func InitSchema(env *app.MetadataContext) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, &Resolver{env: env})
	return Schema
}
