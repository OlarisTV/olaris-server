package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"gitlab.com/bytesized/bytesized-streaming/app"
	"gitlab.com/bytesized/bytesized-streaming/metadata/db"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	mctx := db.NewDefaultMDContext()
	defer mctx.Db.Close()

	r := mux.NewRouter()

	r.PathPrefix("/app").Handler(http.StripPrefix("/app", app.GetHandler(mctx)))

	srv := &http.Server{Addr: ":8080", Handler: r}
	go srv.ListenAndServe()

	// Wait for termination signal
	<-stopChan
	fmt.Println("Stopping services and cleaning up")

	mctx.ExitChan <- 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
