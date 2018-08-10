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

func BuildConstantSegmentDurations(keyframeIntervals []Interval, segmentDuration time.Duration) []SegmentList {
	totalDuration := keyframeIntervals[len(keyframeIntervals)-1].EndTimestamp
	numFullSegments := int(totalDuration / segmentDuration)

	session := SegmentList{}
	for i := 0; i < int(numFullSegments); i++ {
		session = append(session,
			Segment{
				Interval{
					// Casting time.Duration to int is OK here because segmentDuration is small
					time.Duration(i * int(segmentDuration)),
					time.Duration((i + 1) * int(segmentDuration))},
				i,
			})
	}
	session = append(session,
		Segment{Interval{
			session[len(session)-1].EndTimestamp,
			totalDuration}, numFullSegments})
	return []SegmentList{session}
}

func buildIntervals(startTimestamps []time.Duration, totalDuration time.Duration) []Interval {
	intervals := []Interval{}

	if len(startTimestamps) == 0 {
		panic("startTimestamps must contain at least one element")
	}

	for i := 1; i < len(startTimestamps); i++ {
		intervals = append(intervals,
			Interval{startTimestamps[i-1], startTimestamps[i]})
	}
	intervals = append(intervals,
		Interval{startTimestamps[len(startTimestamps)-1], totalDuration})

	return intervals
}
