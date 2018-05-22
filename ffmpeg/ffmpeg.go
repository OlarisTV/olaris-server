// Convenience wrapper around ffmpeg as a transcoder to DASH chunks
// https://github.com/go-cmd/cmd/blob/master/cmd.go was very useful while writing this module.
package ffmpeg

import (
	"time"
)

type Representation struct {
	RepresentationId string

	// The rest is just metadata for display
	BitRate int64
	// e.g. "video/mp4"
	Container string
	// codecs string ready for DASH/HLS serving
	Codecs string

	// Mutually exclusive
	transcoded bool
	transmuxed bool
}

type StreamRepresentation struct {
	Stream         Stream
	Representation Representation

	SegmentStartTimestamps []time.Duration
}

// MinSegDuration defines the duration of segments that ffmpeg will generate. In the transmuxing case this is really
// just a minimum time, the actual segments will be longer because they are cut at keyframes. For transcoding, we can
// force keyframes to occur exactly every MinSegDuration, so MinSegDuration will be the actualy duration of the
// segments.
const MinTransmuxedSegDuration = 5000 * time.Millisecond

// fragmentsPerSession defines the number of segments to encode per launch of ffmpeg. This constant should strike a
// balance between minimizing the overhead cause by launching new ffmpeg processes and minimizing the minutes of video
// transcoded but never watched by the user. Note that this constant is currently only used for the transcoding case.
const segmentsPerSession = 12

func sum(input ...time.Duration) time.Duration {
	var sum time.Duration
	for _, i := range input {
		sum += i
	}
	return sum
}

type EncoderParams struct {
	// One of these may be -1 to keep aspect ratio
	// TODO(Leon Handreke): Add note about -2
	width        int
	height       int
	videoBitrate int
	audioBitrate int
}

func ComputeSegmentDurations(
	segmentStartTimestamps []time.Duration,
	totalDuration time.Duration) []time.Duration {

	// Insert dummy keyframe timestamp at the end so that the last segment duration is correctly reported
	segmentStartTimestamps = append(segmentStartTimestamps, totalDuration)

	segmentDurations := []time.Duration{}

	for i := 1; i < len(segmentStartTimestamps); i++ {
		segmentDurations = append(segmentDurations,
			segmentStartTimestamps[i]-segmentStartTimestamps[i-1])
	}

	return segmentDurations
}
