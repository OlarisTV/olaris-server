package streaming

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/dash"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata/auth"
)

func serveDASHManifest(w http.ResponseWriter, r *http.Request) {
	fileLocator, statusErr := getFileLocatorOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
		return
	}

	playableCodecs := r.URL.Query()["playableCodecs"]
	capabilities := ffmpeg.ClientCodecCapabilities{
		PlayableCodecs: playableCodecs,
	}

	streams, err := ffmpeg.GetStreams(fileLocator)
	if err != nil {
		http.Error(w, "Failed to get streams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	videoStream := dash.StreamRepresentations{Stream: streams.GetVideoStream()}
	// Get transmuxed or similar transcoded representation
	fullQualityRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(streams.GetVideoStream(), capabilities)
	videoStream.Representations = append(videoStream.Representations, fullQualityRepresentation)

	lowQualityRepresentations := ffmpeg.GetStandardPresetVideoRepresentations(streams.GetVideoStream())
	for _, r := range lowQualityRepresentations {
		if r.Representation.BitRate < fullQualityRepresentation.Representation.BitRate {
			videoStream.Representations = append(videoStream.Representations, r)
		}
	}

	audioStreams := []dash.StreamRepresentations{}
	for _, s := range streams.AudioStreams {
		r, err := ffmpeg.GetTransmuxedOrTranscodedRepresentation(s, capabilities)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audioStreams = append(audioStreams,
			dash.StreamRepresentations{
				Stream:          s,
				Representations: []ffmpeg.StreamRepresentation{r}})

	}

	subtitleStreams := []dash.SubtitleStreamRepresentation{}
	subtitleRepresentations := ffmpeg.GetSubtitleStreamRepresentations(streams.SubtitleStreams)
	for _, s := range subtitleRepresentations {
		// NOTE(Leon Handreke): Because we'd have to propagate the UserID here through
		// context or something like that and it's not used anyway, just use 0 here.
		// We need to use s.Stream.FileLocator here because the subtitle file may be external
		// next to the video file.
		jwt, err := auth.CreateStreamingJWT(0, s.Stream.FileLocator.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		subtitleStreams = append(subtitleStreams, dash.SubtitleStreamRepresentation{
			StreamRepresentation: s,
			// TODO(Maran) It would be better to somehow pass routing information along and not hard-code this in place.
			URI: fmt.Sprintf("/olaris/s/files/jwt/%s/%s/%d/%s/0.vtt",
				jwt,
				mux.Vars(r)["sessionID"],
				s.Stream.StreamId,
				s.Representation.RepresentationId),
		})
	}

	manifest := dash.BuildManifest(videoStream, audioStreams, subtitleStreams)
	w.Write([]byte(manifest))
}
