package main

//go:generate go-bindata-assetfs -pkg $GOPACKAGE static/...

import (
	"context"
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/peak6/envflag"
	"github.com/rs/cors"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"gitlab.com/bytesized/bytesized-streaming/hls"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"

	"time"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path to the media files to be served")

var sessions = []*ffmpeg.TranscodingSession{}

// Read-modify-write mutex for sessions. This ensures that two parallel requests don't both create a session.
var sessionsMutex = sync.Mutex{}

func main() {
	flag.Parse()
	envflag.Parse()

	mctx := metadata.NewMDContext()
	defer mctx.Db.Close()
	libraryManager := metadata.NewLibraryManager(mctx)
	libraryManager.ActivateAll()

	imageManager := metadata.NewImageManager(mctx)
	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := mux.NewRouter()
	r.PathPrefix("/player").Handler(http.StripPrefix("/player", http.FileServer(assetFS())))
	r.HandleFunc("/api/v1/files", serveFileIndex)
	r.HandleFunc("/api/v1/state", handleSetMediaPlaybackState).Methods("POST")
	r.Handle("/query", metadata.NewRelayHandler(mctx))
	r.Handle("/images/{provider}/{size}/{id}", http.HandlerFunc(imageManager.HttpHandler))
	r.HandleFunc("/graphiql", http.HandlerFunc(metadata.GraphiQLHandler))
	// Currently, we serve these as two different manifests because switching doesn't work at all with misaligned
	// segments.
	r.HandleFunc("/{filename:.*}/hls-transmuxing-manifest.m3u8", serveHlsTransmuxingMasterPlaylist)
	r.HandleFunc("/{filename:.*}/hls-transcoding-manifest.m3u8", serveHlsTranscodingMasterPlaylist)
	r.HandleFunc("/{filename:.*}/hls-manifest.m3u8", serveHlsMasterPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/media.m3u8", serveHlsTranscodingMediaPlaylist)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/{segmentId:[0-9]+}.m4s", serveSegment)
	r.HandleFunc("/{filename:.*}/{streamId}/{representationId}/init.mp4", serveInit)

	//TODO: (Maran) This is probably not serving subfolders yet
	r.Handle("/", http.FileServer(http.Dir(*mediaFilesDir)))

	var handler http.Handler
	handler = r
	handler = cors.AllowAll().Handler(handler)
	handler = handlers.LoggingHandler(os.Stdout, handler)

	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)

	for _, s := range sessions {
		s.Destroy()

	}
}

func serveHlsMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	playableCodecs := r.URL.Query()["playableCodecs"]
	// TODO(Leon Handreke): Get this from the client
	capabilities := ffmpeg.ClientCodecCapabilities{
		PlayableCodecs: playableCodecs,
	}

	videoStream, err := ffmpeg.GetVideoStream(mediaFilePath)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	videoRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(videoStream, capabilities)

	audioStreams, err := ffmpeg.GetAudioStreams(mediaFilePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	audioStreamRepresentations := []ffmpeg.StreamRepresentation{}
	for _, s := range audioStreams {
		r, err := ffmpeg.GetTransmuxedOrTranscodedRepresentation(s, capabilities)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreamRepresentations = append(audioStreamRepresentations, r)
	}

	subtitleStreamRepresentations, _ := ffmpeg.GetSubtitleStreamRepresentations(mediaFilePath)

	manifest := hls.BuildMasterPlaylistFromFile(
		[]hls.RepresentationCombination{
			{
				VideoStream:    videoRepresentation,
				AudioStreams:   audioStreamRepresentations,
				AudioGroupName: "audio",
				// TODO(Leon Handreke): Fill this from the audio codecs.
				AudioCodecs: "mp4a.40.2",
			},
		},
		subtitleStreamRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTransmuxingMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	videoStream, err := ffmpeg.GetVideoStream(mediaFilePath)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	transmuxedVideoStream, err := ffmpeg.GetTransmuxedRepresentation(videoStream)

	audioStreams, err := ffmpeg.GetAudioStreams(mediaFilePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	audioStreamRepresentations := []ffmpeg.StreamRepresentation{}
	for _, s := range audioStreams {
		transmuxedStream, err := ffmpeg.GetTransmuxedRepresentation(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreamRepresentations = append(audioStreamRepresentations, transmuxedStream)
	}

	subtitleStreamRepresentations, _ := ffmpeg.GetSubtitleStreamRepresentations(mediaFilePath)

	manifest := hls.BuildMasterPlaylistFromFile(
		[]hls.RepresentationCombination{
			{
				VideoStream:    transmuxedVideoStream,
				AudioStreams:   audioStreamRepresentations,
				AudioGroupName: "transmuxed",
				// TODO(Leon Handreke): Fill this from the audio codecs.
				AudioCodecs: "mp4a.40.2",
			},
		},
		subtitleStreamRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	audioStreams, _ := ffmpeg.GetAudioStreams(mediaFilePath)
	subtitleRepresentations, _ := ffmpeg.GetSubtitleStreamRepresentations(mediaFilePath)

	videoStream, err := ffmpeg.GetVideoStream(mediaFilePath)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	videoRepresentations := ffmpeg.GetTranscodedVideoRepresentations(videoStream)

	representationCombinations := []hls.RepresentationCombination{}
	for i, r := range videoRepresentations {
		// NOTE(Leon Handreke): This will lead to multiple identical audio groups but whatevs
		audioGroupName := "audio-group-" + strconv.Itoa(i)
		c := hls.RepresentationCombination{
			VideoStream:    r,
			AudioGroupName: audioGroupName,
			AudioCodecs:    "mp4a.40.2",
		}
		for _, s := range audioStreams {
			audioRepresentations := ffmpeg.GetTranscodedAudioRepresentations(s)

			// Lowest audio for lowest video streams and so forth
			// Once we run out of audio, just use the highest one.
			// TODO(Leon Handreke): Do something more sophisticated here instead of relying on
			// the order of the StreamRepresentations
			audioRepresentationIdx := i
			if audioRepresentationIdx > len(audioRepresentations)-1 {
				audioRepresentationIdx = len(audioRepresentations) - 1
			}
			c.AudioStreams = append(c.AudioStreams, audioRepresentations[audioRepresentationIdx])
		}
		representationCombinations = append(representationCombinations, c)
	}

	manifest := hls.BuildMasterPlaylistFromFile(
		representationCombinations, subtitleRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMediaPlaylist(w http.ResponseWriter, r *http.Request) {
	streamKey, err := buildStreamKey(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stream, err := findStream(
		streamKey,
		mux.Vars(r)["representationId"])

	manifest := hls.BuildTranscodingMediaPlaylistFromFile(stream)
	w.Write([]byte(manifest))
}

func serveSegment(w http.ResponseWriter, r *http.Request) {
	segmentId, err := strconv.Atoi(mux.Vars(r)["segmentId"])
	if err != nil {
		http.Error(w, "Invalid segmentId", http.StatusBadRequest)
	}

	streamKey, err := buildStreamKey(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stream, err := findStream(
		streamKey,
		mux.Vars(r)["representationId"])

	session, _ := getOrStartTranscodingSession(stream, int64(segmentId))

	segmentPath, err := session.GetSegment(int64(segmentId), 20*time.Second)
	http.ServeFile(w, r, segmentPath)
}

func getSessions(streamKey ffmpeg.StreamKey, representationId string) []*ffmpeg.TranscodingSession {
	matching := []*ffmpeg.TranscodingSession{}

	for _, s := range sessions {
		if s.Stream.Stream.StreamKey == streamKey && s.Stream.Representation.RepresentationId == representationId {
			matching = append(matching, s)
		}
	}
	return matching
}

func getOrStartTranscodingSession(stream ffmpeg.StreamRepresentation, segmentId int64) (*ffmpeg.TranscodingSession, error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	var s *ffmpeg.TranscodingSession

	representationId := stream.Representation.RepresentationId

	matchingSessions := getSessions(stream.Stream.StreamKey, representationId)
	for _, matchingSession := range matchingSessions {
		if matchingSession.IsProjectedAvailable(segmentId, 20*time.Second) {
			s = matchingSession
			break
		}
	}

	if s == nil {
		var err error
		if stream.Representation.RepresentationId == "direct" {
			s, err = ffmpeg.NewTransmuxingSession(stream, os.TempDir(), segmentId)
		} else {
			if strings.Contains(stream.Representation.RepresentationId, "video") {
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
			if strings.Contains(representationId, "webvtt") {
				s, err = ffmpeg.NewSubtitleSession(stream, os.TempDir())
			}

		}
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, s)
		s.Start()
	}

	return s, nil
}

func serveInit(w http.ResponseWriter, r *http.Request) {
	streamKey, err := buildStreamKey(
		mux.Vars(r)["filename"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stream, err := findStream(
		streamKey,
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

func buildStreamKey(filename string, streamIdStr string) (ffmpeg.StreamKey, error) {
	streamId, err := strconv.Atoi(streamIdStr)
	if err != nil {
		return ffmpeg.StreamKey{}, err
	}

	return ffmpeg.StreamKey{
		StreamId:      int64(streamId),
		MediaFilePath: getAbsoluteFilepath(filename),
	}, nil
}

func findStream(streamKey ffmpeg.StreamKey, representationId string) (ffmpeg.StreamRepresentation, error) {
	videoStreams, err := ffmpeg.GetVideoStreams(streamKey.MediaFilePath)
	if err != nil {
		return ffmpeg.StreamRepresentation{}, err
	}
	audioStreams, err := ffmpeg.GetAudioStreams(streamKey.MediaFilePath)
	if err != nil {
		return ffmpeg.StreamRepresentation{}, err
	}

	for _, s := range append(videoStreams, audioStreams...) {
		if s.StreamKey != streamKey {
			continue
		}

		transmuxedStream, err := ffmpeg.GetTransmuxedRepresentation(s)
		if err != nil {
			return ffmpeg.StreamRepresentation{}, err
		}
		if transmuxedStream.Representation.RepresentationId == representationId {
			return transmuxedStream, nil
		}

		transcodedRepresentations := []ffmpeg.StreamRepresentation{}
		if strings.Contains(representationId, "audio") {
			transcodedRepresentations = ffmpeg.GetTranscodedAudioRepresentations(s)
		} else if strings.Contains(representationId, "video") {
			transcodedRepresentations = ffmpeg.GetTranscodedVideoRepresentations(s)
		}
		for _, r := range transcodedRepresentations {
			if r.Representation.RepresentationId == representationId {
				return r, nil
			}
		}
	}

	subtitleStreams, err := ffmpeg.GetSubtitleStreamRepresentations(streamKey.MediaFilePath)
	if err != nil {
		return ffmpeg.StreamRepresentation{}, err
	}
	for _, s := range subtitleStreams {
		if s.Stream.StreamKey != streamKey {
			continue
		}
		return s, nil
	}

	return ffmpeg.StreamRepresentation{},
		fmt.Errorf("No such stream %s/%s found for file %s",
			streamKey.StreamId, representationId, streamKey.MediaFilePath)

}
