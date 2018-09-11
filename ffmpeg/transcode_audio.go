package ffmpeg

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/olaris/olaris-server/ffmpeg/ffchunk_options"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

var AudioEncoderPresets = map[string]EncoderParams{
	"64k-audio":  {audioBitrate: 64000, Codecs: "mp4a.40.2"},
	"128k-audio": {audioBitrate: 128000, Codecs: "mp4a.40.2"},
}

// This is exactly 234 AAC frames (1024 samples each) @ 48kHz.
// TODO(Leon Handreke): Do we need to set this differently for different sampling rates?
const transcodedAudioSegmentDuration = 4992 * time.Millisecond

func NewAudioTranscodingSession(
	stream StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	// TODO(Leon Handreke): Fix the prerun
	//startDuration := timestampToDuration(segments[0].StartTimestamp, stream.Stream.TimeBase)
	//endDuration := timestampToDuration(segments[len(segments)-1].EndTimestamp, stream.Stream.TimeBase)

	// With AAC, we always encode an extra segment before to avoid encoder priming on the first segment we actually want
	//runDuration := segmentsPerSession*transcodedAudioSegmentDuration + transcodedAudioSegmentDuration
	//runStartDuration := startDuration - transcodedAudioSegmentDuration
	//
	//if runStartDuration < 0 {
	//	runStartDuration = 0
	//}
	//if startNumber < 0 {
	//	startNumber = 0
	//}

	splitTimes := []int64{}
	for _, s := range segments[1:] {
		splitTimes = append(splitTimes, int64(s.StartTimestamp))
	}

	options := ffchunk_options.FFChunkOptions{
		InputFile:         stream.Stream.MediaFileURL,
		OutputDir:         outputDir,
		StreamIndex:       stream.Stream.StreamId,
		StartDts:          int64(segments[0].StartTimestamp),
		EndDts:            int64(segments[len(segments)-1].EndTimestamp),
		SegmentStartIndex: int64(segments[0].SegmentId),
		SplitDts:          splitTimes,
	}
	optionsSerialized, _ := proto.Marshal(&options)

	cmd := exec.Command("ffchunk_transcode_audio")
	log.Println("ffchunk_transcode_audio started with", cmd.Args, options.String())

	logSink := getTranscodingLogSink("ffchunk_transcode_audio")
	io.WriteString(logSink, fmt.Sprintf("%s %s\n\n", cmd.Args, options.String()))
	cmd.Stderr = logSink

	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	stdin, _ := cmd.StdinPipe()
	stdin.Write(optionsSerialized)
	stdin.Close()

	return &TranscodingSession{
		cmd:       cmd,
		Stream:    stream,
		outputDir: outputDir,
		segments:  segments,
	}, nil
}

func GetTranscodedAudioRepresentation(stream Stream, representationId string, encoderParams EncoderParams) StreamRepresentation {
	keyFrameItervals, _ := GetKeyframeIntervals(stream)

	segmentStartTimestamps := BuildConstantSegmentDurations(
		keyFrameItervals, transcodedAudioSegmentDuration)

	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.audioBitrate,
			Container:        "audio/mp4",
			Codecs:           encoderParams.Codecs,
			transcoded:       true,
		},
		SegmentStartTimestamps: segmentStartTimestamps,
	}
}
