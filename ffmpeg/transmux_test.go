package ffmpeg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func testGuessTransmuxedSegmentList(t *testing.T) {
	keyframeIntervals := []Interval{
		{0, 1000},
		{1000, 6000},
		{6000, 10100},
		{10100, 10200},
	}
	result := guessTransmuxedSegmentList(keyframeIntervals)
	assert.Equal(t,
		[]Segment{
			{Interval{0, 6000}, 0},
			{Interval{6000, 10100}, 1},
			{Interval{10100, 10200}, 2},
		},
		result)
}
