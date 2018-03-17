package main

//go:generate go-bindata-assetfs -pkg $GOPACKAGE static/

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/dash"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"gitlab.com/bytesized/bytesized-streaming/hls"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var supportedExtensions = map[string]bool{
	".mp4": true,
	".mkv": true,
	".mov": true,
}

var mediaFilesDir = flag.String("media_files_dir", "", "Path to the media files to be served")

var sessions = []*ffmpeg.TranscodingSession{}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

func main() {
	flag.Parse()

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := mux.NewRouter()
	// Currently, we serve these as two different manifests because switching doesn't work at all with misaligned
	// segments.
	r.PathPrefix("/player/").Handler(http.StripPrefix("/player/", http.FileServer(assetFS())))
	r.HandleFunc("/api/v1/files", serveFileIndex)
	r.HandleFunc("/{filename}/transmuxing-manifest.mpd", serveTransmuxingManifest)
	r.HandleFunc("/{filename}/transcoding-manifest.mpd", serveTranscodingManifest)
	r.HandleFunc("/{filename}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingManifest)
	r.HandleFunc("/{filename}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.HandleFunc("/{filename}/{representationId}/init.mp4", serveInit)

	//TODO: (Maran) This is probably not serving subfolders yet
	r.Handle("/", http.FileServer(http.Dir(*mediaFilesDir)))

	srv := &http.Server{Addr: ":8080", Handler: handlers.LoggingHandler(os.Stdout, cors.Default().Handler(r))}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)

	for _, s := range sessions {
		s.Destroy()
	}
}

type MediaFile struct {
	Ext             string `json:"ext"`
	TranscodingPath string `json:"path"`
}

func serveFileIndex(w http.ResponseWriter, r *http.Request) {
	files := []MediaFile{}
	err := filepath.Walk(*mediaFilesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if supportedExtensions[filepath.Ext(path)] {
			relPath := strings.SplitAfter(path, *mediaFilesDir)

			files = append(files, MediaFile{TranscodingPath: relPath[1] + "/transcoding-manifest.mpd"})
			files = append(files, MediaFile{TranscodingPath: relPath[1] + "/transmuxing-manifest.mpd"})
		}

		return nil
	})
	if err != nil {
		io.WriteString(w, `{"error": true}`)
		return
	}

	json.NewEncoder(w).Encode(files)
}

func serveTransmuxingManifest(w http.ResponseWriter, r *http.Request) {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(*mediaFilesDir, mux.Vars(r)["filename"])

	manifest := dash.BuildTransmuxingManifestFromFile(mediaFilePath)
	w.Write([]byte(manifest))
}

func serveHlsTransmuxingManifest(w http.ResponseWriter, r *http.Request) {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(*mediaFilesDir, mux.Vars(r)["filename"])

	manifest := hls.BuildTransmuxingManifestFromFile(mediaFilePath)
	w.Write([]byte(manifest))
}

func serveTranscodingManifest(w http.ResponseWriter, r *http.Request) {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(*mediaFilesDir, mux.Vars(r)["filename"])

	manifest := dash.BuildTranscodingManifestFromFile(mediaFilePath)
	w.Write([]byte(manifest))
}

func splitRepresentationId(representationId string) (string, string, error) {
	separatorIndex := strings.LastIndex(representationId, "-")
	if separatorIndex == -1 {
		return "", "", fmt.Errorf("Invaild representationId, should be representationIdBase-streamId")
	}
	representationIdBase := representationId[:strings.LastIndex(representationId, "-")]
	streamId := representationId[strings.LastIndex(representationId, "-")+1:]
	return representationIdBase, streamId, nil

}

func serveSegment(w http.ResponseWriter, r *http.Request) {
	//sessionId := mux.Vars(r)["sessionId"]
	filename := mux.Vars(r)["filename"]

	representationIdBase, streamId, err := splitRepresentationId(mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	s, _ := getOrStartTranscodingSession(filename, representationIdBase, segmentId)

	segmentPath, err := s.GetSegment(streamId, segmentId, 20*time.Second)
	http.ServeFile(w, r, segmentPath)
}

func getSessions(filename string, representationIdBase string) []*ffmpeg.TranscodingSession {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(*mediaFilesDir, filename)

	matching := []*ffmpeg.TranscodingSession{}

	for _, s := range sessions {
		if s.InputPath == mediaFilePath && s.RepresentationIdBase == representationIdBase {
			matching = append(matching, s)
		}
	}
	return matching
}

func getOrStartTranscodingSession(filename string, representationIdBase string, segmentId int) (*ffmpeg.TranscodingSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var s *ffmpeg.TranscodingSession

	matchingSessions := getSessions(filename, representationIdBase)
	for _, matchingSession := range matchingSessions {
		if matchingSession.IsProjectedAvailable("0", segmentId, 20*time.Second) {
			s = matchingSession
			break
		}
	}

	if s == nil {
		// TODO(Leon Handreke): This probably allows escaping from the directory, look at
		// https://golang.org/src/net/http/fs.go to see how they prevent that.
		mediaFilePath := path.Join(*mediaFilesDir, filename)
		// At the moment, this will always be 0 for direct-stream
		startTime := time.Duration(segmentId) * ffmpeg.MinSegDuration

		if representationIdBase == "direct-stream" {
			s, _ = ffmpeg.NewTransmuxingSession(mediaFilePath, os.TempDir(), startTime, segmentId)
		} else {
			if encoderParams, ok := ffmpeg.EncoderPresets[representationIdBase]; ok {
				s, _ = ffmpeg.NewTranscodingSession(mediaFilePath, os.TempDir(), startTime, segmentId, encoderParams)
			} else {
				return nil, fmt.Errorf("No such encoder preset", representationIdBase)
			}

		}
		s.RepresentationIdBase = representationIdBase
		sessions = append(sessions, s)
		s.Start()
		time.Sleep(2 * time.Second)
	}

	return s, nil
}

func serveInit(w http.ResponseWriter, r *http.Request) {
	filename := mux.Vars(r)["filename"]

	representationIdBase, streamId, err := splitRepresentationId(mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	s, _ := getOrStartTranscodingSession(filename, representationIdBase, 0)

	for {
		initPath, _ := s.InitialSegment(streamId)
		if _, err := os.Stat(initPath); err == nil {
			http.ServeFile(w, r, initPath)
			return

		}
		time.Sleep(500 * time.Millisecond)
	}
}
