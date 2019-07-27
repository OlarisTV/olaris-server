package resolvers

import "gitlab.com/olaris/olaris-server/metadata/db"

func buildDatabaseQueryDetails(offset *int32, limit *int32) db.QueryDetails {
	qd := db.QueryDetails{}

	qd.Limit = 50
	if limit != nil {
		qd.Limit = int(*limit)
	}

	qd.Offset = 0
	if offset != nil {
		qd.Offset = int(*offset)
	}

	return qd
}
