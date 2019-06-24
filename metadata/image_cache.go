package metadata

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/olaris/olaris-server/helpers"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

// ImageManager cache implementation for themoviedb.
type ImageManager struct {
	// Path where cached images will be stored.
	cachePath string
}

// NewImageManager creates a new instance of a image caching server for themoviedb.
func NewImageManager() *ImageManager {
	cachePath := path.Join(helpers.CacheDir(), "images")
	helpers.EnsurePath(cachePath)
	return &ImageManager{cachePath: cachePath}
}

// HTTPHandler responsible for proxying calls to themoviedb. If an image is already present locally it will be served from filesystem. If it's not present it will attempt to download a file into the filesystem cache from tmdb.
func (man *ImageManager) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]
	size := mux.Vars(r)["size"]
	id := mux.Vars(r)["id"]
	folderPath := path.Join(man.cachePath, provider, size)
	filePath := path.Join(folderPath, id)
	if helpers.FileExists(filePath) {
		log.WithFields(log.Fields{"file": filePath}).Debugln("Requested file already in cache.")
		file, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Warnln("Could not read file from disk.")
		} else {
			w.Write(file)
		}
	} else {
		log.WithFields(log.Fields{"file": filePath}).Debugln("Requested file not in cache yet.")

		helpers.EnsurePath(folderPath)
		openFile, err := os.Create(filePath)
		if err != nil {
			log.Warnf("Error while creating file '%s': %s", filePath, err)
			return
		}
		defer openFile.Close()

		url := fmt.Sprintf("http://image.tmdb.org/t/p/%s/%s", size, id)
		response, err := http.Get(url)
		if err != nil {
			log.Warnf("Could not reach image url '%s': %s", url, err)
			return
		}
		defer response.Body.Close()

		var b bytes.Buffer
		_, err = io.Copy(&b, response.Body)
		if err != nil {
			log.Warnf("Error while downloading file from '%s': '%s'", url, err)
			return
		}
		// Write to a secondary variable so we can serve the image right away without rereading it from disk
		imageB := b.Bytes()
		_, err = b.WriteTo(openFile)
		if err != nil {
			log.Warnln("Error writing downloaded file to disk:", err)
		}

		w.Write(imageB)
	}
}
