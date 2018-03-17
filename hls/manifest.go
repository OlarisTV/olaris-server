package hls

import (
	"bytes"
	"gitlab.com/bytesized/bytesized-streaming/ffmpeg"
	"log"
	"text/template"
	"time"
)

const transmuxingManifestTemplate = `
#EXTM3U
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-TARGETDURATION:5
#EXT-X-VERSION:7
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-MAP:URI="direct-stream-video/init.mp4"
{{ range $index, $duration := .segmentDurations }}
#EXTINF:{{ $duration }},
direct-stream-video/{{ $index }}.m4s{{ end }}
#EXT-X-ENDLIST
`

func BuildTransmuxingManifestFromFile(filePath string) string {
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
	t := template.Must(template.New("manifest").Parse(transmuxingManifestTemplate))
	t.Execute(&buf, templateData)
	return buf.String()
}
