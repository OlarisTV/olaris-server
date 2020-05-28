package user_create

import (
	"github.com/goava/di"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/user"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type UserCreateCommand cmd.Command

func New() di.Option {
	return di.Options(
		di.Provide(NewUserCreateCommand, di.As(new(UserCreateCommand))),
		di.Invoke(RegisterUserCreateCommand),
	)
}

func RegisterUserCreateCommand(userCommand user.UserCommand, userCreateCommand UserCreateCommand) {
	userCommand.GetCobraCommand().AddCommand(userCreateCommand.GetCobraCommand())
}

func NewUserCreateCommand() *cmd.CobraCommand {
	var username string
	var password string
	var admin bool

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			mctx := app.NewDefaultMDContext()
			defer mctx.Db.Close()

			_, err := db.CreateUser(username, password, admin)

			return err
		},
	}

	c.Flags().StringVar(&username, "username", "", "")
	c.MarkFlagRequired("username")

	c.Flags().StringVar(&password, "password", "", "")
	c.MarkFlagRequired("password")

	c.Flags().BoolVar(&admin, "admin", false, "Whether the new user should be an admin")

	return &cmd.CobraCommand{Command: c}
}
