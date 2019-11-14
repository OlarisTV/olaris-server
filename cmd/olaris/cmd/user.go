package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Subcommand required")
	},
}

var username string
var password string
var admin bool

// createCmd represents the create command
var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		mctx := app.NewDefaultMDContext()
		defer mctx.Db.Close()

		_, err := db.CreateUser(username, password, admin)

		return err
	},
}

func init() {
	userCreateCmd.Flags().StringVar(&username, "username", "", "")
	userCreateCmd.MarkFlagRequired("username")
	userCreateCmd.Flags().StringVar(&password, "password", "", "")
	userCreateCmd.MarkFlagRequired("password")
	userCreateCmd.Flags().BoolVar(&admin, "admin", false, "Whether the new user should be an admin")
	userCmd.AddCommand(userCreateCmd)

	rootCmd.AddCommand(userCmd)
}
