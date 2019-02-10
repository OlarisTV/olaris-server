package hls

import (
	"bytes"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"text/template"
)

type RepresentationCombination struct {
	VideoStream    ffmpeg.StreamRepresentation
	AudioStreams   []ffmpeg.StreamRepresentation
	AudioGroupName string
	AudioCodecs    string
}

type SubtitlePlaylistItem struct {
	ffmpeg.StreamRepresentation
	URI string
}

const transcodingMasterPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-INDEPENDENT-SEGMENTS

{{ range $ci, $c := .representationCombinations -}}
{{ range $si, $s := $c.AudioStreams -}}
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="{{$c.AudioGroupName}}",NAME="{{$s.Stream.Title}}",CHANNELS="2",URI="{{$s.Stream.StreamId}}/{{$s.Representation.RepresentationId}}/media.m3u8",AUTOSELECT=YES
{{- if $s.Stream.EnabledByDefault -}}
,DEFAULT=YES
{{ else -}}
,DEFAULT=NO
{{ end -}}
{{ end -}}
{{ end }}

{{ range $i, $s := .subtitlePlaylistItems -}}
#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="webvtt",NAME="{{$s.Stream.Title}}",LANGUAGE="{{$s.Stream.Language}}",AUTOSELECT=YES,URI="{{$s.URI}}"
{{- if $s.Stream.EnabledByDefault -}}
,DEFAULT=YES
{{ else -}}
,DEFAULT=NO
{{ end -}}
{{ end }}

{{ range $ci, $c := .representationCombinations -}}
#EXT-X-STREAM-INF:BANDWIDTH={{$c.VideoStream.Representation.BitRate}},CODECS="{{$c.VideoStream.Representation.Codecs}},{{$c.AudioCodecs}}",AUDIO="{{$c.AudioGroupName}}"
{{- if $.subtitlePlaylistItems -}}
,SUBTITLES="webvtt"
{{- end }}
{{$c.VideoStream.Stream.StreamId}}/{{$c.VideoStream.Representation.RepresentationId}}/media.m3u8
{{ end }}
`

/*
EXT-X-TARGETDURATION must be larger than every segment duration specified in EXTINF,
otherwise iOS won't even bother trying to play the stream.
*/
const transcodingMediaPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:1000
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-MAP:URI="init.mp4"
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
{{ $index }}.m4s{{ end }}
#EXT-X-ENDLIST
`

// The subtitle playlist only contains one "segment",
// therefore the target duration equals the total duration
const subtitleMediaPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:{{.s.TotalDuration.Seconds}}
#EXT-X-PLAYLIST-TYPE:VOD
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
{{ $index }}.vtt{{ end }}
#EXT-X-ENDLIST
`

func BuildMasterPlaylistFromFile(
	representationCombinations []RepresentationCombination,
	subtitlePlaylistItems []SubtitlePlaylistItem) string {

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingMasterPlaylistTemplate))

	t.Execute(&buf, map[string]interface{}{
		"subtitlePlaylistItems":      subtitlePlaylistItems,
		"representationCombinations": representationCombinations,
	})
	return buf.String()
}

func BuildTranscodingMediaPlaylistFromFile(sr ffmpeg.StreamRepresentation) string {
	totalInterval := ffmpeg.Interval{
		TimeBase:       sr.Stream.TimeBase.Denom().Int64(),
		StartTimestamp: 0,
		EndTimestamp:   sr.Stream.TotalDurationDts,
	}
	segmentDurations := ffmpeg.ComputeSegmentDurations(
		[][]ffmpeg.Segment{
			ffmpeg.BuildConstantSegmentDurations(totalInterval, ffmpeg.SegmentDuration, 0),
		})
	segmentDurationsSeconds := []float64{}
	for _, d := range segmentDurations {
		segmentDurationsSeconds = append(segmentDurationsSeconds, d.Seconds())
	}

	templateData := map[string]interface{}{
		"s":                sr.Stream,
		"segmentDurations": segmentDurationsSeconds,
	}

	tmpl := transcodingMediaPlaylistTemplate
	if sr.Stream.StreamType == "subtitle" {
		tmpl = subtitleMediaPlaylistTemplate
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(tmpl))
	t.Execute(&buf, templateData)
	return buf.String()
}
