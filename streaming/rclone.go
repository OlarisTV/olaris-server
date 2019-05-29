package streaming

import (
	"fmt"
	_ "github.com/ncw/rclone/backend/drive"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/vfs"
	"gitlab.com/olaris/olaris-server/filesystem"
	"net/http"
	"path"
	"time"
)

func serveRcloneFile(w http.ResponseWriter, r *http.Request, node filesystem.Node) {
	rcloneNode := node.(*filesystem.RcloneNode)

	f, err := rcloneNode.Node.(*vfs.File).Open(0)
	if err != nil {
		http.Error(w,
			fmt.Sprintf("Failed get file \"%s\" from rclone: %s", node.Path(), err.Error()),
			http.StatusInternalServerError)
		return
	}
	defer f.Close()

	http.ServeContent(w, r, path.Base(node.Path()), time.Now(), f)
}
