package streaming

import (
	"fmt"
	"gitlab.com/olaris/olaris-server/dash"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"net/http"
	"net/url"
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

	videoStream := dash.StreamRepresentations{Stream: streams.GetVideoStream()}
	// Get transmuxed or similar transcoded representation
	fullQualityRepresentation, _ := ffmpeg.GetTransmuxedOrTranscodedRepresentation(streams.GetVideoStream(), capabilities)
	videoStream.Representations = append(videoStream.Representations,
		fullQualityRepresentation)

	// Build lower-quality transcoded versions
	for _, preset := range []string{"preset:480-1000k-video", "preset:720-5000k-video", "preset:1080-10000k-video"} {
		r, _ := ffmpeg.StreamRepresentationFromRepresentationId(
			streams.GetVideoStream(), preset)
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
		// TODO(Leon Handreke): Build a JWT here
		mediaFileURLAsURL, _ := url.Parse(s.Stream.MediaFileURL)
		subtitleStreams = append(subtitleStreams, dash.SubtitleStreamRepresentation{
			StreamRepresentation: s,
			URI: fmt.Sprintf("/s/files%s/%d/%s/0.vtt",
				mediaFileURLAsURL.Path,
				s.Stream.StreamId,
				s.Representation.RepresentationId),
		})
	}

	manifest := dash.BuildManifest(videoStream, audioStreams, subtitleStreams)
	w.Write([]byte(manifest))
}
