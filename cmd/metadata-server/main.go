package main

import (
	"flag"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"log"
	"net/http"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	//defer db.Close()
	ctx := metadata.NewMDContext()
	defer ctx.Db.Close()
	libraryManager := metadata.NewLibraryManager(ctx)

	/*
		var count int
		ctx.Db.Table("libraries").Count(&count)
		if count == 0 {
			libraryManager.AddLibrary("Movies", *mediaFilesDir)
		}*/

	refresh := make(chan int)
	ctx.RefreshChan = refresh
	libraryManager.ActivateAll()

	http.Handle("/", http.HandlerFunc(metadata.GraphiQLHandler))

	http.Handle("/query", metadata.NewRelayHandler(ctx))

	go func() {
		for _ = range refresh {
			libraryManager.ActivateAll()
		}
	}()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
