package db_test

import (
	"testing"

	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

func TestBeforeCreate(t *testing.T) {
	app.NewMDContext("/tmp/", false)
	stream := db.Stream{Stream: ffmpeg.Stream{Codecs: "test"}}
	db.CreateStream(&stream)
	if stream.UUID == "" {
		t.Errorf("Stream was created without a UUID\n")
	}
}
