package dash

import (
	"bytes"
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"text/template"
	"time"
)

const dashManifestTemplate = `<?xml version="1.0" encoding="utf-8"?>
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
		<AdaptationSet contentType="video">
			{{ range $si, $s := .videoStream.Representations -}}
			<Representation
					id="{{$s.Representation.RepresentationId}}"
					mimeType="video/mp4"
					codecs="{{$s.Representation.Codecs}}"
					height="480" bandwidth="{{$s.Representation.BitRate}}">
				<SegmentTemplate timescale="1000" duration="5000" initialization="{{$s.Stream.StreamId}}/$RepresentationID$/init.mp4" media="{{$s.Stream.StreamId}}/$RepresentationID$/$Number$.m4s" startNumber="0">
				</SegmentTemplate>
			</Representation>
			{{ end }}
		</AdaptationSet>
		{{ range $i, $audioStream := .audioStreams -}}
		<AdaptationSet contentType="audio" lang="{{ $audioStream.Stream.Language }}">
			{{ range $si, $s := $audioStream.Representations -}}
			<Representation
					id="{{ $s.Representation.RepresentationId }}"
					mimeType="audio/mp4" codecs="mp4a.40.2"
					bandwidth="{{$s.Representation.BitRate}}">
				<SegmentTemplate timescale="1000" duration="4992" initialization="{{$s.Stream.StreamId}}/$RepresentationID$/init.mp4" media="{{$s.Stream.StreamId}}/$RepresentationID$/$Number$.m4s" startNumber="0">
				</SegmentTemplate>
			</Representation>
			{{ end }}
		</AdaptationSet>
		{{ end }}
		{{ range $i, $s := .subtitleStreams -}}
		<AdaptationSet contentType="text" lang="{{ $s.Stream.Language }}" title="lol">
			<Representation id="{{ $s.Representation.RepresentationId }}"
					mimeType="application/mp4" codecs="wvtt">
				<BaseURL>{{ $s.URI }}</BaseURL>
			</Representation>
		</AdaptationSet>
		{{ end }}
	</Period>
</MPD>`

// NOTE(Leon Handreke): Duplicated from hls/manifest.go - factor out, but where to?
type SubtitleStreamRepresentation struct {
	ffmpeg.StreamRepresentation
	URI string
}

type StreamRepresentations struct {
	Stream          ffmpeg.Stream
	Representations []ffmpeg.StreamRepresentation
}

func BuildManifest(
	videoStream StreamRepresentations,
	audioStreams []StreamRepresentations,
	subtitleStreams []SubtitleStreamRepresentation) string {

	totalDuration := videoStream.Stream.TotalDuration.Round(time.Millisecond)
	durationXml := toXmlDuration(totalDuration)

	templateData := map[string]interface{}{
		"videoStream":     videoStream,
		"audioStreams":    audioStreams,
		"subtitleStreams": subtitleStreams,
		"duration":        durationXml,
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(dashManifestTemplate))
	t.Execute(&buf, templateData)
	return buf.String()
}

func toXmlDuration(duration time.Duration) string {
	return fmt.Sprintf("PT%dH%dM%d.%dS",
		duration/time.Hour,
		(duration%time.Hour)/time.Minute,
		(duration%time.Minute)/time.Second,
		(duration%time.Second)/time.Millisecond)
}
