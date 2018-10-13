package streaming

import (
	"gitlab.com/olaris/olaris-server/dash"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"net/http"
)

func serveDASHManifest(w http.ResponseWriter, r *http.Request) {
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

	// Build lower-quality transcoded versions
	for _, preset := range []string{"preset:480-1000k-video", "preset:720-5000k-video", "preset:1080-10000k-video"} {
		r, _ := ffmpeg.StreamRepresentationFromRepresentationId(
			streams.GetVideoStream(), preset)
		if r.Representation.BitRate < fullQualityRepresentation.Representation.BitRate {
			videoRepresentations = append(videoRepresentations, r)
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

	//subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)

	manifest := dash.BuildManifest(videoRepresentations, audioStreamRepresentations)
	w.Write([]byte(manifest))
}
