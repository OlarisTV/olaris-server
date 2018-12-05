package ffmpeg

import "fmt"

func GetSimilarEncoderParams(stream Stream) (EncoderParams, error) {
	if stream.StreamType == "video" {
		return EncoderParams{
			videoBitrate: int(stream.BitRate),
			Codecs:       "avc1.640028",
			width:        -2,
			height:       stream.Height,
		}, nil
	} else if stream.StreamType == "audio" {
		return EncoderParams{
			audioBitrate: int(stream.BitRate),
			Codecs:       "mp4a.40.2",
		}, nil

	}
	return EncoderParams{}, fmt.Errorf("Cannot produce similar transcoded version for %s", stream.StreamType)
}
