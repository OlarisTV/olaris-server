package managers

import (
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

// FfmpegStreamFromDatabaseStream creates a ffmpeg stream object based on a database object
func FfmpegStreamFromDatabaseStream(s db.Stream) ffmpeg.Stream {
	return ffmpeg.Stream{
		StreamKey: ffmpeg.StreamKey{
			FileLocator: s.StreamKey.FileLocator,
			StreamId:    s.StreamKey.StreamId,
		},
		TotalDuration:    s.TotalDuration,
		TimeBase:         s.TimeBase,
		TotalDurationDts: ffmpeg.DtsTimestamp(s.TotalDurationDts),
		Codecs:           s.Codecs,
		CodecName:        s.CodecName,
		Profile:          s.Profile,
		BitRate:          s.BitRate,
		FrameRate:        s.FrameRate,
		Width:            s.Width,
		Height:           s.Height,
		StreamType:       s.StreamType,
		Language:         s.Language,
		Title:            s.Title,
		EnabledByDefault: s.EnabledByDefault,
	}
}

// DatabaseStreamFromFfmpegStream does the reverse of the above.
func DatabaseStreamFromFfmpegStream(s ffmpeg.Stream) db.Stream {
	return db.Stream{
		StreamKey: db.StreamKey{
			FileLocator: s.StreamKey.FileLocator,
			StreamId:    s.StreamKey.StreamId,
		},
		TotalDuration:    s.TotalDuration,
		TimeBase:         s.TimeBase,
		TotalDurationDts: int64(s.TotalDurationDts),
		Codecs:           s.Codecs,
		CodecName:        s.CodecName,
		Profile:          s.Profile,
		BitRate:          s.BitRate,
		FrameRate:        s.FrameRate,
		Width:            s.Width,
		Height:           s.Height,
		StreamType:       s.StreamType,
		Language:         s.Language,
		Title:            s.Title,
		EnabledByDefault: s.EnabledByDefault,
	}
}
