package serve

import (
	"context"
	"fmt"
	"gitlab.com/olaris/olaris-server/helpers"
	"net/http"
	_ "net/http/pprof" // For Profiling
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/goava/di"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/grandcat/zeroconf"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/metadata"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
	"gitlab.com/olaris/olaris-server/react"
	"gitlab.com/olaris/olaris-server/streaming"
)

type ServeCommand cmd.Command

func New() di.Option {
	return di.Options(
		di.Provide(NewServeCommand, di.As(new(ServeCommand))),
		di.Invoke(RegisterServeCommand),
	)
}

func RegisterServeCommand(rootCommand root.RootCommand, serveCommand ServeCommand) {
	rootCommand.GetCobraCommand().AddCommand(serveCommand.GetCobraCommand())

	rootCommand.GetCobraCommand().Flags().AddFlagSet(serveCommand.GetCobraCommand().Flags())
	rootCommand.GetCobraCommand().Run = serveCommand.GetCobraCommand().Run
}

func NewServeCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use:   "serve",
		Short: "Start the olaris server",
		Run: func(cmd *cobra.Command, args []string) {
			if viper.GetBool("server.verbose") {
				log.SetLevel(log.DebugLevel)
			}
			viper.WatchConfig()

			// Check FFmpeg version and warn if it's missing
			ffmpegVersion, err := ffmpeg.GetFfmpegVersion()
			if err != nil {
				if parseErr, ok := err.(*ffmpeg.VersionParseError); ok {
					log.WithError(parseErr).Warn("unable to determine installed FFmpeg version")
				} else {
					log.WithError(err).Warn("FFmpeg not found. STREAMING WILL NOT WORK IF FFMPEG IS NOT INSTALLED AND IN YOUR PATH!")
				}
			} else {
				log.WithField("version", ffmpegVersion.ToString()).Debugf("FFmpeg found")
			}

			// Check FFprobe version and warn if it's missing
			ffprobeVersion, err := ffmpeg.GetFfprobeVersion()
			if err != nil {
				if parseErr, ok := err.(*ffmpeg.VersionParseError); ok {
					log.WithError(parseErr).Warn("unable to determine installed FFprobe version")
				} else {
					log.WithError(err).Warn("FFprobe not found. STREAMING WILL NOT WORK IF FFPROBE IS NOT INSTALLED AND IN YOUR PATH!")
				}
			} else {
				log.WithField("version", ffprobeVersion.ToString()).Debugf("FFprobe found")
			}

			mainRouter := mux.NewRouter()

			r := mainRouter.PathPrefix("/olaris")
			rr := mainRouter.PathPrefix("/olaris")
			rrr := mainRouter.PathPrefix("/olaris")

			dbOptions := db.DatabaseOptions{
				Connection: viper.GetString("database.connection"),
				LogMode:    viper.GetBool("server.DBLog"),
			}

			mctx := app.NewMDContext(dbOptions, agents.NewTmdbAgent())

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
					log.WithFields(log.Fields{"error": err}).Fatal("error starting server")
				}
			}()

			stopChan := make(chan os.Signal, 2)
			signal.Notify(stopChan, os.Interrupt, os.Kill)

			// Wait for termination signal
			<-stopChan
			log.Println("shutting down...")

			// Clean up the metadata context
			mctx.Cleanup()

			// Clean up playback/transcode sessions
			sessionCleanupContext, sessionCleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer sessionCleanupCancel()
			streaming.PBSManager.DestroyAll(sessionCleanupContext)

			// Shut down the HTTP server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(ctx)

			log.Println("shut down complete, exiting.")
		},
	}

	c.Flags().IntP("port", "p", 8080, "http port")
	c.Flags().BoolP("verbose", "v", true, "verbose logging")
	c.Flags().Bool("db-log", false, "sets whether the database should log queries")
	c.Flags().String("db-conn", "", "sets the database connection string")
	c.Flags().String("sqlite_dir", path.Join(helpers.BaseConfigDir(), "metadb"), "Path where the SQLite database should be stored")
	c.Flags().Bool("scan-hidden", false, "sets whether to scan hidden directories (directories starting with a .)")

	viper.BindPFlag("server.port", c.Flags().Lookup("port"))
	viper.BindPFlag("server.verbose", c.Flags().Lookup("verbose"))
	viper.BindPFlag("server.DBLog", c.Flags().Lookup("db-log"))
	viper.BindPFlag("server.sqliteDir", c.Flags().Lookup("sqlite_dir"))
	viper.BindPFlag("database.connection", c.Flags().Lookup("db-conn"))
	viper.BindPFlag("metadata.scan_hidden", c.Flags().Lookup("scan-hidden"))

	return &cmd.CobraCommand{Command: c}
}
