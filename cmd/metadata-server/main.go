package main

import (
	"context"
	"flag"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/bytesized/bytesized-streaming/app"
	"gitlab.com/bytesized/bytesized-streaming/helpers"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var mediaFilesDir = flag.String("media_files_dir", "/var/media", "Path used if no libraries exist for the default library")

func main() {
	flag.Parse()

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	mctx := db.NewDefaultMDContext()
	defer mctx.Db.Close()

	r := mux.NewRouter()
	r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler(mctx)))
	r.PathPrefix("/app").Handler(http.StripPrefix("/app", app.GetHandler(mctx)))

	srv := &http.Server{Addr: ":8080", Handler: r}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan
	log.Println("Stopping services and cleaning up.")

	mctx.ExitChan <- 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
