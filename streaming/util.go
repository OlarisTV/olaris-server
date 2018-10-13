package streaming

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type fileLocator struct {
	Location string
	Path     string
}

var allowDirectFileAccessFlag = flag.Bool(
	"allow_direct_file_access",
	false,
	"Whether accessing files directly by their path (without presenting a valid JWT) is allowed")

// getFileLocator parses the file that the client is trying to access from a string.
// The passed string may either be in the form of "jwt/<streaming JWT>
// or simply directly an absolute path.
//
// This function also checks whether the user is allowed to access this file, i.e. whether
// the passed JWT is valid or whether accessing paths directly is allowed (controlled by a flag)
func getFileLocator(fileLocatorStr string) (fileLocator, error) {
	return _getFileLocator(fileLocatorStr, false)

}

func _getFileLocator(fileLocatorStr string, allowDirectFileAccess bool) (fileLocator, error) {
	// Allow both with and without leading slash, but canonical version is without
	if fileLocatorStr[0] == '/' {
		fileLocatorStr = fileLocatorStr[1:]
	}

	parts := strings.SplitN(fileLocatorStr, "/", 2)

	if len(parts) != 2 {
		return fileLocator{},
			fmt.Errorf("Failed to split file locator \"%s\"", fileLocatorStr)
	}

	if parts[0] == "jwt" {
		claims, err := auth.ValidateStreamingJWT(parts[1])
		if err != nil {
			return fileLocator{}, fmt.Errorf("Failed to validate JWT: %s", err.Error())
		}
		// Set authenticated to
		return _getFileLocator(claims.FilePath, true)
	}

	if !(allowDirectFileAccess || *allowDirectFileAccessFlag) {
		return fileLocator{}, errors.New("Direct file access is not allowed!")
	}

	if parts[0] == "rclone" {
		return fileLocator{parts[0], "/" + parts[1]}, nil
	}

	// Don't require an explicit local prefix for now
	return fileLocator{"local", "/" + fileLocatorStr}, nil

}

func getMediaFileURL(fileLocatorStr string) (string, error) {
	l, err := getFileLocator(fileLocatorStr)
	if err != nil {
		return "", err
	}

	if l.Location == "local" {
		return "file://" + l.Path, nil
	} else if l.Location == "rclone" {
		// TODO(Leon Handreke): Find a better way to do this
		return "http://127.0.0.1:8080/s/files/rclone/" + l.Path, nil
	}
	return "", fmt.Errorf("Could not build media file URL: Unknown file locator \"%s\"", l.Location)
}

func mediaFileURLExists(mediaFileURLStr string) bool {
	mediaFileURL, _ := url.Parse(mediaFileURLStr)
	if mediaFileURL.Scheme == "file" {
		if _, err := os.Stat(mediaFileURL.Path); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func getStreamKey(fileLocatorStr string, streamIdStr string) (ffmpeg.StreamKey, error) {
	streamId, err := strconv.Atoi(streamIdStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	url, err := getMediaFileURL(fileLocatorStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	return ffmpeg.StreamKey{
		StreamId:     int64(streamId),
		MediaFileURL: url,
	}, nil
}

func getMediaFileURLOrFail(r *http.Request) (string, Error) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		return "", StatusError{
			Err:  fmt.Errorf("Failed to build media file URL: %s", err.Error()),
			Code: http.StatusInternalServerError,
		}
	}
	if !mediaFileURLExists(mediaFileURL) {
		return "", StatusError{
			Err:  fmt.Errorf("Media file \"%s\" doee not exist.", mediaFileURL),
			Code: http.StatusNotFound,
		}
	}
	return mediaFileURL, nil
}
