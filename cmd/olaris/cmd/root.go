package cmd

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/olaris/olaris-server/helpers"
)

var rootCmd = &cobra.Command{
	Use: "olaris",
	Run: func(cmd *cobra.Command, args []string) {
		// Let root without arguments be an alias for serve
		serveCmd.Run(cmd, args)
	},
}

// Execute is the main launcher for the root command; this is
// where we bind some useful flags
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("allow_direct_file_access", false,
		"Whether accessing files directly by path (without a valid JWT) is allowed")
	viper.BindPFlag("server.directFileAccess", rootCmd.PersistentFlags().Lookup("allow_direct_file_access"))

	rootCmd.PersistentFlags().Bool("use_system_ffmpeg", false,
		"Whether to use system FFmpeg instead of binary builtin")
	viper.BindPFlag("server.systemFFmpeg", rootCmd.PersistentFlags().Lookup("use_system_ffmpeg"))

	rootCmd.PersistentFlags().Bool("enable_streaming_debug_pages", false,
		"Whether to enable debug pages in the streaming server")
	viper.BindPFlag("debug.streamingPages", rootCmd.PersistentFlags().Lookup("enable_streaming_debug_pages"))

	rootCmd.PersistentFlags().Bool("write_transcoder_log", true,
		"Whether to write transcoder output to logfile")
	viper.BindPFlag("debug.transcoderLog", rootCmd.PersistentFlags().Lookup("write_transcoder_log"))

	serveCmd.Flags().IntP("port", "p", 8080, "http port")
	viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))

	serveCmd.Flags().BoolP("verbose", "v", true, "verbose logging")
	viper.BindPFlag("server.verbose", serveCmd.Flags().Lookup("verbose"))

	serveCmd.Flags().Bool("db-log", false, "sets whether the database should log queries")
	viper.BindPFlag("server.DBLog", serveCmd.Flags().Lookup("db-log"))

	rootCmd.AddCommand(serveCmd)

	defaultConfigDir := path.Join(helpers.GetHome(), ".config", "olaris")
	if configDirEnv := os.Getenv("OLARIS_CONFIG_DIR"); configDirEnv != "" {
		defaultConfigDir = configDirEnv
	}
	var configDir string
	rootCmd.PersistentFlags().StringVar(&configDir, "config_dir", defaultConfigDir,
		"Default configuration directory for config files.")

	cobra.OnInitialize(func() {
		viper.AddConfigPath(configDir)
		viper.Set("configdir", configDir)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// the user has no config file
			}
		}

		config := &Config{}
		if err := viper.Unmarshal(config); err != nil {
			log.Debugf("error applying configuration: %s\n", err.Error())
		}
	})
}
