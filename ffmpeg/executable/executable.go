package executable

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/helpers"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

//go:generate mkdir -p build/$GOOS-$GOARCH
//go:generate go-bindata -pkg $GOPACKAGE -prefix "build/$GOOS-$GOARCH" build/$GOOS-$GOARCH/...

// To ensure that the executable isn't trying to be installed by multiple callers at the
// same time, leading to crashes.
var installExecutableMutex = &sync.Mutex{}

func getExecutablePath(name string) string {
	if viper.GetBool("server.systemFFmpeg") {
		return name
	}
	binaryDir := path.Join(viper.GetString("server.cacheDir"), "ffmpeg")
	binaryPath := path.Join(binaryDir, name)

	info, err := AssetInfo(name)
	if err != nil {
		log.Warnf("No compiled version of Olaris %s found. Falling back to the system's version. Please ensure you use https://gitlab.com/olaris/ffmpeg/ or this will fail.", name)
		return name
	}

	if stat, err := os.Stat(binaryPath); err == nil && stat.Size() == info.Size() {
		return binaryPath
	}

	installExecutableMutex.Lock()
	defer installExecutableMutex.Unlock()

	// Check again in the mutex, maybe somebody already installed it while we waited
	if stat, err := os.Stat(binaryPath); err == nil && stat.Size() == info.Size() {
		return binaryPath
	}

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

	return binaryPath
}

// GetFFmpegExecutablePath gets the path to either a compiled-in or system version of ffmpeg
func GetFFmpegExecutablePath() string {
	return getExecutablePath("ffmpeg")
}

// GetFFprobeExecutablePath gets the path to either a compiled-in or system version of ffprobe
func GetFFprobeExecutablePath() string {
	return getExecutablePath("ffprobe")
}
