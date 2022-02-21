package main

import (
	"os"

	"github.com/goava/di"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"gitlab.com/olaris/olaris-server/cmd"
	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/pkg/config"
	"gitlab.com/olaris/olaris-server/utils"
)

func main() {
	config.RegisterFlags(registerGlobalFlags)
	config.InitViper()

	container, err := di.New(
		cmd.New(),
	)
	if err != nil {
		logrus.Fatal(err)
	}

	var rootCommand root.RootCommand
	utils.MustResolve(container, &rootCommand)

	err = rootCommand.GetCobraCommand().Execute()
	if err != nil {
		logrus.Fatal(err)
	}

	os.Exit(0)
}

func registerGlobalFlags(fs *pflag.FlagSet) {
	fs.Bool("allow_direct_file_access", false, "Whether accessing files directly by path (without a valid JWT) is allowed")
	fs.Bool("enable_streaming_debug_pages", false, "Whether to enable debug pages in the streaming server")
	fs.Bool("write_transcoder_log", true, "Whether to write transcoder output to logfile")

	fs.StringVar(&config.ConfigDir, "config_dir", config.GetDefaultConfigDir(), "Default configuration directory for config files")
	fs.String("rclone_config", helpers.GetDefaultRcloneConfigPath(), "Default rclone configuration file")
	fs.String("cache_dir", helpers.GetDefaultCacheDir(), "Cache directory for transcoding an other temporarily files")

	viper.BindPFlag("server.cacheDir", fs.Lookup("cache_dir"))
	viper.BindPFlag("server.directFileAccess", fs.Lookup("allow_direct_file_access"))
	viper.BindPFlag("debug.streamingPages", fs.Lookup("enable_streaming_debug_pages"))
	viper.BindPFlag("debug.transcoderLog", fs.Lookup("write_transcoder_log"))
	viper.BindPFlag("rclone.configFile", fs.Lookup("rclone_config"))
}
