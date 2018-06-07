// Convenience wrapper around ffmpeg as a transcoder to DASH chunks
// https://github.com/go-cmd/cmd/blob/master/cmd.go was very useful while writing this module.
package ffmpeg

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Representation struct {
	RepresentationId string

	// The rest is just metadata for display
	BitRate int
	// e.g. "video/mp4"
	Container string
	// codecs string ready for DASH/HLS serving
	Codecs string

	// Mutually exclusive
	transcoded bool
	transmuxed bool

	encoderParams EncoderParams
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

type ClientCodecCapabilities struct {
	PlayableCodecs []string `json:"playableCodecs"`
}

func GetTransmuxedOrTranscodedRepresentation(
	stream Stream,
	capabilities ClientCodecCapabilities) (StreamRepresentation, error) {

	// We interpret emtpy PlayableCodecs as no preference
	if len(capabilities.PlayableCodecs) == 0 {
		return GetTransmuxedRepresentation(stream)
	}

	for _, playableCodec := range capabilities.PlayableCodecs {
		if playableCodec == stream.Codecs {
			return GetTransmuxedRepresentation(stream)
		}
	}
	representations := []StreamRepresentation{}

	similarEncoderParams, _ := GetSimilarEncoderParams(stream)
	if stream.StreamType == "audio" {
		representations = append(representations,
			GetTranscodedAudioRepresentation(
				stream,
				// TODO(Leon Handreke): Make a util method for this prefix.
				"transcode:"+EncoderParamsToString(similarEncoderParams),
				similarEncoderParams))

		// TODO(Leon Handreke): Ugly hardcode to 128k AAC
		representation, _ := StreamRepresentationFromRepresentationId(
			stream, "preset:128k-audio")
		representations = append(representations, representation)
	}
	if stream.StreamType == "video" {
		representations = append(representations,
			GetTranscodedVideoRepresentation(
				stream,
				// TODO(Leon Handreke): Make a util method for this prefix.
				"transcode:"+EncoderParamsToString(similarEncoderParams),
				similarEncoderParams))

		// TODO(Leon Handreke): Ugly hardcode to 720p-5000k H264
		representation, _ := StreamRepresentationFromRepresentationId(
			stream, "preset:720-5000k-video")
		representations = append(representations, representation)

	}
	for _, r := range representations {
		for _, playableCodec := range capabilities.PlayableCodecs {
			if playableCodec == r.Representation.Codecs {
				return r, nil
			}
		}
	}
	return StreamRepresentation{},
		fmt.Errorf("Could not find appropriate representation for stream %s", stream.StreamType)
}

func StreamRepresentationFromRepresentationId(
	s Stream,
	representationId string) (StreamRepresentation, error) {

	if s.StreamType == "subtitle" {
		return GetSubtitleStreamRepresentation(s), nil
	}

	if representationId == "direct" {
		transmuxedStream, err := GetTransmuxedRepresentation(s)
		if err != nil {
			return StreamRepresentation{}, err
		}
		if transmuxedStream.Representation.RepresentationId == representationId {
			return transmuxedStream, nil
		}
	} else if strings.HasPrefix(representationId, "preset:") {
		presetId := representationId[7:]
		if encoderParams, ok := VideoEncoderPresets[presetId]; ok {
			return GetTranscodedVideoRepresentation(s, representationId, encoderParams), nil
		}
		if encoderParams, ok := AudioEncoderPresets[presetId]; ok {
			return GetTranscodedAudioRepresentation(s, representationId, encoderParams), nil
		}
	} else if strings.HasPrefix(representationId, "transcode:") {
		encoderParamsStr := representationId[10:]
		encoderParams, err := EncoderParamsFromString(encoderParamsStr)
		if err != nil {
			return StreamRepresentation{}, err
		}
		if s.StreamType == "video" {
			return GetTranscodedVideoRepresentation(s, representationId, encoderParams), nil
		} else if s.StreamType == "audio" {
			return GetTranscodedAudioRepresentation(s, representationId, encoderParams), nil
		}
	}

	return StreamRepresentation{},
		fmt.Errorf("No such stream %d/%s found for file %s",
			s.StreamId, representationId, s.MediaFilePath)
}

func NewTranscodingSession(s StreamRepresentation, segmentId int) (*TranscodingSession, error) {
	if s.Representation.RepresentationId == "direct" {
		session, err := NewTransmuxingSession(s, os.TempDir(), segmentId)
		if err != nil {
			return nil, err
		}
		return session, nil
	} else {
		var session *TranscodingSession
		var err error

		if s.Stream.StreamType == "video" {
			session, err = NewVideoTranscodingSession(s, os.TempDir(), segmentId)
			return session, nil
		} else if s.Stream.StreamType == "audio" {
			session, err = NewAudioTranscodingSession(s, os.TempDir(), segmentId)
			return session, nil
		} else if s.Stream.StreamType == "subtitle" {
			session, err = NewSubtitleSession(s, os.TempDir())
		}
		if err != nil {
			return nil, err
		}
		return session, nil
	}
}
