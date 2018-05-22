package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/db"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
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

			videoStream, _ := ffmpeg.GetVideoStream(walkPath)
			audioStreams, _ := ffmpeg.GetAudioStreams(walkPath)
			// Golang doesn't have a set, so use this hack
			audioCodecsSet := map[string]struct{}{}
			for _, s := range audioStreams {
				audioCodecsSet[s.Codecs] = struct{}{}
			}
			audioCodecs := []string{}
			for codec, _ := range audioCodecsSet {
				audioCodecs = append(audioCodecs, codec)
			}

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
				Codecs:                 append(audioCodecs, videoStream.Codecs),
				TranscodedCodecs:       []string{"mp4a.40.2", "avc1.64001e", "avc1.64001f", "avc1.640028"},
				HlsManifest:            path.Join(relPath, "hls-manifest.m3u8"),
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
	Ext                    string   `json:"ext"`
	Name                   string   `json:"name"`
	Key                    string   `json:"key"`
	Size                   int64    `json:"size"`
	Playtime               int      `json:"playtime"`
	Codecs                 []string `json:"codecs"`
	TranscodedCodecs       []string `json:"transcodedCodecs"`
	HlsManifest            string   `json:"hlsManifest"`
	HlsTranscodingManifest string   `json:"hlsTranscodingManifest"`
	HlsTransmuxingManifest string   `json:"hlsTransmuxingManifest"`
}

func MD5Ify(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
