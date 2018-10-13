package ffmpeg

import "time"

type DtsTimestamp int64

// A time interval [StartTimestamp, EndTimestamp) in DTS
type Interval struct {
	TimeBase       int64
	StartTimestamp DtsTimestamp
	EndTimestamp   DtsTimestamp
}

type Segment struct {
	Interval
	SegmentId int
}

func (i *Interval) Duration() time.Duration {
	return i.EndDuration() - i.StartDuration()
}

func (i *Interval) StartDuration() time.Duration {
	return time.Duration(float64(time.Second) * float64(i.StartTimestamp) / float64(i.TimeBase))
}

func (i *Interval) EndDuration() time.Duration {
	return time.Duration(float64(time.Second) * float64(i.EndTimestamp) / float64(i.TimeBase))
}
