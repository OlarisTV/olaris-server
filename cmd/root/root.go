package root

import (
	"github.com/goava/di"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type RootCommand cmd.Command

func New() di.Option {
	return di.Options(
		di.Provide(NewRootCommand, di.As(new(RootCommand))),
	)
}

// Execute is the main launcher for the root command; this is
// where we bind some useful flags
func NewRootCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use: "olaris",
	}

	return &cmd.CobraCommand{Command: c}
}
