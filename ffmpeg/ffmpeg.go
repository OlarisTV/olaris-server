// Convenience wrapper around ffmpeg as a transcoder to DASH chunks
// https://github.com/go-cmd/cmd/blob/master/cmd.go was very useful while writing this module.
package ffmpeg

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/helpers"
	"os"
	"os/user"
	"path"
	"strings"
	"time"
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

// MinSegDuration defines the duration of segments that ffmpeg will generate. In the transmuxing case this is really
// just a minimum time, the actual segments will be longer because they are cut at keyframes. For transcoding, we can
// force keyframes to occur exactly every MinSegDuration, so MinSegDuration will be the actualy duration of the
// segments.
const TransmuxedSegDuration = 5000 * time.Millisecond

// fragmentsPerSession defines the number of segments to encode per launch of ffmpeg. This constant should strike a
// balance between minimizing the overhead cause by launching new ffmpeg processes and minimizing the minutes of video
// transcoded but never watched by the user. Note that this constant is currently only used for the transcoding case.
const segmentsPerSession = 12

type ClientCodecCapabilities struct {
	PlayableCodecs []string `json:"playableCodecs"`
}

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
		fmt.Errorf("No such stream %d/%s found for file %s",
			s.StreamId, representationId, s.MediaFileURL)
}

func NewTranscodingSession(s StreamRepresentation, segmentStartIndex int, feedbackURL string) (*TranscodingSession, error) {
	runtimeDir := getTranscodingSessionRuntimeDir()
	helpers.EnsurePath(runtimeDir)

	startTime := time.Duration(int64(segmentStartIndex) * int64(TransmuxedSegDuration))
	if s.Representation.RepresentationId == "direct" {
		session, err := NewTransmuxingSession(s, startTime, segmentStartIndex, runtimeDir, feedbackURL)
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
			session, err = NewVideoTranscodingSession(s, startTime, segmentStartIndex, runtimeDir, feedbackURL)
		} else if s.Stream.StreamType == "audio" {
			session, err = NewAudioTranscodingSession(s, startTime, segmentStartIndex, runtimeDir, feedbackURL)
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
	u, _ := user.Current()
	return path.Join(os.TempDir(), fmt.Sprintf("olaris-%s", u.Uid))
}
