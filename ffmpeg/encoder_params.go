package ffmpeg

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
)

type EncoderParams struct {
	// One of these may be -2 to keep aspect ratio. Using -2 (instead of -1) ensures
	// that the number is rounded to the nearest full pixel.
	width        int
	height       int
	videoBitrate int
	audioBitrate int

	// The codecs (https://tools.ietf.org/html/rfc6381#section-3.3) that these params will produce.
	Codecs string
}

// SetWidthAndHeight allows you to update the dimensions to scaled values when transcoding
func (e *EncoderParams) SetWidthAndHeight(width int, height int) {
	e.width = width
	e.height = height
}

func EncoderParamsToString(m EncoderParams) string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		panic(`failed gob Encode`)
	}
	return base64.URLEncoding.EncodeToString(b.Bytes())
}

func EncoderParamsFromString(str string) (EncoderParams, error) {
	m := EncoderParams{}
	by, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return EncoderParams{}, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		return EncoderParams{}, err
	}
	return m, nil
}
