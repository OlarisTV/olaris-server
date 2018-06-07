package main

import (
	"flag"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/metadata/resolvers"
	"log"
	"net/http"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	//defer db.Close()
	ctx := db.NewMDContext()
	defer ctx.Db.Close()
	libraryManager := metadata.NewLibraryManager(ctx)

	refresh := make(chan int)
	ctx.RefreshChan = refresh
	libraryManager.ActivateAll()

	http.Handle("/", http.HandlerFunc(resolvers.GraphiQLHandler))

	http.Handle("/auth", http.HandlerFunc(db.AuthHandler))

	http.Handle("/m/query", db.AuthMiddleWare(resolvers.NewRelayHandler(ctx)))

	go func() {
		for _ = range refresh {
			libraryManager.ActivateAll()
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
