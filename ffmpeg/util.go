package ffmpeg

import (
	"fmt"
	"net/url"
	"time"
)

func reverseMap(m map[string]string) map[string]string {
	n := make(map[string]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}

// TODO(Leon Handreke): Get a proper list according to the standard
var langTagToHumanized = map[string]string{
	"eng": "English",
	"ger": "German",
	"jpn": "Japanese",
	"ita": "Italian",
	"fre": "French",
	"spa": "Spanish",
	"hun": "Hungarian",
	"unk": "Unknown",
}

var humanizedToLangTag = reverseMap(langTagToHumanized)

func GetTitleOrHumanizedLanguage(stream ProbeStream) string {
	title := stream.Tags["title"]
	if title != "" {
		return title
	}

	lang := GetLanguageTag(stream)

	humanizedLang := langTagToHumanized[lang]
	if humanizedLang != "" {
		return humanizedLang
	}
	if lang != "" {
		return lang
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
	// We just assume that the time_base is the same for all.
	timeBase := keyframeIntervals[0].TimeBase
	totalInterval := Interval{
		timeBase,
		keyframeIntervals[0].StartTimestamp,
		keyframeIntervals[len(keyframeIntervals)-1].EndTimestamp,
	}
	totalDuration := totalInterval.Duration()
	segmentDurationDts := int64(segmentDuration.Seconds() * float64(timeBase))
	numFullSegments := int(totalDuration / segmentDuration)

	session := SegmentList{}
	for i := 0; i < numFullSegments; i++ {
		session = append(session,
			Segment{
				Interval{
					timeBase,
					// Casting time.Duration to int is OK here because segmentDuration is small
					DtsTimestamp(int64(i) * segmentDurationDts),
					DtsTimestamp(int64(i+1) * segmentDurationDts),
				},
				int(i),
			})
	}
	lastEndTimestamp := session[len(session)-1].EndTimestamp
	session = append(session,
		Segment{Interval{
			timeBase,
			lastEndTimestamp,
			keyframeIntervals[len(keyframeIntervals)-1].EndTimestamp},
			numFullSegments})
	return []SegmentList{session}
}

func buildIntervals(startTimestamps []DtsTimestamp, totalDuration DtsTimestamp, timeBase int64) []Interval {
	intervals := []Interval{}

	if len(startTimestamps) == 0 {
		panic("startTimestamps must contain at least one element")
	}

	for i := 1; i < len(startTimestamps); i++ {
		intervals = append(intervals,
			Interval{timeBase, startTimestamps[i-1], startTimestamps[i]})
	}
	intervals = append(intervals,
		Interval{timeBase, startTimestamps[len(startTimestamps)-1], totalDuration})

	return intervals
}

func timestampToDuration(ts DtsTimestamp, timeBase int64) time.Duration {
	return time.Duration(float64(time.Second) * float64(ts) / float64(timeBase))
}

func mediaFileURLToFilepath(mediaFileURLStr string) (string, error) {
	mediaFileURL, _ := url.Parse(mediaFileURLStr)
	if mediaFileURL.Scheme == "file" {
		return mediaFileURL.Path, nil
	}
	return "", fmt.Errorf("%s is not a local file", mediaFileURLStr)
}
