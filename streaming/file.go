package streaming

import (
	"gitlab.com/olaris/olaris-server/filesystem"
	"net/http"
	"path"
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	fileLocator, statusErr := getFileLocatorOrFail(r)
	if statusErr != nil {
		http.Error(w, statusErr.Error(), statusErr.Status())
		return
	}

	node, err := filesystem.GetNodeFromFileLocator(fileLocator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if node.BackendType() == filesystem.BackendLocal {
		http.ServeFile(w, r, path.Clean(node.Path()))
		return
	} else if node.BackendType() == filesystem.BackendRclone {
		serveRcloneFile(w, r, node)
		return
	}

	http.NotFound(w, r)
}
