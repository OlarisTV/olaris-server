package streaming

import (
	"fmt"
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/hls"
	"net/http"
	"strconv"
)

func getMediaFileURLOrFail(r *http.Request) (string, Error) {
	mediaFileURL, err := getMediaFileURL(mux.Vars(r)["fileLocator"])
	if err != nil {
		return "", StatusError{
			Err:  fmt.Errorf("Failed to build media file URL: %s", err.Error()),
			Code: http.StatusInternalServerError,
		}
	}
	if !mediaFileURLExists(mediaFileURL) {
		return "", StatusError{
			Err:  fmt.Errorf("Media file \"%s\" doee not exist.", mediaFileURL),
			Code: http.StatusNotFound,
		}
	}
	return mediaFileURL, nil
}

func serveHlsMasterPlaylist(w http.ResponseWriter, r *http.Request) {
	mediaFileURL, statusErr := getMediaFileURLOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
		return
	}

	playableCodecs := r.URL.Query()["playableCodecs"]
	capabilities := ffmpeg.ClientCodecCapabilities{
		PlayableCodecs: playableCodecs,
	}

	streams, err := ffmpeg.GetStreams(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	videoRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(streams.GetVideoStream(), capabilities)

	audioStreamRepresentations := []ffmpeg.StreamRepresentation{}
	for _, s := range streams.AudioStreams {
		r, err := ffmpeg.GetTransmuxedOrTranscodedRepresentation(s, capabilities)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreamRepresentations = append(audioStreamRepresentations, r)
	}

	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)

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
	mediaFileURL, statusErr := getMediaFileURLOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
	}

	streams, err := ffmpeg.GetStreams(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get streams: "+err.Error(), http.StatusInternalServerError)
		return
	}
	transmuxedVideoStream, err := ffmpeg.GetTransmuxedRepresentation(streams.GetVideoStream())

	audioStreamRepresentations := []ffmpeg.StreamRepresentation{}
	for _, s := range streams.AudioStreams {
		transmuxedStream, err := ffmpeg.GetTransmuxedRepresentation(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreamRepresentations = append(audioStreamRepresentations, transmuxedStream)
	}

	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)

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
	mediaFileURL, statusErr := getMediaFileURLOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
	}

	streams, err := ffmpeg.GetStreams(mediaFileURL)
	if err != nil {
		http.Error(w, "Failed to get streams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	videoRepresentation1, _ := ffmpeg.StreamRepresentationFromRepresentationId(
		streams.GetVideoStream(), "preset:480-1000k-video")
	videoRepresentation2, _ := ffmpeg.StreamRepresentationFromRepresentationId(
		streams.GetVideoStream(), "preset:720-5000k-video")
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
		for _, s := range streams.AudioStreams {
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

	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)

	manifest := hls.BuildMasterPlaylistFromFile(
		representationCombinations, subtitleRepresentations)
	w.Write([]byte(manifest))
}

func serveHlsTranscodingMediaPlaylist(w http.ResponseWriter, r *http.Request) {
	_, statusErr := getMediaFileURLOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
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
