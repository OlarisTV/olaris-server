package main

import (
	"flag"
	"github.com/peak6/envflag"
	"gitlab.com/olaris/olaris-server/cmd/olaris/cmd"
	"os"
)

func main() {
	flag.Parse()
	envflag.Parse()

	cmd.Execute()

	os.Exit(0)

}
