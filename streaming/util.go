package streaming

import (
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"path"
	"strconv"
)

func getAbsoluteFilepath(filename string) string {
	return path.Join(*mediaFilesDir, path.Clean(filename))
}

func buildStreamKey(filename string, streamIdStr string) (ffmpeg.StreamKey, error) {
	streamId, err := strconv.Atoi(streamIdStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	return ffmpeg.StreamKey{
		StreamId:      int64(streamId),
		MediaFilePath: getAbsoluteFilepath(filename),
	}, nil
}
