package streaming

import (
	"github.com/gorilla/mux"
	"gitlab.com/olaris/olaris-server/filesystem"
	"net/http"
	"path"
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	node, err := getNode(mux.Vars(r)["fileLocator"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if node.BackendType() == filesystem.BackendLocal {
		http.ServeFile(w, r, path.Clean(node.Path()))
		return
	} else if node.BackendType() == filesystem.BackendRclone {
		serveRcloneFile(w, r, node)
	}

	http.NotFound(w, r)
}
