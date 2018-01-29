package main

import(
	"github.com/gorilla/mux"
	"net/http"
	"path"
	"bytesized-hosting.com/media/streaming/ffmpeg"
	"log"
	"encoding/json"
	"strconv"
	"os"
	"time"
	"context"
	"os/signal"
)

const mediaFilesDir = "/home/leon/Videos"
const cacheDir = "/tmp"
const segmentDuration = 5

// TODO(Leon Handreke): Get rid of the singleton pattern.
var sessions = make(map[string]*ffmpeg.TranscodingSession)

func main() {
	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	r := mux.NewRouter()
	r.HandleFunc("/{filename}/{sessionId}/manifest.mpd", serveManifest)
	r.HandleFunc("/{filename}/{sessionId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.Handle("/", http.FileServer(http.Dir(mediaFilesDir)))

	srv := &http.Server{Addr: ":8080", Handler: r}
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

	probeDataJson, err := json.Marshal(probeData)
	if err != nil {
		log.Fatal("Failed to encode probe data as JSON")
	}
	w.Write(probeDataJson)
}

func serveSegment(w http.ResponseWriter, r *http.Request) {
	sessionId := mux.Vars(r)["sessionId"]
	s := sessions[sessionId]
	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	if s != nil {
		availableSegments, _ := s.AvailableSegments()
		if _, ok := availableSegments[segmentId]; !ok {
			go s.Destroy()
			s = nil
		}
	}

	if s == nil {
		// TODO(Leon Handreke): This probably allows escaping from the directory, look at
		// https://golang.org/src/net/http/fs.go to see how they prevent that.
		mediaFilePath := path.Join(mediaFilesDir, mux.Vars(r)["filename"])
		startTime := int64(segmentId * segmentDuration)
		s, _ = ffmpeg.NewTranscodingSession(mediaFilePath, os.TempDir(), startTime, segmentId - 1)
		sessions[mux.Vars(r)["sessionId"]] = s
		s.Start()

	}

	for {
		availableSegments, _ := s.AvailableSegments()
		if path, ok := availableSegments[segmentId]; ok {
			http.ServeFile(w, r, path)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}