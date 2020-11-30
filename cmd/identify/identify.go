package identify

import (
	"github.com/goava/di"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type IdentifyCommand cmd.Command

func RegisterIdentifyCommand(rootCommand root.RootCommand, identifyCommand IdentifyCommand) {
	rootCommand.GetCobraCommand().AddCommand(identifyCommand.GetCobraCommand())
}

func New() di.Option {
	return di.Options(
		di.Provide(NewIdentifyCommand, di.As(new(IdentifyCommand))),
		di.Invoke(RegisterIdentifyCommand),
	)
}

func NewIdentifyCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use:   "identify",
		Short: "Identify media files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Subcommand required")
		},
	}

	return &cmd.CobraCommand{Command: c}
}
