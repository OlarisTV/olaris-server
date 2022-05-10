package streaming

var schemaTxt = `schema {
    query: Query
}


# The query type, represents all of the entry points into our object graph
type Query {
    sessions(): [Session]!
}

type Session {
    fileLocator: String!
    sessionID: String!
    lastAccessed: String!
    playbackSessionID: String!
    container: String!
    resolution: String!
    codecs: String!
    codecName: String!
    streamType: String!
    language: String!
    title: String!
    userID: Int!
    lastRequestedSegmentIdx: Int!
    transcodingPercentage: Int!
    throttled: Boolean!
    transcoded: Boolean!
    transmuxed: Boolean!
    bitRate: Int!
}`
