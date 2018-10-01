package executable

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"io/ioutil"
	"os"
	"path"
)

//go:generate go-bindata -pkg $GOPACKAGE -prefix "build/" build/...

var useSystemFFmpeg = flag.Bool(
	"use_system_ffmpeg",
	false,
	"Whether to use system FFmpeg instead of binary builtin")

func getExecutablePath(name string) string {
	if *useSystemFFmpeg {
		return name
	}
	binaryDir := path.Join(helpers.CacheDir(), "ffmpeg")
	binaryPath := path.Join(binaryDir, name)
	// TODO(Leon Handreke): Add ability to update the ffmpeg binary, maybe by size?
	if _, err := os.Stat(binaryPath); err != nil {
		data, err := Asset(name)
		if err != nil {
			log.Warnf("No %s compiled in, using system system version instead", name)
			return name
		}
		helpers.EnsurePath(binaryDir)
		if err := ioutil.WriteFile(binaryPath, data, 0700); err != nil {
			fmt.Println(err.Error())
			log.Warnf("Failed to write %s built-in binary, using system version instead", name)
			return name
		}
	}
	return binaryPath
}

func GetFFmpegExecutablePath() string {
	return getExecutablePath("ffmpeg")
}

func GetFFprobeExecutablePath() string {
	return getExecutablePath("ffprobe")
}
