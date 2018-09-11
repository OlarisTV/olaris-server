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

var VideoEncoderPresets = map[string]EncoderParams{
	"480-1000k-video":   {height: 480, width: -2, videoBitrate: 1000000, Codecs: "avc1.64001e"},
	"720-5000k-video":   {height: 720, width: -2, videoBitrate: 5000000, Codecs: "avc1.64001f"},
	"1080-10000k-video": {height: 1080, width: -2, videoBitrate: 10000000, Codecs: "avc1.640028"},
}

// Doesn't have to be the same as audio, but why not.
const transcodedVideoSegmentDuration = 4992 * time.Millisecond

func NewVideoTranscodingSession(
	stream StreamRepresentation,
	segments SegmentList,
	outputDirBase string) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	//startDuration := timestampToDuration(segments[0].StartTimestamp, stream.Stream.TimeBase)
	//endDuration := timestampToDuration(segments[len(segments)-1].EndTimestamp, stream.Stream.TimeBase)
	// TODO(Leon Handreke): Pass encoder params to ffchunk
	//encoderParams := stream.Representation.encoderParams

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

	cmd := exec.Command("ffchunk_transcode_video")
	log.Println("ffchunk_transcode_video started with", cmd.Args, options.String())

	logSink := getTranscodingLogSink("ffchunk_transcode_video")
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

func GetTranscodedVideoRepresentation(
	stream Stream,
	representationId string,
	encoderParams EncoderParams) StreamRepresentation {

	keyFrameItervals, _ := GetKeyframeIntervals(stream)

	segmentStartTimestamps := BuildConstantSegmentDurations(
		keyFrameItervals, transcodedAudioSegmentDuration)

	return StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: representationId,
			BitRate:          encoderParams.videoBitrate,
			Container:        "video/mp4",
			Codecs:           encoderParams.Codecs,
			transcoded:       true,
		},
		SegmentStartTimestamps: segmentStartTimestamps,
	}
}
