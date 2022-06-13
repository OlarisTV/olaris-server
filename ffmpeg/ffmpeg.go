// Convenience wrapper around ffmpeg as a transcoder to DASH chunks
// https://github.com/go-cmd/cmd/blob/master/cmd.go was very useful while writing this module.
package ffmpeg

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/helpers"
)

type Representation struct {
	RepresentationId string

	encoderParams EncoderParams

	// The rest is just metadata for display
	BitRate int
	Height  int
	Width   int
	// e.g. "video/mp4"
	Container string
	// codecs string ready for DASH/HLS serving
	Codecs string

	// Mutually exclusive
	Transcoded bool
	Transmuxed bool
}

type StreamRepresentation struct {
	Stream         Stream
	Representation Representation
}

// SegmentDuration defines the duration of segments that ffmpeg will generate. In the transmuxing case this is really
// just a minimum time, the actual segments will be longer because they are cut at keyframes. For transcoding, we can
// force keyframes to occur exactly every SegmentDuration, so SegmentDuration will be the actual duration of the
// segments.
const SegmentDuration = 5000 * time.Millisecond

// segmentsPerSession defines the number of segments to encode per launch of ffmpeg. This constant should strike a
// balance between minimizing the overhead cause by launching new ffmpeg processes and minimizing the minutes of video
// transcoded but never watched by the user. Note that this constant is currently only used for the transcoding case.
const segmentsPerSession = 12

func ComputeSegmentDurations(sessions [][]Segment) []time.Duration {
	segmentDurations := []time.Duration{}

	for _, session := range sessions {
		for _, segment := range session {
			segmentDurations = append(segmentDurations, segment.Duration())
		}
	}

	return segmentDurations
}

func GetTransmuxedOrTranscodedRepresentation(
	stream Stream,
	capabilities ClientCodecCapabilities) (StreamRepresentation, error) {

	transmuxed := GetTransmuxedRepresentation(stream)
	// We interpret empty PlayableCodecs as no preference
	if len(capabilities.PlayableCodecs) == 0 || capabilities.CanPlay(transmuxed) {
		return transmuxed, nil
	}
	return GetSimilarTranscodedRepresentation(stream), nil
}

func GetSimilarTranscodedRepresentation(stream Stream) StreamRepresentation {
	similarEncoderParams, _ := GetSimilarEncoderParams(stream)
	if stream.StreamType == "audio" {
		return GetTranscodedAudioRepresentation(
			stream,
			// TODO(Leon Handreke): Make a util method for this prefix.
			"transcode:"+EncoderParamsToString(similarEncoderParams),
			similarEncoderParams)
	}
	if stream.StreamType == "video" {
		return GetTranscodedVideoRepresentation(
			stream,
			// TODO(Leon Handreke): Make a util method for this prefix.
			"transcode:"+EncoderParamsToString(similarEncoderParams),
			similarEncoderParams)

	}

	panic("GetSimliarTranscodedRepresentation for stream that is not audio/video")
}

// TODO(Leon Handreke): Should this really return an error?
func StreamRepresentationFromRepresentationId(
	s Stream,
	representationId string) (StreamRepresentation, error) {

	if s.StreamType == "subtitle" {
		return GetSubtitleStreamRepresentation(s), nil
	}

	if representationId == "direct" {
		return GetTransmuxedRepresentation(s), nil
	} else if strings.HasPrefix(representationId, "preset:") {
		presetId := representationId[7:]

		encoderParams, err := GetVideoEncoderPreset(s, presetId)

		if err == nil {
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
		fmt.Errorf("no such stream %d/%s found for file %s",
			s.StreamId, representationId, s.FileLocator)
}

func NewTranscodingSession(s StreamRepresentation, segmentStartIndex int) (*TranscodingSession, error) {
	runtimeDir := getTranscodingSessionRuntimeDir()
	helpers.EnsurePath(runtimeDir)

	startTime := time.Duration(int64(segmentStartIndex) * int64(SegmentDuration))
	if s.Representation.RepresentationId == "direct" {
		session, err := NewTransmuxingSession(s, startTime, segmentStartIndex, runtimeDir)
		if err != nil {
			return nil, err
		}
		session.Start()
		if err != nil {
			return nil, err
		}
		return session, nil
	} else {
		var session *TranscodingSession
		var err error

		if s.Stream.StreamType == "video" {
			session, err = NewVideoTranscodingSession(s, startTime, segmentStartIndex, runtimeDir)
		} else if s.Stream.StreamType == "audio" {
			session, err = NewAudioTranscodingSession(s, startTime, segmentStartIndex, runtimeDir)
		} else if s.Stream.StreamType == "subtitle" {
			session, err = NewSubtitleSession(s, runtimeDir)
		}
		if err != nil {
			return nil, err
		}
		session.Start()
		if err != nil {
			return nil, err
		}
		return session, nil
	}
}

func CleanTranscodingCache() error {
	return os.RemoveAll(getTranscodingSessionRuntimeDir())
}

func getTranscodingSessionRuntimeDir() string {
	return path.Join(viper.GetString("server.cacheDir"), "transcoding-sessions")
}
