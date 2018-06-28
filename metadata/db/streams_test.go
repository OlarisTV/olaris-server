package db

import (
	"testing"

	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
)

func TestBeforeCreate(t *testing.T) {
	NewMDContext("/tmp/", false)
	stream := Stream{Stream: ffmpeg.Stream{Codecs: "test"}}
	env.Db.Create(&stream)
	if stream.UUID == "" {
		t.Errorf("Stream was created without a UUID\n")
	}
}
