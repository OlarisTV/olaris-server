package user

import (
	"errors"

	"github.com/goava/di"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type UserCommand cmd.Command

func New() di.Option {
	return di.Options(
		di.Provide(NewUserCommand, di.As(new(UserCommand))),
		di.Invoke(RegisterUserCommand),
	)
}

func RegisterUserCommand(rootCommand root.RootCommand, userCommand UserCommand) {
	rootCommand.GetCobraCommand().AddCommand(userCommand.GetCobraCommand())
}

func NewUserCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("Subcommand required")
		},
	}

	return &cmd.CobraCommand{Command: c}
}
