package hls

import (
	"bytes"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"text/template"
)

/*
EXT-X-TARGETDURATION must be larger than every segment duration specified in EXTINF,
otherwise iOS won't even bother trying to play the stream.
*/
const transmuxingMasterPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:100
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-MAP:URI="{{.s.StreamId}}/{{.s.RepresentationId}}/init.mp4"
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
{{$.s.StreamId}}/{{$.s.RepresentationId}}/{{ $index }}.m4s{{ end }}
#EXT-X-ENDLIST
`

type VideoStreamCombination struct {
	Video       ffmpeg.OfferedStream
	AudioGroup  string
	AudioCodecs string
}

const transcodingMasterPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-INDEPENDENT-SEGMENTS

{{ range $index, $s := .audioStreams -}}
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="{{$s.RepresentationId}}",NAME="{{$s.Title}}",CHANNELS="2",URI="{{$s.StreamId}}/{{$s.RepresentationId}}/media.m3u8",AUTOSELECT=YES
{{- if $s.EnabledByDefault -}}
,DEFAULT=YES
{{ else -}}
,DEFAULT=NO
{{ end -}}
{{ end }}

{{ range $index, $s := .subtitleStreams -}}
#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="webvtt",NAME="{{$s.Title}}",LANGUAGE="{{$s.Language}}",AUTOSELECT=YES,URI="{{$s.StreamId}}/{{$s.RepresentationId}}/media.m3u8"
{{- if $s.EnabledByDefault -}}
,DEFAULT=YES
{{ else -}}
,DEFAULT=NO
{{ end -}}
{{ end }}

{{ range $index, $c := .streamCombinations -}}
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH={{ $c.Video.BitRate }},CODECS="{{$c.Video.Codecs}},{{$c.AudioCodecs}}",AUDIO="{{$c.AudioGroup}}",SUBTITLES="webvtt"
{{$c.Video.StreamId}}/{{$c.Video.RepresentationId}}/media.m3u8
{{ end }}
`

const transcodingMediaPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-TARGETDURATION:5
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
{{ $index }}.m4s{{ end }}
#EXT-X-ENDLIST
`

func BuildTransmuxingMasterPlaylistFromFile(streams []ffmpeg.OfferedStream) string {
	stream := streams[0]

	segmentDurations := ffmpeg.ComputeSegmentDurations(stream.SegmentStartTimestamps, stream.TotalDuration)
	segmentDurationsSeconds := []float64{}
	for _, d := range segmentDurations {
		segmentDurationsSeconds = append(segmentDurationsSeconds, d.Seconds())
	}

	templateData := map[string]interface{}{
		"s":                stream,
		"segmentDurations": segmentDurationsSeconds,
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transmuxingMasterPlaylistTemplate))
	t.Execute(&buf, templateData)
	return buf.String()
}

func BuildTranscodingMasterPlaylistFromFile(streams []ffmpeg.OfferedStream) string {
	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingMasterPlaylistTemplate))

	audioStreams := []ffmpeg.OfferedStream{}
	videoStreams := []ffmpeg.OfferedStream{}
	subtitleStreams := []ffmpeg.OfferedStream{}
	for _, stream := range streams {
		if stream.StreamType == "audio" {
			audioStreams = append(audioStreams, stream)
		} else if stream.StreamType == "video" {
			videoStreams = append(videoStreams, stream)
		} else if stream.StreamType == "subtitle" {
			subtitleStreams = append(subtitleStreams, stream)
		}
	}
	// TODO(Leon Handreke): Have some smart heuristic here to match audio and video streams.
	streamCombinations := []VideoStreamCombination{
		{
			Video:       videoStreams[0],
			AudioCodecs: audioStreams[0].Codecs,
			AudioGroup:  audioStreams[0].RepresentationId,
		},
	}

	t.Execute(&buf, map[string]interface{}{
		"videoStreams":       videoStreams,
		"audioStreams":       audioStreams,
		"subtitleStreams":    subtitleStreams,
		"streamCombinations": streamCombinations,
	})
	return buf.String()
}

func BuildTranscodingMediaPlaylistFromFile(stream ffmpeg.OfferedStream) string {

	segmentDurations := ffmpeg.ComputeSegmentDurations(stream.SegmentStartTimestamps, stream.TotalDuration)
	segmentDurationsSeconds := []float64{}
	for _, d := range segmentDurations {
		segmentDurationsSeconds = append(segmentDurationsSeconds, d.Seconds())
	}

	templateData := map[string]interface{}{
		"s":                stream,
		"segmentDurations": segmentDurationsSeconds,
	}

	tmpl := transcodingMediaPlaylistTemplate
	if stream.StreamType == "subtitle" {
		tmpl = subtitleMediaPlaylistTemplate
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(tmpl))
	t.Execute(&buf, templateData)
	return buf.String()
}
