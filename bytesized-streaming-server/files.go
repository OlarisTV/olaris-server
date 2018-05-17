package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/db"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var supportedExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".mov": true,
	".avi": true,
}

func serveFileIndex(w http.ResponseWriter, r *http.Request) {
	files := []MediaFile{}
	err := filepath.Walk(*mediaFilesDir, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(walkPath)] {
			relPath := strings.SplitAfter(walkPath, *mediaFilesDir)[1]
			fileInfo, err := os.Stat(walkPath)

			if err != nil {
				// This catches broken symlinks
				if _, ok := err.(*os.PathError); ok {
					fmt.Println("Got an error while statting file:", err)
					return nil
				}
				return err
			}

			mediaPlaybackState, err := db.GetSharedDB().GetMediaPlaybackState(relPath)
			if err != nil {
				return err
			}
			// TODO(Leon Handreke): Have an enum NOT_STARTED, STARTED, FINISHED in the response.
			playtime := -1
			if mediaPlaybackState != nil {
				playtime = mediaPlaybackState.Playtime
			}

			files = append(files, MediaFile{
				Key:                    MD5Ify(walkPath),
				Name:                   fileInfo.Name(),
				Size:                   fileInfo.Size(),
				Playtime:               playtime,
				HlsTranscodingManifest: path.Join(relPath, "hls-transcoding-manifest.m3u8"),
				HlsTransmuxingManifest: path.Join(relPath, "hls-transmuxing-manifest.m3u8")})
		}

		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(files)
}

type MediaFile struct {
	Ext                    string `json:"ext"`
	Name                   string `json:"name"`
	Key                    string `json:"key"`
	Size                   int64  `json:"size"`
	Playtime               int    `json:"playtime"`
	HlsTranscodingManifest string `json:"hlsTranscodingManifest"`
	HlsTransmuxingManifest string `json:"hlsTransmuxingManifest"`
}

func MD5Ify(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
