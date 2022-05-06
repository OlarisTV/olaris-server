package streaming

var schemaTxt = `schema {
    query: Query
    mutation: Mutation
}


# The query type, represents all of the entry points into our object graph
type Query {
    sessions(): [Session]!
}

type Mutation {
}


type Session {
    fileLocator: String!
    lastAccessed: String!
    playbackSessionID: String!
    container: String!
    resolution: String!
    codecs: String!
    userID: Int!
    lastRequestedSegmentIdx: Int!
    transcodingPercentage: Int!
    throttled: Boolean!
    transcoded: Boolean!
    transmuxed: Boolean!
    bitRate: Int!
}`
