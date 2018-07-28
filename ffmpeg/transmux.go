package ffmpeg

import (
	"fmt"
	"gitlab.com/bytesized/bytesized-streaming/streaming/db"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"
)

// NewTransmuxingSession starts a new transmuxing-only (aka "Direct Stream") session.
func NewTransmuxingSession(streamRepresentation StreamRepresentation, outputDirBase string, segmentOffset int) (*TranscodingSession, error) {

	outputDir, err := ioutil.TempDir(outputDirBase, "transcoding-session-")
	if err != nil {
		return nil, err
	}

	var numSegments int
	var startTimestamp, endTimestamp time.Duration

	if streamRepresentation.Stream.StreamType == "video" {
		startTimestamp = streamRepresentation.SegmentStartTimestamps[segmentOffset]
		if segmentOffset+segmentsPerSession >= len(streamRepresentation.SegmentStartTimestamps) {
			endTimestamp = streamRepresentation.Stream.TotalDuration
		} else {
			endTimestamp = streamRepresentation.SegmentStartTimestamps[segmentOffset+segmentsPerSession]
		}
		numSegments = segmentsPerSession
	} else { // audio
		startTimestamp = streamRepresentation.SegmentStartTimestamps[0]
		endTimestamp = streamRepresentation.Stream.TotalDuration
		numSegments = len(streamRepresentation.SegmentStartTimestamps)
	}

	cmd := exec.Command("ffmpeg",
		// -ss being before -i is important for fast seeking
		"-ss", fmt.Sprintf("%.3f", startTimestamp.Seconds()),
		"-i", streamRepresentation.Stream.MediaFileURL,
		"-copyts",
		"-to", fmt.Sprintf("%.3f", endTimestamp.Seconds()),
		"-map", fmt.Sprintf("0:%d", streamRepresentation.Stream.StreamId),
		"-c:0", "copy",
		"-threads", "2",
		"-f", "hls",
		"-start_number", fmt.Sprintf("%d", segmentOffset),
		"-hls_time", fmt.Sprintf("%.3f", MinTransmuxedSegDuration.Seconds()),
		"-hls_segment_type", "1", // fMP4
		"-hls_segment_filename", "stream0_%d.m4s",
		// We serve our own manifest, so we don't really care about this.
		path.Join(outputDir, "generated_by_ffmpeg.m3u"))
	log.Println("ffmpeg started with", cmd.Args)
	cmd.Stderr, _ = os.Open(os.DevNull)
	cmd.Stdout = os.Stdout
	cmd.Dir = outputDir

	return &TranscodingSession{
		cmd:            cmd,
		Stream:         streamRepresentation,
		outputDir:      outputDir,
		firstSegmentId: segmentOffset,
		numSegments:    numSegments,
	}, nil
}

func GetTransmuxedRepresentation(stream Stream) (StreamRepresentation, error) {
	representation := StreamRepresentation{
		Stream: stream,
		Representation: Representation{
			RepresentationId: "direct",
			Container:        "video/mp4",
			Codecs:           stream.Codecs,
			BitRate:          int(stream.BitRate),
			transmuxed:       true,
		},
	}

	if stream.StreamType == "audio" {
		representation.SegmentStartTimestamps = BuildConstantSegmentDurations(
			MinTransmuxedSegDuration, stream.TotalDuration)
	} else if stream.StreamType == "video" {
		// TODO(Leon Handreke): In the DB we sometimes use the absolute path,
		// sometimes just a name. We need some other good descriptor for files,
		// preferably including a checksum
		keyframeCache, err := db.GetSharedDB().GetKeyframeCache(stream.MediaFileURL)
		if err != nil {
			return StreamRepresentation{}, err
		}

		keyframeTimestamps := []time.Duration{}

		if keyframeCache != nil {
			//glog.Infof("Reading keyframes for %s from cache", stream.MediaFileURL)
			for _, v := range keyframeCache.KeyframeTimestamps {
				keyframeTimestamps = append(keyframeTimestamps, time.Duration(v))
			}
		} else {
			keyframeTimestamps, err = ProbeKeyframes(stream.MediaFileURL)
			if err != nil {
				return StreamRepresentation{}, err
			}

			keyframeCache := db.KeyframeCache{Filename: stream.MediaFileURL}
			for _, v := range keyframeTimestamps {
				keyframeCache.KeyframeTimestamps = append(keyframeCache.KeyframeTimestamps, int64(v))
			}
			db.GetSharedDB().InsertOrUpdateKeyframeCache(keyframeCache)
		}
		representation.SegmentStartTimestamps = guessTransmuxedSegmentStartTimestamps(keyframeTimestamps)
	}

	return representation, nil
}

func guessTransmuxedSegmentStartTimestamps(keyframeTimestamps []time.Duration) []time.Duration {
	segmentTimestamps := []time.Duration{
		// First keyframe should equal first frame, but who knows, video is weird...
		keyframeTimestamps[0],
	}
	for _, keyframe := range keyframeTimestamps {
		d := keyframe - segmentTimestamps[len(segmentTimestamps)-1]
		if d > MinTransmuxedSegDuration {
			segmentTimestamps = append(segmentTimestamps, keyframe)
		}
	}

	return segmentTimestamps
}
