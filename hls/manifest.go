package hls

import (
	"bytes"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"log"
	"strings"
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
#EXT-X-MAP:URI="direct-stream-video/init.mp4"
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
direct-stream-video/{{ $index }}.m4s{{ end }}
#EXT-X-ENDLIST
`

const transcodingMasterPlaylistTemplate = `#EXTM3U
#EXT-X-VERSION:7
#EXT-X-INDEPENDENT-SEGMENTS

#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=1000000,CODECS="avc1.64001e,mp4a.40.2",AUDIO="64k"
480-1000k-video/media.m3u8
#EXT-X-STREAM-INF:PROGRAM-ID=1,BANDWIDTH=5000000,CODECS="avc1.64001f,mp4a.40.2",AUDIO="128k"
720-5000k-video/media.m3u8

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="64k",NAME="English",CHANNELS="2",URI="64k-audio/media.m3u8"
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="128k",NAME="English",CHANNELS="2",URI="128k-audio/media.m3u8"
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
	segmentDurations := ffmpeg.GuessSegmentDurations(keyframes, totalDuration, ffmpeg.MinSegDuration)

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

func BuildTranscodingMasterPlaylistFromFile(filePath string) string {
	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingMasterPlaylistTemplate))
	t.Execute(&buf, map[string]interface{}{})
	return buf.String()
}

func BuildTranscodingMediaPlaylistFromFile(filePath string, representationId string) string {
	probeData, err := ffmpeg.Probe(filePath)
	if err != nil {
		log.Fatal("Failed to ffprobe", filePath)
	}

	var segmentDuration float32 // in ms
	if strings.Contains(representationId, "audio") {
		segmentDuration = 4.992
	} else {
		segmentDuration = 5.000
	}

	totalDuration := probeData.Format.Duration()
	numFullSegments := int(totalDuration / (time.Duration(segmentDuration) * time.Second))

	segmentDurationsSeconds := []float32{}
	// We want one more segment to cover the end. For the moment we don't
	// care that it's a bit longer in the manifest, the client will play till EOF
	for i := 0; i < numFullSegments+1; i++ {
		segmentDurationsSeconds = append(segmentDurationsSeconds, segmentDuration)
	}

	templateData := map[string]interface{}{
		"segmentDurations": segmentDurationsSeconds,
	}

	buf := bytes.Buffer{}
	t := template.Must(template.New("manifest").Parse(transcodingMediaPlaylistTemplate))
	t.Execute(&buf, templateData)
	return buf.String()
}
