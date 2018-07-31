package ffmpeg

import (
	"fmt"
	"time"
)

func GetTitleOrHumanizedLanguage(stream ProbeStream) string {
	title := stream.Tags["title"]
	if title != "" {
		return title
	}

	lang := GetLanguageTag(stream)
	// TODO(Leon Handreke): Get a proper list according to the standard
	humanizedLang := map[string]string{
		"eng": "English",
		"ger": "German",
		"jpn": "Japanese",
		"ita": "Italian",
		"fre": "French",
		"spa": "Spanish",
		"hun": "Hungarian",
		"unk": "Unknown",
	}[lang]

	if humanizedLang != "" {
		return humanizedLang
	}

	return fmt.Sprintf("stream-%d", stream.Index)

}

func GetLanguageTag(stream ProbeStream) string {
	lang := stream.Tags["language"]
	if lang != "" {
		return lang
	}
	return "unk"
}

func BuildConstantSegmentDurations(start time.Duration, segmentDuration time.Duration, totalDuration time.Duration) []time.Duration {
	numFullSegments := int64(totalDuration / segmentDuration)
	segmentStartTimestamps := []time.Duration{start}
	for i := int64(0); i < numFullSegments; i++ {
		segmentStartTimestamps = append(segmentStartTimestamps,
			segmentStartTimestamps[i]+segmentDuration)
	}
	return segmentStartTimestamps
}
