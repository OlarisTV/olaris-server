package streaming

import (
	"net/http"
)

func serveRcloneFile(w http.ResponseWriter, r *http.Request, filepath string) {
	//if filepath[0] == '/' {
	//	filepath = filepath[1:]
	//}
	//parts := strings.SplitN(filepath, "/", 2)
	//
	//if len(parts) != 2 {
	//	http.Error(w,
	//		fmt.Sprintf("Failed to split rclone path \"%s\"", filepath),
	//		http.StatusBadRequest)
	//}
	//rcloneRemote := parts[0]
	//rclonePath := parts[1]
	//
	//filesystem, err := fs.NewFs(rcloneRemote + ":/")
	//if err != nil {
	//	http.Error(w, "Failed to create rclone Fs: "+err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//
	//vfilesystem := vfs.New(filesystem, &vfs.Options{ReadOnly: true, CacheMode: vfs.CacheModeFull})
	//defer vfilesystem.Shutdown()
	//
	//f, err := vfilesystem.OpenFile(rclonePath, 0, 0)
	//if err != nil {
	//	http.Error(w,
	//		fmt.Sprintf("Failed get file \"%s\" from rclone: %s", rclonePath, err.Error()),
	//		http.StatusInternalServerError)
	//	return
	//}
	//defer f.Close()
	//
	//http.ServeContent(w, r, path.Base(rclonePath), time.Now(), f)
	//
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	http.Error(w, "", http.StatusInternalServerError)
}
