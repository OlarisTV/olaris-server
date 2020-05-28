package config

import (
	"github.com/spf13/pflag"
)

// hack to get rid of the unwanted extra pflags from rclone/fs/log
// not really a hack anymore but this should be in the container
var flagSet = pflag.NewFlagSet("olaris", pflag.ExitOnError)

func init() {
	pflag.CommandLine = flagSet
}

func provideFlags() *pflag.FlagSet {
	return pflag.CommandLine
}

func RegisterFlags(f func(*pflag.FlagSet)) {
	f(flagSet)
}
