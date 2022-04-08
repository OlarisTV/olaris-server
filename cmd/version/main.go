package version

import (
	"fmt"
	"github.com/goava/di"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type VersionCommand cmd.Command

func RegisterVersionCommand(rootCommand root.RootCommand, versionCommand VersionCommand) {
	rootCommand.GetCobraCommand().AddCommand(versionCommand.GetCobraCommand())
}

func New() di.Option {
	return di.Options(
		di.Provide(NewVersionCommand, di.As(new(VersionCommand))),
		di.Invoke(RegisterVersionCommand),
	)
}

func NewVersionCommand() *cmd.CobraCommand {
	c := &cobra.Command{
		Use:   "version",
		Short: "Displays the current olaris-server version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(helpers.Version)
		},
	}

	return &cmd.CobraCommand{Command: c}
}
