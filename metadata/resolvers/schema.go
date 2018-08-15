package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

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
		season(uuid: String!): Season!
		episode(uuid: String!): Episode
		users(): [User]!
		recentlyAdded(): [MediaItem]
		upNext(): [MediaItem]
		search(name: String!): [SearchItem]
		invites(): [Invite]
	}

	type Mutation {
		# Tell the application to index all the supported files in the given directory.
		# 'kind' can be 0 for movies and 1 for series.
		createLibrary(name: String!, file_path: String!, kind: Int!): LibraryResponse!

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
		file_path: String!

		movies: [Movie]!
		episodes: [Episode]!
	}

	type Series {
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

	type Season {
		name: String!
		overview: String!
		season_number: Int!
		air_date: String!
		poster_path: String!
		tmdb_id: Int!
		episodes: [Episode]!
		uuid: String!
	}

	type Episode {
		name: String!
		overview: String!
		still_path: String!
		air_date: String!
		episode_number: Int!
		tmdb_id: Int!
		uuid: String!
		files: [EpisodeFile]!
		play_state: PlayState
	}

	type EpisodeFile {
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
		uuid: String!
		streams: [Stream]!
	}

	type Stream {
	  # Name of the codec used for encoding
	  codec_name: String
	  # Mimetype for the codec
	  codec_mime: String
	  # Encoding profile used for codec
	  profile: String
	  # Stream bitrate (not file)
	  bit_rate: Int
	  # Type of stream can be either 'video', 'audio' or 'subtitle'
	  stream_type: String
	  # Language used for audio or subtitle types
	  language: String
	  # Title for audio and subtitle streams
	  title: String
	  # Title for audio and subtitle streams
	  resolution: String
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

	type MovieFile {
		# Filename
		file_name: String!
		# Absolute path to the filesystem
		file_path: String!
		library_id: Int!
		uuid: String!
		# Stream information (subtitles / audio and video streams)
		streams: [Stream]!
	}

	# Invite that can be used to allow other users access to your server.
	type Invite {
		code: String
		user: User
	}

`

func InitSchema(env *db.MetadataContext) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, &Resolver{env: env})
	return Schema
}
