package ffmpeg

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"gitlab.com/olaris/olaris-server/helpers"
	"io"
	"net/url"
	"os"
	"path"
	"time"
)

var writeTranscoderLog = flag.Bool(
	"write_transcoder_log",
	false,
	"Whether to write transcoder output to logfile")

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
	"dut": "Dutch",
	"por": "Portugese",
	"pol": "Polish",
	"rus": "Russian",
	"vie": "Vietnamese",
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

func BuildConstantSegmentDurations(interval Interval, segmentDuration time.Duration, startSegmentIndex int) SegmentList {
	// We just assume that the time_base is the same for all.
	timeBase := interval.TimeBase
	totalDuration := interval.Duration()
	segmentDurationDts := int64(segmentDuration.Seconds() * float64(timeBase))
	numFullSegments := int(totalDuration / segmentDuration)

	session := SegmentList{}
	for i := 0; i < numFullSegments; i++ {
		session = append(session,
			Segment{
				Interval{
					timeBase,
					// Casting time.Duration to int is OK here because segmentDuration is small
					interval.StartTimestamp + DtsTimestamp(int64(i)*segmentDurationDts),
					interval.StartTimestamp + DtsTimestamp(int64(i+1)*segmentDurationDts),
				},
				startSegmentIndex + int(i),
			})
	}
	if numFullSegments == 0 {
		session = append(session,
			Segment{Interval{
				timeBase,
				interval.StartTimestamp,
				interval.EndTimestamp},
				startSegmentIndex + numFullSegments})
	} else {
		// NOTE(Leon Handreke): Longer last segment
		//session[len(session)-1].EndTimestamp = interval.EndTimestamp
		// NOTE(Leon Handreke): Shorter last segment
		session = append(session, Segment{
			Interval{
				timeBase,
				session[len(session)-1].EndTimestamp,
				interval.EndTimestamp,
			},
			startSegmentIndex + numFullSegments,
		})
	}
	return session
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

func getTranscodingLogSink(prefix string) io.WriteCloser {
	if !(*writeTranscoderLog) {
		f, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0600)
		return f
	}

	filename := fmt.Sprintf("%s_%s.log",
		prefix, time.Now().UTC().Format(time.RFC3339))
	filepath := path.Join(helpers.LogPath(), filename)
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		glog.Error("Failed to open log file ", filepath, ": ", err.Error())
		f, _ = os.OpenFile(os.DevNull, os.O_RDWR, 0600)
	}
	fmt.Println(filepath)
	return f
}
