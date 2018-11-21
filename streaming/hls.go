package streaming

import (
	"fmt"
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/hls"
	"gitlab.com/olaris/olaris-server/metadata/auth"
	"net/http"
	"net/url"
	"strconv"
)

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

	// Get transmuxed or similar transcoded representation
	fullQualityRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(streams.GetVideoStream(), capabilities)
	videoRepresentations := []ffmpeg.StreamRepresentation{fullQualityRepresentation}

	// TODO(Leon Handreke): I've observed issues with switching from transmuxed representations to transcoded
	// (garbled output). Therefore, serve alternative streams only for transcoded for now. See
	// https://gitlab.com/olaris/olaris-server/issues/48
	if fullQualityRepresentation.Representation.Transcoded {
		// Build lower-quality transcoded versions
		for _, preset := range []string{"preset:480-1000k-video", "preset:720-5000k-video", "preset:1080-10000k-video"} {
			r, _ := ffmpeg.StreamRepresentationFromRepresentationId(
				streams.GetVideoStream(), preset)
			if r.Representation.BitRate < fullQualityRepresentation.Representation.BitRate {
				videoRepresentations = append(videoRepresentations, r)
			}
		}
	}

	audioStreamRepresentations := []ffmpeg.StreamRepresentation{}
	for _, s := range streams.AudioStreams {
		r, err := ffmpeg.GetTransmuxedOrTranscodedRepresentation(s, capabilities)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreamRepresentations = append(audioStreamRepresentations, r)
	}

	combinations := []hls.RepresentationCombination{}
	for _, v := range videoRepresentations {
		combinations = append(combinations, hls.RepresentationCombination{
			VideoStream:    v,
			AudioStreams:   audioStreamRepresentations,
			AudioGroupName: "audio",
			// TODO(Leon Handreke): Is just using the first one always correct?
			AudioCodecs: audioStreamRepresentations[0].Stream.Codecs,
		})
	}

	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)
	subtitlePlaylistItems := buildSubtitlePlaylistItems(subtitleRepresentations, mux.Vars(r)["sessionID"])

	manifest := hls.BuildMasterPlaylistFromFile(combinations, subtitlePlaylistItems)
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
	subtitlePlaylistItems := buildSubtitlePlaylistItems(subtitleRepresentations, mux.Vars(r)["sessionID"])

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
		subtitlePlaylistItems)
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
	subtitlePlaylistItems := buildSubtitlePlaylistItems(subtitleRepresentations, mux.Vars(r)["sessionID"])

	manifest := hls.BuildMasterPlaylistFromFile(
		representationCombinations, subtitlePlaylistItems)
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

func buildSubtitlePlaylistItems(representations []ffmpeg.StreamRepresentation, sessionID string) []hls.SubtitlePlaylistItem {
	// Subtitles may be in another file, so we need to list their absolute URI.
	subtitlePlaylistItems := []hls.SubtitlePlaylistItem{}
	for _, s := range representations {
		mediaFileURL, _ := url.Parse(s.Stream.MediaFileURL)
		// NOTE(Leon Handreke): Because we'd have to propagate the UserID here through
		// context or something like that and it's not used anyway, just use 0 here.
		jwt, _ := auth.CreateStreamingJWT(0, mediaFileURL.Path)
		subtitlePlaylistItems = append(subtitlePlaylistItems,
			hls.SubtitlePlaylistItem{
				StreamRepresentation: s,
				URI: fmt.Sprintf("/s/files/jwt/%s/%s/%d/%s/media.m3u8",
					jwt,
					sessionID,
					s.Stream.StreamId,
					s.Representation.RepresentationId),
			})
	}
	return subtitlePlaylistItems
}
