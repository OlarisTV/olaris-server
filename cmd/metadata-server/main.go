package main

import (
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"net/http"
	"strings"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	mctx := db.NewMDContext()
	defer mctx.Db.Close()

	libraryManager := db.NewLibraryManager(watcher)
	// Scan on start-up
	go libraryManager.RefreshAll()

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				fmt.Println("event:", event)
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					fmt.Println("File removed, removing watcher")
					watcher.Remove(event.Name)
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Println("modified file:", event.Name)
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fmt.Println("Added file:", event.Name)
					watcher.Add(event.Name)
					fmt.Println("asking lib to scan")
					for _, lib := range db.AllLibraries() {
						if strings.Contains(event.Name, lib.FilePath) {
							fmt.Println("Scanning file for lib:", lib.Name)
							libraryManager.ProbeFile(&lib, event.Name)
							// We can probably only get the MD for the recently added file here
							libraryManager.UpdateMD(&lib)
						}
					}
				}
			case err := <-watcher.Errors:
				fmt.Println("error:", err)
			}
		}
	}()

	r := mux.NewRouter()
	r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler(mctx)))

	srv := &http.Server{Addr: ":8080", Handler: r}
	srv.ListenAndServe()
}
