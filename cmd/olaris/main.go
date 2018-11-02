package main

import (
	goflag "flag"
	"github.com/peak6/envflag"
	"github.com/spf13/pflag"
	"gitlab.com/olaris/olaris-server/cmd/olaris/cmd"
	"os"
)

func main() {
	envflag.Parse()

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	cmd.Execute()

	os.Exit(0)

}
