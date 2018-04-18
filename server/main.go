package main

//go:generate go-bindata-assetfs -pkg $GOPACKAGE static/...

import (
	"context"
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/bssdb"
	"gitlab.com/bytesized/bytesized-streaming/dash"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"gitlab.com/bytesized/bytesized-streaming/hls"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mediaFilesDir = flag.String("media_files_dir", "", "Path to the media files to be served")

var sessions = []*ffmpeg.TranscodingSession{}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

func main() {
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Can't get user's home folder.", err)
	}

	ldb, err := bssdb.NewDb(path.Join(usr.HomeDir, ".config", "bss", "db"))
	ms := bssdb.NewMediaState(ldb, *mediaFilesDir)
	defer ldb.Close()

	if err != nil {
		fmt.Println("can't open db", err)
		os.Exit(1)
	}

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := mux.NewRouter()
	// Currently, we serve these as two different manifests because switching doesn't work at all with misaligned
	// segments.
	r.PathPrefix("/player/").Handler(http.StripPrefix("/player/", http.FileServer(assetFS())))
	r.HandleFunc("/api/v1/files", ms.ServeFileIndex)
	r.HandleFunc("/api/v1/state", ms.Handler)
	r.HandleFunc("/{filename:.*}/transmuxing-manifest.mpd", serveTransmuxingManifest)
	r.HandleFunc("/{filename:.*}/transcoding-manifest.mpd", serveTranscodingManifest)
	r.HandleFunc("/{filename:.*}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingManifest)
	r.HandleFunc("/{filename:.*}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/init.mp4", serveInit)

	//TODO: (Maran) This is probably not serving subfolders yet
	r.Handle("/", http.FileServer(http.Dir(*mediaFilesDir)))

	var handler http.Handler
	handler = r
	handler = cors.AllowAll().Handler(handler)
	handler = handlers.LoggingHandler(os.Stdout, handler)

	srv := &http.Server{Addr: ":8080", Handler: handler}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)

	for _, s := range sessions {
		s.Destroy()
	}
}

func serveTransmuxingManifest(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	manifest := dash.BuildTransmuxingManifestFromFile(mediaFilePath)
	w.Write([]byte(manifest))
}

func serveHlsTransmuxingManifest(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	offeredStreams, _ := ffmpeg.GetOfferedTransmuxedStreams(mediaFilePath)
	manifest := hls.BuildTransmuxingMasterPlaylistFromFile(offeredStreams)
	w.Write([]byte(manifest))
}

func serveTranscodingManifest(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	manifest := dash.BuildTranscodingManifestFromFile(mediaFilePath)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	offeredStreams, _ := ffmpeg.GetOfferedTranscodedStreams(mediaFilePath)
	manifest := hls.BuildTranscodingMasterPlaylistFromFile(offeredStreams)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMediaPlaylist(w http.ResponseWriter, r *http.Request) {
	stream, err := findStream(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"],
		mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	manifest := hls.BuildTranscodingMediaPlaylistFromFile(stream)
	w.Write([]byte(manifest))
}

func serveSegment(w http.ResponseWriter, r *http.Request) {
	stream, err := findStream(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"],
		mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}
	session, _ := getOrStartTranscodingSession(stream, int64(segmentId))

	segmentPath, err := session.GetSegment(int64(segmentId), 20*time.Second)
	http.ServeFile(w, r, segmentPath)
}

func getSessions(streamKey ffmpeg.StreamKey) []*ffmpeg.TranscodingSession {
	matching := []*ffmpeg.TranscodingSession{}

	for _, s := range sessions {
		if s.Stream.StreamKey == streamKey {
			matching = append(matching, s)
		}
	}
	return matching
}

func getOrStartTranscodingSession(stream ffmpeg.OfferedStream, segmentId int64) (*ffmpeg.TranscodingSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var s *ffmpeg.TranscodingSession

	matchingSessions := getSessions(stream.StreamKey)
	for _, matchingSession := range matchingSessions {
		if matchingSession.IsProjectedAvailable(segmentId, 20*time.Second) {
			s = matchingSession
			break
		}
	}

	representationId := stream.RepresentationId

	if s == nil {
		var err error
		if stream.RepresentationId == "direct-stream-video" {
			s, err = ffmpeg.NewTransmuxingSession(stream, os.TempDir(), segmentId)
		} else {
			if strings.Contains(stream.RepresentationId, "video") {
				if encoderParams, ok := ffmpeg.VideoEncoderPresets[representationId]; ok {
					s, err = ffmpeg.NewVideoTranscodingSession(
						stream, os.TempDir(), segmentId, encoderParams)
				} else {
					return nil, fmt.Errorf("No such encoder preset %s", representationId)
				}
			}
			if strings.Contains(representationId, "audio") {
				if encoderParams, ok := ffmpeg.AudioEncoderPresets[representationId]; ok {
					s, err = ffmpeg.NewAudioTranscodingSession(
						stream, os.TempDir(), segmentId, encoderParams)
				} else {
					return nil, fmt.Errorf("No such encoder preset %s", representationId)
				}

			}

		}
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, s)
		s.Start()
		time.Sleep(2 * time.Second)
	}

	return s, nil
}

func serveInit(w http.ResponseWriter, r *http.Request) {
	stream, err := findStream(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"],
		mux.Vars(r)["representationId"])
	session, err := getOrStartTranscodingSession(stream, 0)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for {
		initPath := session.InitialSegment()
		if _, err := os.Stat(initPath); err == nil {
			http.ServeFile(w, r, initPath)
			return

		}
		time.Sleep(500 * time.Millisecond)
	}
}

func getAbsoluteFilepath(filename string) string {
	return path.Join(*mediaFilesDir, path.Clean(filename))
}

func findStream(filename string, streamIdStr string, representationId string) (ffmpeg.OfferedStream, error) {
	streamId, _ := strconv.Atoi(streamIdStr)

	streams, err := ffmpeg.GetOfferedStreams(getAbsoluteFilepath(filename))
	if err != nil {
		return ffmpeg.OfferedStream{}, err
	}

	if stream, found := ffmpeg.FindStream(streams, int64(streamId), representationId); found {
		return stream, nil
	}
	return ffmpeg.OfferedStream{},
		fmt.Errorf("No such stream %s/%s found for file %s", streamIdStr, representationId, filename)

}
