package hls

import (
	"bytes"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"log"
	"text/template"
	"time"
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
#EXT-X-MAP:URI="0/direct-stream-video/init.mp4"
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
0/direct-stream-video/{{ $index }}.m4s{{ end }}
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

{{ range $index, $c := .streamCombinations }}
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH={{ $c.Video.BitRate }},CODECS="{{$c.Video.Codecs}},{{$c.AudioCodecs}}",AUDIO="{{$c.AudioGroup}}"
{{$c.Video.StreamId}}/{{$c.Video.RepresentationId}}/media.m3u8
{{ end }}

{{ range $index, $s := .audioStreams }}
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="{{$s.RepresentationId}}",NAME="{{$s.Title}}",CHANNELS="2",URI="{{$s.StreamId}}/{{$s.RepresentationId}}/media.m3u8"
{{ end }}
`

/*
,AUDIO="64k"
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=5000000,CODECS="avc1.64001f"
720-5000k-video/media.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=10000000,CODECS="avc1.640028"
1080-10000k-video/media.m3u8
*/

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

func BuildTransmuxingMasterPlaylistFromFile(filePath string) string {
	probeData, err := ffmpeg.Probe(filePath)
	if err != nil {
		log.Fatal("Failed to ffprobe", filePath)
	}

	totalDuration := probeData.Format.Duration().Round(time.Millisecond)

	keyframes, err := ffmpeg.ProbeKeyframes(filePath)
	if err != nil {
		log.Fatal("Failed to ffprobe", filePath)
	}
	segmentDurations := ffmpeg.GuessSegmentDurations(keyframes, totalDuration, ffmpeg.MinTransmuxedSegDuration)

	// Segment durations in ms
	segmentDurationsSeconds := []float32{}
	for _, d := range segmentDurations {
		segmentDurationsSeconds = append(segmentDurationsSeconds, float32(d.Seconds()))

	}

	templateData := map[string]interface{}{
		"videoBitRate":     probeData.Streams[0].BitRate,
		"videoWidth":       probeData.Streams[0].Width,
		"videoHeight":      probeData.Streams[0].Height,
		"videoCodecSpecs":  probeData.Streams[0].GetMime(),
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
	for _, stream := range streams {
		if stream.StreamType == "audio" {
			audioStreams = append(audioStreams, stream)
		} else if stream.StreamType == "video" {
			videoStreams = append(videoStreams, stream)
		}
	}
	// TODO(Leon Handreke): Have some smart heuristic here to match audio and video streams.
	streamCombinations := []VideoStreamCombination{
		{
			Video:       videoStreams[0],
			AudioCodecs: audioStreams[0].Codecs,
			AudioGroup:  audioStreams[0].RepresentationId,
		},
		{
			Video:       videoStreams[1],
			AudioCodecs: audioStreams[0].Codecs,
			AudioGroup:  audioStreams[0].RepresentationId,
		},
		{
			Video:       videoStreams[2],
			AudioCodecs: audioStreams[1].Codecs,
			AudioGroup:  audioStreams[1].RepresentationId,
		},
	}

	t.Execute(&buf, map[string]interface{}{
		"videoStreams":       videoStreams,
		"audioStreams":       audioStreams,
		"streamCombinations": streamCombinations,
	})
	return buf.String()
}

func BuildTranscodingMediaPlaylistFromFile(filePath string, stream ffmpeg.OfferedStream) string {
	segmentDurationsSeconds := []float64{}
	for _, d := range stream.GetSegmentDurations() {
		segmentDurationsSeconds = append(segmentDurationsSeconds, d.Seconds())
	}

	templateData := map[string]interface{}{
		"segmentDurations": segmentDurationsSeconds,
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingMediaPlaylistTemplate))
	t.Execute(&buf, templateData)
	return buf.String()
}
