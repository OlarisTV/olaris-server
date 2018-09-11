package streaming

import (
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/hls"
	"net/http"
	"strconv"
)

func serveHlsMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, "Failed to build media file URL: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !mediaFileURLExists(mediaFileURL) {
		http.NotFound(w, r)
		return
	}

	playableCodecs := r.URL.Query()["playableCodecs"]
	capabilities := ffmpeg.ClientCodecCapabilities{
		PlayableCodecs: playableCodecs,
	}

	videoStream, err := ffmpeg.GetVideoStream(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	videoRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(videoStream, capabilities)

	audioStreams, err := ffmpeg.GetAudioStreams(mediaFileURL)
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

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFileURL)
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(subtitleStreams)

	manifest := hls.BuildMasterPlaylistFromFile(
		[]hls.RepresentationCombination{
			{
				VideoStream:    videoRepresentation,
				AudioStreams:   audioStreamRepresentations,
				AudioGroupName: "audio",
				// TODO(Leon Handreke): Is just using the first one always correct?
				AudioCodecs: audioStreamRepresentations[0].Stream.Codecs,
			},
		},
		subtitleRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTransmuxingMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, "Failed to build media file URL: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !mediaFileURLExists(mediaFileURL) {
		http.NotFound(w, r)
		return
	}

	videoStream, err := ffmpeg.GetVideoStream(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	transmuxedVideoStream, err := ffmpeg.GetTransmuxedRepresentation(videoStream)

	audioStreams, err := ffmpeg.GetAudioStreams(mediaFileURL)
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

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFileURL)
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(subtitleStreams)

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
		subtitleRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, "Failed to build media file URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO(Leon Handreke): Error handling
	audioStreams, _ := ffmpeg.GetAudioStreams(mediaFileURL)

	videoStream, err := ffmpeg.GetVideoStream(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get video streams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	videoRepresentation1, _ := ffmpeg.StreamRepresentationFromRepresentationId(
		videoStream, "preset:480-1000k-video")
	videoRepresentation2, _ := ffmpeg.StreamRepresentationFromRepresentationId(
		videoStream, "preset:720-5000k-video")
	videoRepresentations := []ffmpeg.StreamRepresentation{
		videoRepresentation1, videoRepresentation2}

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
			var audioRepresentation ffmpeg.StreamRepresentation
			if i == 0 {
				audioRepresentation, _ = ffmpeg.StreamRepresentationFromRepresentationId(
					s, "preset:64k-audio")
			} else {
				audioRepresentation, _ = ffmpeg.StreamRepresentationFromRepresentationId(
					s, "preset:128k-audio")
			}
			c.AudioStreams = append(c.AudioStreams, audioRepresentation)
		}
		representationCombinations = append(representationCombinations, c)
	}

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFileURL)
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(subtitleStreams)

	manifest := hls.BuildMasterPlaylistFromFile(
		representationCombinations, subtitleRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMediaPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, "Failed to build media file URL: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !mediaFileURLExists(mediaFileURL) {
		http.NotFound(w, r)
		return
	}

	streamKey, err := getStreamKey(
		mux.Vars(r)["fileLocator"],
		mux.Vars(r)["streamId"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stream, err := ffmpeg.GetStream(streamKey)
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream,
		mux.Vars(r)["representationId"])

	manifest := hls.BuildTranscodingMediaPlaylistFromFile(streamRepresentation)
	w.Write([]byte(manifest))
}
