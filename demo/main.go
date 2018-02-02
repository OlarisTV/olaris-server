package main

import(
	"github.com/gorilla/mux"
	"net/http"
	"path"
	"bytesized-hosting.com/media/streaming/ffmpeg"
	"log"
	"strconv"
	"os"
	"time"
	"context"
	"os/signal"
	"text/template"
	"strings"
	"fmt"
	"github.com/rs/cors"
	"github.com/gorilla/handlers"
	"sync"
)

const mediaFilesDir = "/home/leon/Videos"
const cacheDir = "/tmp"
const segmentDuration = 5

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

const manifestTemplate = `<?xml version="1.0" encoding="utf-8"?>
<MPD xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xmlns="urn:mpeg:dash:schema:mpd:2011"
	xmlns:xlink="http://www.w3.org/1999/xlink"
	xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 http://standards.iso.org/ittf/PubliclyAvailableStandards/MPEG-DASH_schema_files/DASH-MPD.xsd"
	profiles="urn:mpeg:dash:profile:isoff-live:2011"
	type="static"
	mediaPresentationDuration="{{ .duration }}"
	maxSegmentDuration="PT10S"
	minBufferTime="PT30S">
	<Period start="PT0S" id="0" duration="{{ .duration }}">
		<AdaptationSet segmentAlignment="true" contentType="video">
			<SegmentTemplate timescale="1" duration="5" initialization="$RepresentationID$/init.mp4" media="$RepresentationID$/$Number$.m4s" startNumber="0">
			</SegmentTemplate>
			<Representation id="direct-stream-video" mimeType="video/mp4" codecs="avc1.64001e" width="1024" height="552">
			</Representation>
		</AdaptationSet>
		<AdaptationSet segmentAlignment="true" contentType="audio">
			<SegmentTemplate timescale="1" duration="5" initialization="$RepresentationID$/init.mp4" media="$RepresentationID$/$Number$.m4s" startNumber="0">
			</SegmentTemplate>
			<Representation id="direct-stream-audio" mimeType="audio/mp4" codecs="mp4a.40.2" bandwidth="0" audioSamplingRate="48000">
			</Representation>
		</AdaptationSet>
	</Period>
</MPD>`

func serveManifest(w http.ResponseWriter, r *http.Request) {
	// TODO(Leon Handreke): This probably allows escaping from the directory, look at
	// https://golang.org/src/net/http/fs.go to see how they prevent that.
	mediaFilePath := path.Join(mediaFilesDir, mux.Vars(r)["filename"])

	probeData, err := ffmpeg.Probe(mediaFilePath)
	if err != nil {
		log.Fatal("Failed to ffprobe %s", mediaFilePath)
	}

	d := probeData.Format.Duration().Round(time.Millisecond)
	durationXml := fmt.Sprintf("PT%dH%dM%d.%dS",
		d / time.Hour,
		(d % time.Hour) / time.Minute,
		(d % time.Minute) / time.Second,
		(d % time.Second) / time.Millisecond)

	t := template.Must(template.New("manifest").Parse(manifestTemplate))
	templateData := map[string]string{"duration": durationXml}
	t.Execute(w, templateData)
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

func getOrStartTranscodingSession(sessionId string, filename string, representationIdBase string, segmentId int) (*ffmpeg.TranscodingSession, error){
	s := sessions[sessionId]

	if s == nil {
		// TODO(Leon Handreke): This probably allows escaping from the directory, look at
		// https://golang.org/src/net/http/fs.go to see how they prevent that.
		mediaFilePath := path.Join(mediaFilesDir, filename)
		startTime := int64(segmentId * segmentDuration)
		s, _ = ffmpeg.NewTranscodingSession(mediaFilePath, os.TempDir(), startTime, segmentId - 1)
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


