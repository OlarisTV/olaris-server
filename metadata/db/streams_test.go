package db

import (
	"testing"

	"gitlab.com/olaris/olaris-server/ffmpeg"
)

func TestBeforeCreate(t *testing.T) {
	NewMDContext("/tmp/", false)
	stream := Stream{Stream: ffmpeg.Stream{Codecs: "test"}}
	env.Db.Create(&stream)
	if stream.UUID == "" {
		t.Errorf("Stream was created without a UUID\n")
	}
}
