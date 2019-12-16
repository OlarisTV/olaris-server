package resolvers

import (
	"github.com/graph-gophers/graphql-go"
	"gitlab.com/olaris/olaris-server/metadata/app"
)

// SchemaTxt defines the graphql schema.
var SchemaTxt = `
	schema {
		query: Query
		subscription: Subscription
		mutation: Mutation
	}

	union MediaItem = Movie | Episode
	union SearchItem = Movie | Series

	type Subscription {
		movieAdded(): MovieAddedEvent!
		episodeAdded(): EpisodeAddedEvent!
		seriesAdded(): SeriesAddedEvent!
		seasonAdded(): SeasonAddedEvent!
	}

	# The query type, represents all of the entry points into our object graph
	type Query {
		movies(uuid: String, offset: Int, limit: Int): [Movie]!
		libraries(): [Library]!
		series(uuid: String, offset: Int, limit: Int): [Series]!
		season(uuid: String): Season!
		episode(uuid: String): Episode
		users(): [User]!
		recentlyAdded(): [MediaItem]
		upNext(): [MediaItem]
		search(name: String!): [SearchItem]
		invites(): [Invite]
		# List of all remotes found in a rclone config file if one exists.
		remotes(): [String]!

		unidentifiedMovieFiles(offset: Int, limit: Int): [MovieFile]!
		unidentifiedEpisodeFiles(offset: Int, limit: Int): [EpisodeFile]!

		tmdbSearchMovies(query: String!): [TmdbMovieSearchItem]!
		tmdbSearchSeries(query: String!): [TmdbSeriesSearchItem]!
	}

	type Mutation {
		# Tell the application to index all the supported files in the given directory.
		# 'kind' can be 0 for movies and 1 for series.
		# 'backend' can be 0 for local and 1 for Rclone.
		createLibrary(name: String!, filePath: String!, kind: Int!, backend: Int!, rcloneName: String): LibraryResponse!

		# Delete a library and remove all collected metadata.
		deleteLibrary(id: Int!): LibraryResponse!

		# Create a invite code so a user can register on the server
		createUserInvite(): UserInviteResponse!

		# Create a playstate for the given media item can be the UUID of an episode or movie.
		# Playtime should always be given in seconds.
		createPlayState(uuid: String!, finished: Boolean!, playtime: Float!): PlayStateResponse!

		# Request permission to play a certain file
		createStreamingTicket(uuid: String!): CreateSTResponse!

		# Delete a user from the database, please note that the user will be able to keep using the account until the JWT expires.
		deleteUser(id: Int!): UserResponse!

		# Rescans the mediaFile with the given ID (or all, if ID omitted) and updates the stream information in the database.
		updateStreams(uuid: String): Boolean!

		# Refreshes all external agent information (poster/overviews etc.) for the given library or season/series/episode.
		refreshAgentMetadata(libraryID: Int, uuid: String): Boolean!

		# Rescan all library paths for new files that are not indexed yet.
		rescanLibraries(): Boolean!

		# Tag an unidentified MovieFile
		updateMovieFileMetadata(input: UpdateMovieFileMetadataInput!): UpdateMovieFileMetadataPayload!

		# Retag one or multiple EpisodeFiles
		updateEpisodeFileMetadata(input: UpdateEpisodeFileMetadataInput!): UpdateEpisodeFileMetadataPayload!
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

	type BoolResponse {
		success: Boolean!
	}

	type CreateSTResponse {
		error: Error
		metadataPath: String!
		# Path with a JWT that will stream your file.
		hlsStreamingPath: String!
		dashStreamingPath: String!
		jwt: String!
		streams: [Stream]!
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
		uuid: String!
	}

	type PlayStateResponse {
		uuid: String!
		playState: PlayState
	}

	# A media library
	type Library {
		id: Int!

		# Library type (0 - movies, 1 - series)
		kind: Int!

		# Human readable name of the Library (unused)
		name: String!

		# Path that this library manages
		filePath: String!

		# Whether olaris-server is currently scanning the library
		isRefreshing: Boolean!

		# Backend for the library (0 - Local filesystem, 1 - Rclone)
		backend: Int!

		# If Backend is Rclone it will return the name of the remote
		rcloneName: String

		# This attribute will be false whenever a Rclone remote can't be reached
		healthy: Boolean!

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
		# FileSize in bytes
		fileSize: Int!
		# Get the library for the given file
		library: Library!
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
	  # Stream/Track ID as found in the original file
	  streamID: Int
	  # StreamURL
	  streamURL: String
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
		# FileSize in bytes
		fileSize: Int!
		# Get the library for the given file
		library: Library!
	}

	input UpdateMovieFileMetadataInput {
		// UUID of the movie file to update
		movieFileUUID: String!

		// TMDB ID to update with. The mutation will retrieve the details from TMDB and update the
		// item appropriately.
		tmdbID: Int!
	}

	type UpdateMovieFileMetadataPayload {
		error: Error

		mediaItem: MediaItem
	}

	input UpdateEpisodeFileMetadataInput {
		// UUID of the episode file to update
		episodeFileUUID: String
		// UUID of the series for which all EpisodeFiles should be updated.
		// episodeFileUUID and seriesUUID are mutually exclusive.
		seriesUUID: String

		// TMDB ID to update with. The mutation will retrieve the details from TMDB and update the
		// item appropriately.
		tmdbID: Int!
	}

	type UpdateEpisodeFileMetadataPayload {
		error: Error
	}

	type MovieAddedEvent {
		movie: Movie!
	}

	type EpisodeAddedEvent {
		episode: Episode!
	}

	type SeriesAddedEvent {
		series: Series!
	}

	type SeasonAddedEvent {
		season: Season!
	}

	# Invite that can be used to allow other users access to your server.
	type Invite {
		code: String
		user: User
	}

	type TmdbMovieSearchItem {
		# Official title according to the MovieDB
		title: String!
		# Release year
		releaseYear: Int
		# Short description of the movie
		overview: String!
		# TMDB ID
		tmdbID: Int!
		# ID to retrieve backdrop
		backdropPath: String!
		# ID to retrieve poster
		posterPath: String!
	}

	type TmdbSeriesSearchItem {
		# Official title according to the MovieDB
		name: String!
		# Release year
		firstAirYear: Int
		# TMDB ID
		tmdbID: Int!
		# ID to retrieve backdrop
		backdropPath: String!
		# ID to retrieve poster
		posterPath: String!
	}

`

// InitSchema inits the graphql schema.
func InitSchema(env *app.MetadataContext) *graphql.Schema {
	Schema := graphql.MustParseSchema(SchemaTxt, NewResolver(env))
	return Schema
}
