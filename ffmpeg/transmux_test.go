package ffmpeg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func testGuessTransmuxed[]Segment(t *testing.T) {
	timeBase := int64(1000)
	keyframeIntervals := []Interval{
		{timeBase, 0, 1000},
		{timeBase, 1000, 6000},
		{timeBase, 6000, 10100},
		{timeBase, 10100, 10200},
	}
	result := guessTransmuxed[]Segment(keyframeIntervals)
	assert.Equal(t,
		[]Segment{
			{Interval{timeBase, 0, 6000}, 0},
			{Interval{timeBase, 6000, 10100}, 1},
			{Interval{timeBase, 10100, 10200}, 2},
		},
		result)
}
