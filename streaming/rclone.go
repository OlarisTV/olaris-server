package streaming

import (
	"github.com/gorilla/mux"
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/vfs"
	"net/http"
	"path"
	"time"
)

func serveRcloneFile(w http.ResponseWriter, r *http.Request) {
	rcloneRemote := mux.Vars(r)["rcloneRemote"]
	rclonePath := mux.Vars(r)["rclonePath"]

	filesystem, err := fs.NewFs(rcloneRemote + ":/")
	if err != nil {
		http.Error(w, "Failed to create rclone Fs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	vfilesystem := vfs.New(filesystem, &vfs.Options{ReadOnly: true, CacheMode: vfs.CacheModeFull})
	defer vfilesystem.Shutdown()

	f, err := vfilesystem.OpenFile(rclonePath, 0, 0)
	defer f.Close()
	if err != nil {
		http.Error(w, "Failed get file from rclone: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, path.Base(rclonePath), time.Now(), f)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
