package main

import (
	"bytesized-hosting.com/media/streaming/dash"
	"bytesized-hosting.com/media/streaming/ffmpeg"
	"context"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const mediaFilesDir = "/home/leon/Videos"
const cacheDir = "/tmp"
const minSegDuration = time.Duration(5 * time.Second)

// TODO(Leon Handreke): Get rid of the singleton pattern.
var sessions = make(map[string]*ffmpeg.TranscodingSession)

// TODO(Leon Handreke): Enable concurrency once we've figured out how to let TranscodingSessions report what they *almost* have
var requestMutex = sync.Mutex{}

func main() {
	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	r := mux.NewRouter()
	r.HandleFunc("/{filename}/{sessionId}/manifest.mpd", serveManifest)
	r.HandleFunc("/{filename}/{sessionId}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.HandleFunc("/{filename}/{sessionId}/{representationId}/init.mp4", serveInit)
	r.Handle("/", http.FileServer(http.Dir(mediaFilesDir)))

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

func serveManifest(w http.ResponseWriter, r *http.Request) {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(mediaFilesDir, mux.Vars(r)["filename"])

	probeData, err := ffmpeg.Probe(mediaFilePath)
	if err != nil {
		log.Fatal("Failed to ffprobe %s", mediaFilePath)
	}

	totalDuration := probeData.Format.Duration().Round(time.Millisecond)

	keyframes, err := ffmpeg.ProbeKeyframes(mediaFilePath)
	if err != nil {
		log.Fatal("Failed to ffprobe %s", mediaFilePath)
	}

	manifest := dash.BuildManifest(ffmpeg.GuessSegmentDurations(keyframes, minSegDuration), totalDuration)
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
	requestMutex.Lock()
	defer requestMutex.Unlock()

	sessionId := mux.Vars(r)["sessionId"]
	filename := mux.Vars(r)["filename"]

	representationIdBase, streamId, err := splitRepresentationId(mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	s, _ := getOrStartTranscodingSession(sessionId, filename, representationIdBase, segmentId)

	availableSegments, _ := s.AvailableSegments(streamId)
	if _, ok := availableSegments[segmentId]; !ok {
		go s.Destroy()
		sessions[sessionId] = nil
		s, _ = getOrStartTranscodingSession(sessionId, filename, representationIdBase, segmentId)
	}

	for {
		availableSegments, _ := s.AvailableSegments(streamId)
		if path, ok := availableSegments[segmentId]; ok {
			http.ServeFile(w, r, path)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func getOrStartTranscodingSession(sessionId string, filename string, representationIdBase string, segmentId int) (*ffmpeg.TranscodingSession, error) {
	s := sessions[sessionId]

	if s == nil {
		// TODO(Leon Handreke): This probably allows escaping from the directory, look at
		// https://golang.org/src/net/http/fs.go to see how they prevent that.
		mediaFilePath := path.Join(mediaFilesDir, filename)
		startTime := int64(segmentId) * int64(minSegDuration.Seconds())
		s, _ = ffmpeg.NewTranscodingSession(mediaFilePath, os.TempDir(), startTime, segmentId-1)
		sessions[sessionId] = s
		s.Start()
		time.Sleep(2 * time.Second)
	}

	return s, nil
}

func serveInit(w http.ResponseWriter, r *http.Request) {
	requestMutex.Lock()
	defer requestMutex.Unlock()

	sessionId := mux.Vars(r)["sessionId"]
	filename := mux.Vars(r)["filename"]

	representationIdBase, streamId, err := splitRepresentationId(mux.Vars(r)["representationId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	s, _ := getOrStartTranscodingSession(sessionId, filename, representationIdBase, 0)

	for {
		initPath, _ := s.InitialSegment(streamId)
		if _, err := os.Stat(initPath); err == nil {
			http.ServeFile(w, r, initPath)
			return

		}
		time.Sleep(500 * time.Millisecond)
	}
}
