package streaming

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"net/http"
	"strconv"
	"strings"
)

var allowDirectFileAccessFlag = flag.Bool(
	"allow_direct_file_access",
	false,
	"Whether accessing files directly by their path (without presenting a valid JWT) is allowed")

// getNode parses the file that the client is trying to access from a string.
// The passed string may either be in the form of "jwt/<streaming JWT>
// or simply directly an absolute path.
//
// This function also checks whether the user is allowed to access this file, i.e. whether
// the passed JWT is valid or whether accessing paths directly is allowed (controlled by a flag)
func getFileLocator(urlFileLocatorStr string) (filesystem.FileLocator, error) {
	return _getFileLocator(urlFileLocatorStr, false)

}

func getStreamingClaims(urlFileLocator string) (*auth.StreamingClaims, error) {
	// Allow both with and without leading slash, but canonical version is without
	if urlFileLocator[0] == '/' {
		urlFileLocator = urlFileLocator[1:]
	}

	parts := strings.SplitN(urlFileLocator, "/", 2)

	if len(parts) != 2 {
		return nil,
			fmt.Errorf("Failed to split urlFileLocator \"%s\"", urlFileLocator)
	}

	if parts[0] == "jwt" {
		claims, err := auth.ValidateStreamingJWT(parts[1])
		if err != nil {
			return nil, fmt.Errorf("Failed to validate JWT: %s", err.Error())
		}
		return claims, nil
	}
	return nil, fmt.Errorf("No JWT in file locator")
}

func _getFileLocator(urlFileLocator string, allowDirectFileAccess bool) (filesystem.FileLocator, error) {
	// Allow both with and without leading slash, but canonical version is without
	if urlFileLocator[0] == '/' {
		urlFileLocator = urlFileLocator[1:]
	}

	parts := strings.SplitN(urlFileLocator, "/", 2)
	if parts[0] == "jwt" {
		claims, err := auth.ValidateStreamingJWT(parts[1])
		if err != nil {
			return filesystem.FileLocator{},
				fmt.Errorf("Failed to validate JWT: %s", err.Error())
		}
		// Set authenticated to
		return _getFileLocator(claims.FilePath, true)
	}

	if !(allowDirectFileAccess || *allowDirectFileAccessFlag) {
		return filesystem.FileLocator{},
			errors.New("Direct file access is not allowed!")
	}

	fileLocator, err := filesystem.ParseFileLocator(urlFileLocator)
	if err != nil {
		return filesystem.FileLocator{}, err

	}
	return fileLocator, nil
}

func getStreamKey(fileLocator filesystem.FileLocator, streamIdStr string) (ffmpeg.StreamKey, error) {
	streamId, err := strconv.Atoi(streamIdStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	return ffmpeg.StreamKey{
		StreamId:    int64(streamId),
		FileLocator: fileLocator,
	}, nil
}

func getFileLocatorOrFail(r *http.Request) (filesystem.FileLocator, Error) {
	fileLocatorStr := mux.Vars(r)["fileLocator"]
	fileLocator, err := getFileLocator(fileLocatorStr)
	if err != nil {
		return filesystem.FileLocator{}, StatusError{
			Err: errors.Wrap(err,
				fmt.Sprintf("Failed to build file locator from %s", fileLocatorStr)),
			Code: http.StatusInternalServerError,
		}
	}
	// Only check for local files cause it's quick
	if fileLocator.Backend == filesystem.BackendLocal {
		_, err = filesystem.GetNodeFromFileLocator(fileLocator)
		if err != nil {
			return filesystem.FileLocator{}, StatusError{
				Err: errors.Wrap(err,
					fmt.Sprintf("Media file \"%s\" does not exist.", fileLocator)),
				Code: http.StatusNotFound,
			}
		}
	}
	return fileLocator, nil
}
