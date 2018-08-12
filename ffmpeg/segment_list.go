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

type SegmentList []Segment

func (l *SegmentList) Contains(ts DtsTimestamp) bool {
	return (*l)[0].StartTimestamp <= ts && ts <= (*l)[len(*l)-1].EndTimestamp
}

func (l *SegmentList) ContainsSegmentId(i int) bool {
	return (*l)[0].SegmentId <= i && i <= (*l)[len(*l)-1].SegmentId
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
