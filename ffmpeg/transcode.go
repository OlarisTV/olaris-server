package ffmpeg

import (
	"fmt"
	"math"
)

func GetSimilarEncoderParams(stream Stream) (EncoderParams, error) {
	if stream.StreamType == "video" {
		return EncoderParams{
			videoBitrate: int(stream.BitRate),
			Codecs:       GetAVC1Tag(uint64(stream.BitRate), stream.Width, stream.Height),
			// TODO(Leon Handreke): Don't even invoke the scale filter in this case.
			width:  -2,
			height: stream.Height,
		}, nil
	} else if stream.StreamType == "audio" {
		return EncoderParams{
			audioBitrate: int(stream.BitRate),
			Codecs:       "mp4a.40.2",
		}, nil

	}
	return EncoderParams{}, fmt.Errorf("Cannot produce similar transcoded version for %s", stream.StreamType)
}

func GetAVC1Tag(bitrate uint64, width int, height int) string {
	frameSizeMacroblocks := uint((width / 16) * (height / 16))
	level := avc1Levels[len(avc1Levels)-1]
	for _, l := range avc1Levels {
		if bitrate < l.MaxBitrate && frameSizeMacroblocks < l.MaxFrameSize {
			level = l
			break
		}
	}
	return fmt.Sprintf("avc1.6400%x", level.Level)
}

// scalePreserveAspectRatio implements ffmpeg-eseque scaling: For one of the two values, a negative value must
// be specified, the result will be divisible by the absolute of the negative value.
func scalePreserveAspectRatio(width int, height int, newWidth int, newHeight int) (int, int) {
	var nw, nh = float64(newWidth), float64(newHeight)

	if nw < 0 {
		nw = nh * (float64(width) / float64(height))

		factor := float64(-newWidth)
		nw = math.Round(nw/factor) * factor
	}
	if nh < 0 {
		nh = nw * (float64(height) / float64(width))

		factor := float64(-newHeight)
		nh = math.Round(nh/factor) * factor
	}

	return int(nw), int(nh)
}

// Copied from libx264 and common/tables.c and Wikipedia
type avc1Level struct {
	Level uint
	/* max frame size (macroblocks) */
	MaxFrameSize uint
	MaxBitrate   uint64
}

// NOTE(Leon Handreke): This table is for Baseline Extended Main. We actually encode at High, so the accuracy
// here could be improved. See also https://de.wikipedia.org/wiki/H.264#Level (German Wikipedia contains max
// bitrates at other profiles).
var avc1Levels = []avc1Level{
	{10, 99, 64000},
	{9, 99, 128000}, /* "1b" */
	{11, 396, 192000},
	{12, 396, 384000},
	{13, 396, 768000},
	{20, 396, 2000000},
	{21, 792, 4000000},
	{22, 1620, 4000000},
	{30, 1620, 10000000},
	{31, 3600, 14000000},
	{32, 5120, 20000000},
	{40, 8192, 20000000},
	{41, 8192, 50000000},
	{42, 8704, 50000000},
	{50, 22080, 135000000},
	{51, 36864, 240000000},
	{52, 36864, 240000000},
	{60, 139264, 240000000},
	{61, 139264, 480000000},
	{62, 139264, 800000000},
}
