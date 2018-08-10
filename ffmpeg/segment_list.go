package ffmpeg

import "time"

// A time interval [StartTimestamp, EndTimestamp)
type Interval struct {
	StartTimestamp time.Duration
	EndTimestamp   time.Duration
}

type Segment struct {
	Interval
	SegmentId int
}

type SegmentList []Segment

func (l *SegmentList) Contains(ts time.Duration) bool {
	return (*l)[0].StartTimestamp <= ts && ts <= (*l)[len(*l)-1].EndTimestamp
}

func (l *SegmentList) ContainsSegmentId(i int) bool {
	return (*l)[0].SegmentId <= i && i <= (*l)[len(*l)-1].SegmentId
}
