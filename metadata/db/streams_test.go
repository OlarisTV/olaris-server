package db_test

import (
	"fmt"
	"testing"

	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

func TestBeforeCreate(t *testing.T) {
	app.NewMDContext(db.DatabaseOptions{
		Connection: fmt.Sprintf("sqlite3://%s", db.Memory),
	}, nil)
	stream := db.Stream{Codecs: "test"}
	db.CreateStream(&stream)
	if stream.UUID == "" {
		t.Errorf("Stream was created without a UUID\n")
	}
}
