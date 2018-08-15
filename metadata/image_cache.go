package metadata

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/helpers"
	"io"
	"io/ioutil"
	"net/http"

	"os"
	"os/user"
	"path"
)

type ImageManager struct {
	cachePath string
}

func NewImageManager() *ImageManager {
	// DRY this up (context.go)
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Failed to determine user's home directory: ", err.Error())
	}
	cachePath := path.Join(usr.HomeDir, ".config", "bss", "metadb", "cache", "images")
	helpers.EnsurePath(cachePath)
	return &ImageManager{cachePath: cachePath}
}

func (self *ImageManager) HttpHandler(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]
	size := mux.Vars(r)["size"]
	id := mux.Vars(r)["id"]
	fmt.Println(provider, size, id)
	folderPath := path.Join(self.cachePath, provider, size)
	filePath := path.Join(folderPath, id)
	if helpers.FileExists(filePath) {
		fmt.Println("We have cache")
		file, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Println("Could read cached file")
		} else {
			w.Write(file)
		}
	} else {
		fmt.Println("We don't have cache")

		helpers.EnsurePath(folderPath)
		openFile, err := os.Create(filePath)
		if err != nil {
			fmt.Println("Error while creating", filePath, ":", err)
			return
		}
		defer openFile.Close()

		url := fmt.Sprintf("http://image.tmdb.org/t/p/%s/%s", size, id)
		response, err := http.Get(url)
		if err != nil {
			fmt.Println("Error while downloading", url, ":", err)
			return
		}
		defer response.Body.Close()

		var b bytes.Buffer
		n, err := io.Copy(&b, response.Body)
		if err != nil {
			fmt.Println("Error while downloading", url, ":", err)
			return
		}
		// Write to a secondary variable so we can serve the image right away without rereading it from disk
		imageB := b.Bytes()
		fmt.Println("Wrote", n, "bytes")
		_, err = b.WriteTo(openFile)
		if err != nil {
			fmt.Println("Wrote file to disk")
		}

		w.Write(imageB)
	}
}
