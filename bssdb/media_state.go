package bssdb

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var supportedExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".mov": true,
	".avi": true,
}

type MediaState struct {
	db          *LDBDatabase
	mediaFolder string
}

func NewMediaState(db *LDBDatabase, mediaFolder string) *MediaState {
	return &MediaState{db: db, mediaFolder: mediaFolder}
}

type MediaStateUpdate struct {
	Filename string `json:"filename"`
	Playtime int    `json:"playtime"`
}

func (ms *MediaState) Handler(w http.ResponseWriter, r *http.Request) {
	var msu MediaStateUpdate

	fmt.Println("Body:", r.Body)
	if r.Body == nil {
		fmt.Println("empty body received")
		return
	}

	err := json.NewDecoder(r.Body).Decode(&msu)
	if err != nil {
		fmt.Println("Not a valid media update", err)
	}

	time, err := ms.db.Get([]byte(msu.Filename))
	if err != nil {
		fmt.Println("error getting value from db", err)
		err = ms.db.Put([]byte(msu.Filename), []byte(strconv.Itoa(msu.Playtime)))
		if err != nil {
			fmt.Println("error")
		}
		return
	}

	rtime, err := strconv.Atoi(string(time))
	if err != nil {
		fmt.Println("error parsing time")
	}
	fmt.Println("checking", rtime, "against", msu.Playtime)

	//	if rtime < msu.Playtime {
	fmt.Println("Updating")
	err = ms.db.Put([]byte(msu.Filename), []byte(strconv.Itoa(msu.Playtime)))
	if err != nil {
		fmt.Println("error")
	}
	//	}

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

func (ms *MediaState) ServeFileIndex(w http.ResponseWriter, r *http.Request) {
	files := []MediaFile{}
	err := filepath.Walk(ms.mediaFolder, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(walkPath)] {
			relPath := strings.SplitAfter(walkPath, ms.mediaFolder)
			fileInfo, err := os.Stat(walkPath)

			if err != nil {
				// This catches broken symlinks
				if _, ok := err.(*os.PathError); ok {
					fmt.Println("Got an error while statting file:", err)
					return nil
				}
				return err
			}

			time, err := ms.db.Get([]byte(fileInfo.Name()))
			if err != nil {
				fmt.Println("no state for:", fileInfo.Name())
			}
			rtime, err := strconv.Atoi(string(time))
			if err != nil {
				fmt.Println("error parsing time")
			}

			files = append(files, MediaFile{
				Key:                    MD5Ify(walkPath),
				Name:                   fileInfo.Name(),
				Size:                   fileInfo.Size(),
				Playtime:               rtime,
				HlsTranscodingManifest: path.Join(relPath[1], "hls-transcoding-manifest.m3u8"),
				HlsTransmuxingManifest: path.Join(relPath[1], "/hls-transmuxing-manifest.m3u8")})
		}

		return nil
	})
	if err != nil {
		io.WriteString(w, fmt.Sprintf(`{"error": true, "error_message": "%s"}`, err))
		return
	}

	json.NewEncoder(w).Encode(files)
}
