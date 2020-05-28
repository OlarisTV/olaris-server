package library

import (
	"github.com/goava/di"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type LibraryCommand cmd.Command

func RegisterLibraryCommand(rootCommand root.RootCommand, libraryCommand LibraryCommand) {
	rootCommand.GetCobraCommand().AddCommand(libraryCommand.GetCobraCommand())
}

func New() di.Option {
	return di.Options(
		di.Provide(NewLibraryCommand, di.As(new(LibraryCommand))),
		di.Invoke(RegisterLibraryCommand),
	)
}

func NewLibraryCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use:   "library",
		Short: "Manage libraries",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Subcommand required")
		},
	}

	return &cmd.CobraCommand{Command: c}
}
