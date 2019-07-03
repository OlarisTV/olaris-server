package cmd

import (
	"context"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	// Backend for Rclone
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/react"
	"gitlab.com/olaris/olaris-server/streaming"
	"net/http"
	_ "net/http/pprof" // For Profiling
	"os"
	"os/signal"
	"time"
)

var port int
var dbLog bool
var verbose bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the olaris server",
	Run: func(cmd *cobra.Command, args []string) {

		mainRouter := mux.NewRouter()

		r := mainRouter.PathPrefix("/olaris")
		rr := mainRouter.PathPrefix("/olaris")
		rrr := mainRouter.PathPrefix("/olaris")

		mctx := app.NewMDContext(helpers.MetadataConfigPath(), dbLog, verbose)

		metaRouter := r.PathPrefix("/m").Subrouter()
		metadata.RegisterRoutes(mctx, metaRouter)

		streamingRouter := rr.PathPrefix("/s").Subrouter()
		streaming.RegisterRoutes(streamingRouter)
		defer streaming.Cleanup()

		// This is just to make sure that no temp files stay behind in case the
		// garbage collection below didn't work properly for some reason.
		// This is also relevant during development because the realize auto-reload
		// tool doesn't properly send SIGTERM.
		ffmpeg.CleanTranscodingCache()
		// TODO(Leon Handreke): Find a better way to do this, maybe a global flag?
		streaming.FfmpegUrlPort = port
		ffmpeg.FfmpegUrlPort = port

		appRoute := rrr.PathPrefix("/app").
			Handler(http.StripPrefix("/olaris/app", react.GetHandler())).
			Name("app")

		appURL, _ := appRoute.URL()
		mainRouter.Path("/").Handler(http.RedirectHandler(appURL.Path, http.StatusMovedPermanently))
		mainRouter.Path("/olaris").Handler(http.RedirectHandler(appURL.Path, http.StatusMovedPermanently))

		handler := cors.AllowAll().Handler(mainRouter)
		handler = handlers.LoggingHandler(os.Stdout, handler)

		log.Infoln("binding on port", port)
		srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: handler}
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				log.WithFields(log.Fields{"error": err}).Fatal("Error starting server.")
			}
		}()

		stopChan := make(chan os.Signal)
		signal.Notify(stopChan, os.Interrupt)
		signal.Notify(stopChan, os.Kill)

		// Wait for termination signal
		<-stopChan
		log.Println("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mctx.Cleanup()
		streaming.Cleanup()
		srv.Shutdown(ctx)
		log.Println("Shut down complete, exiting.")
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "http port")
	serveCmd.Flags().BoolVarP(&verbose, "verbose", "v", true, "verbose logging")
	serveCmd.Flags().BoolVar(&dbLog, "db-log", false, "sets whether the database should log queries")
	rootCmd.AddCommand(serveCmd)
}
