package main

import (
	"os"
	"strings"

	"github.com/spf13/viper"
	"github.com/spf13/pflag"
	"gitlab.com/olaris/olaris-server/cmd/olaris/cmd"
)

func main() {
	viper.SetConfigName("olaris")

	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("olaris")
	viper.AutomaticEnv()

	// hack to get rid of the unwanted extra pflags from rclone/fs/log
	pflag.CommandLine = pflag.NewFlagSet("olaris", pflag.ExitOnError)

	cmd.Execute()

	os.Exit(0)
}
