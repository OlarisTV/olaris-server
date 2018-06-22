package streaming

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"path"
	"strconv"
	"strings"
)

func buildMediaFileURL(fileLocator string) (string, error) {
	parts := strings.SplitN(fileLocator, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("Failed to split file locator \"%s\"", fileLocator)
	}

	if parts[0] == "remote" {
		rcloneParts := strings.SplitN(parts[1], "/", 2)
		if len(rcloneParts) != 2 {
			return "", fmt.Errorf("Failed to split rclone path \"%s\"", rcloneParts)
		}
		rcloneURL, err := router.Get("rcloneFile").URL(
			"rcloneRemote", rcloneParts[0],
			"rclonePath", rcloneParts[1])
		if err != nil {
			return "", fmt.Errorf("Failed to build rclone URL: %s\n ", err.Error())

		}
		// TODO(Leon Handreke): Find a better way to do this
		return "http://127.0.0.1:8080/s" + rcloneURL.String(), nil
	}

	return "file://" + path.Join(*mediaFilesDir, path.Clean(parts[1])), nil
}

func buildStreamKey(fileLocator string, streamIdStr string) (ffmpeg.StreamKey, error) {
	streamId, err := strconv.Atoi(streamIdStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	url, err := buildMediaFileURL(fileLocator)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	return ffmpeg.StreamKey{
		StreamId:     int64(streamId),
		MediaFileURL: url,
	}, nil
}
