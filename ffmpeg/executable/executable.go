package executable

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"io/ioutil"
	"os"
	"path"
)

//go:generate mkdir -p build/$GOOS-$GOARCH
//go:generate go-bindata -pkg $GOPACKAGE -prefix "build/$GOOS-$GOARCH" build/$GOOS-$GOARCH/...

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

	info, err := AssetInfo(name)
	if err != nil {
		log.Warnf("No %s compiled in, using system system version instead", name)
		return name
	}
	if stat, err := os.Stat(binaryPath); err != nil || stat.Size() != info.Size() {
		log.Infof("Installing built-in binary for %s to %s", name, binaryPath)

		data, _ := Asset(name)
		helpers.EnsurePath(binaryDir)

		if err := ioutil.WriteFile(binaryPath, data, 0700); err != nil {
			log.Warnf(
				"Failed to write %s built-in binary, using system version instead: %s",
				name, err.Error())
			return name
		}
		os.Chmod(binaryPath, 0700)
	}
	return binaryPath
}

func GetFFmpegExecutablePath() string {
	return getExecutablePath("ffmpeg")
}

func GetFFprobeExecutablePath() string {
	return getExecutablePath("ffprobe")
}
