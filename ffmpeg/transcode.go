package ffmpeg

import (
	"fmt"
	"math"
	"math/big"
)

func GetSimilarEncoderParams(stream Stream) (EncoderParams, error) {
	if stream.StreamType == "video" {
		return EncoderParams{
			videoBitrate: int(stream.BitRate),
			Codecs:       GetAVC1Tag(stream.Width, stream.Height, stream.BitRate, stream.FrameRate),
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

func GetAVC1Tag(width int, height int, biteRate int64, frameRate *big.Rat) string {
	frameSizeMacroblocks := (width / 16.0) * (height / 16.0)
	frameRateFloat, _ := frameRate.Float64()
	macroblocksPerSecond := float64(frameSizeMacroblocks) * frameRateFloat
	level := avc1Levels[len(avc1Levels)-1]
	for _, l := range avc1Levels {
		if biteRate < l.MaxBitrate &&
			int64(frameSizeMacroblocks) < l.MaxFrameSize &&
			macroblocksPerSecond < float64(l.MaxMacroblocksProcessingRate) {
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
	/* max macroblock processing rate (macroblocks/sec) */
	MaxMacroblocksProcessingRate int64
	/* max frame size (macroblocks) */
	MaxFrameSize int64
	MaxBitrate   int64
}

// NOTE(Leon Handreke): This table is for Baseline Extended Main. We actually encode at High, so the accuracy
// here could be improved. See also https://de.wikipedia.org/wiki/H.264#Level (German Wikipedia contains max
// bitrates at other profiles).
var avc1Levels = []avc1Level{
	{10, 1485, 99, 64000},
	{9, 1485, 99, 128000}, /* "1b" */
	{11, 3000, 396, 192000},
	{12, 6000, 396, 384000},
	{13, 11880, 396, 768000},
	{20, 11880, 396, 2000000},
	{21, 19800, 792, 4000000},
	{22, 20250, 1620, 4000000},
	{30, 40500, 1620, 10000000},
	{31, 108000, 3600, 14000000},
	{32, 216000, 5120, 20000000},
	{40, 245760, 8192, 20000000},
	{41, 245760, 8192, 50000000},
	{42, 522240, 8704, 50000000},
	{50, 589824, 22080, 135000000},
	{51, 983040, 36864, 240000000},
	{52, 2073600, 36864, 240000000},
	{60, 4177920, 139264, 240000000},
	{61, 8355840, 139264, 480000000},
	{62, 16711680, 139264, 800000000},
}
