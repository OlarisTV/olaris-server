package cmd

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // For Profiling
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/react"
	"gitlab.com/olaris/olaris-server/streaming"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the olaris server",
	Run: func(cmd *cobra.Command, args []string) {
		mainRouter := mux.NewRouter()

		r := mainRouter.PathPrefix("/olaris")
		rr := mainRouter.PathPrefix("/olaris")
		rrr := mainRouter.PathPrefix("/olaris")

		dbOptions := db.DatabaseOptions{
			Connection: viper.GetString("database.connection"),
			LogMode:    viper.GetBool("server.DBLog"),
		}

		mctx := app.NewMDContext(dbOptions, agents.NewTmdbAgent())
		if viper.GetBool("server.verbose") {
			log.SetLevel(log.DebugLevel)
		}
		viper.WatchConfig()
		updateConfig := func(in fsnotify.Event) {
			log.Infoln("configuration file change detected")
			if viper.GetBool("server.verbose") {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.InfoLevel)
			}
			mctx.Db.LogMode(viper.GetBool("server.DBLog"))
		}
		viper.OnConfigChange(updateConfig)

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
		port := viper.GetInt("server.port")

		if viper.GetBool("server.zeroconf.enabled") {
			viper.SetDefault("server.zeroconf.domain", "local.")
			domain := viper.GetString("server.zeroconf.domain")
			zeroconfService, err := zeroconf.Register("olaris", "_http._tcp", domain, port, []string{"txtv=0", "lo=1", "la=2"}, nil)
			if err != nil {
				log.WithError(err).Warn("zeroconf setup failed")
			} else {
				log.Info("zeroconf successfully enabled")
			}
			defer zeroconfService.Shutdown()
		}

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
