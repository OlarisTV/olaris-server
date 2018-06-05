package streaming

import (
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"gitlab.com/bytesized/bytesized-streaming/hls"
	"net/http"
	"strconv"
)

func serveHlsMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	playableCodecs := r.URL.Query()["playableCodecs"]
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

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFilePath)
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(subtitleStreams)

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
		subtitleRepresentations)
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

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFilePath)
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
	mediaFilePath := getAbsoluteFilepath(mux.Vars(r)["filename"])

	// TODO(Leon Handreke): Error handling
	audioStreams, _ := ffmpeg.GetAudioStreams(mediaFilePath)

	videoStream, err := ffmpeg.GetVideoStream(mediaFilePath)
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

	subtitleStreams, _ := ffmpeg.GetSubtitleStreams(mediaFilePath)
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(subtitleStreams)

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
	stream, err := ffmpeg.GetStream(streamKey)
	streamRepresentation, err := ffmpeg.StreamRepresentationFromRepresentationId(
		stream,
		mux.Vars(r)["representationId"])

	manifest := hls.BuildTranscodingMediaPlaylistFromFile(streamRepresentation)
	w.Write([]byte(manifest))
}
