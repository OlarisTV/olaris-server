package streaming

import (
	"github.com/gorilla/mux"
	"net/http"
	"path"
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	fileLocator, err := getFileLocator(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if fileLocator.Location == "local" {
		http.ServeFile(w, r, path.Clean(fileLocator.Path))
		return
	}

	if fileLocator.Location == "rclone" {
		serveRcloneFile(w, r, fileLocator.Path)
	}

	http.NotFound(w, r)
}
