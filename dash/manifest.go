package dash

import (
	"bytes"
	"fmt"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"text/template"
	"time"
)

const transcodingManifestTemplate = `<?xml version="1.0" encoding="utf-8"?>
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
			{{ range $si, $s := .videoStreams -}}
			<Representation
					id="{{$s.Representation.RepresentationId}}"
					mimeType="video/mp4"
					codecs="{{$s.Representation.Codecs}}"
					height="480" bandwidth="{{$s.Representation.BitRate}}">
				<SegmentTemplate timescale="1000" duration="5000" initialization="{{$s.Stream.StreamId}}/$RepresentationID$/init.mp4" media="{{$s.Stream.StreamId}}/$RepresentationID$/$Number$.m4s" startNumber="0">
					<SegmentTimeline>
						{{ range $index, $d := $s.SegmentDurationsMilliseconds -}}
						<S {{ if eq $index 0}}t="0" {{ end }}d="{{ $d }}"></S> <!-- {{ $index }} -->
						{{ end }}
					</SegmentTimeline>
				</SegmentTemplate>
			</Representation>
			{{- end }}
		</AdaptationSet>
		<AdaptationSet contentType="audio">
			{{ range $si, $s := .audioStreams -}}
			<Representation id="{{ $s.Representation.RepresentationId }}"
					mimeType="audio/mp4" codecs="mp4a.40.2"
					bandwidth="{{$s.Representation.BitRate}}">
				<SegmentTemplate timescale="1000" duration="4992" initialization="{{$s.Stream.StreamId}}/$RepresentationID$/init.mp4" media="{{$s.Stream.StreamId}}/$RepresentationID$/$Number$.m4s" startNumber="0">
				</SegmentTemplate>
			</Representation>
			{{ end }}
		</AdaptationSet>
	</Period>
</MPD>`

func BuildManifest(
	videoStreams []ffmpeg.StreamRepresentation,
	audioStreams []ffmpeg.StreamRepresentation) string {

	totalDuration := videoStreams[0].Stream.TotalDuration.Round(time.Millisecond)
	durationXml := toXmlDuration(totalDuration)

	templateData := map[string]interface{}{
		"videoStreams": videoStreams,
		"audioStreams": audioStreams,
		"duration":     durationXml,
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingManifestTemplate))
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
