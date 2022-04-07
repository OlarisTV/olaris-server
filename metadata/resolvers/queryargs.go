package resolvers

import (
	"gitlab.com/olaris/olaris-server/metadata/db"
	"strings"
)

type queryArgs struct {
	UUID          *string
	Offset        *int32
	Limit         *int32
	SortDirection *string
}

func (m *queryArgs) asQueryDetails() *db.QueryDetails {
	qd := db.QueryDetails{}

	if m.Limit == nil {
		qd.Limit = 50
	} else {
		qd.Limit = int(*m.Limit)
	}

	if m.Offset == nil {
		qd.Offset = 0
	} else {
		qd.Offset = int(*m.Offset)
	}

	if m.SortDirection != nil {
		qd.SortDirection = strings.ToUpper(*m.SortDirection)
	} else {
		qd.SortDirection = "ASC"
	}

	return &qd
}
