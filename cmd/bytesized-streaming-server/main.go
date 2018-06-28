package main

import (
	"context"
	"flag"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/peak6/envflag"
	"gitlab.com/bytesized/bytesized-streaming/metadata"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"gitlab.com/bytesized/bytesized-streaming/streaming"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	flag.Parse()
	envflag.Parse()

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := mux.NewRouter()

	r.PathPrefix("/s").Handler(http.StripPrefix("/s", streaming.GetHandler()))
	defer streaming.Cleanup()

	mctx := db.NewDefaultMDContext()
	defer mctx.Db.Close()

	r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler(mctx)))

	handler := handlers.LoggingHandler(os.Stdout, r)

	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan

	mctx.ExitChan <- 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

}
