package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

var libraryCmd = &cobra.Command{
	Use: "library",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Subcommand required")
	},
}

var name string
var path string
var mediaType int

var libraryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new library",
	RunE: func(cmd *cobra.Command, args []string) error {
		mctx := app.NewDefaultMDContext()
		defer mctx.Db.Close()

		_, err := db.AddLibrary(name, path, db.MediaType(mediaType))
		return err
	},
}

func init() {
	libraryCreateCmd.Flags().StringVar(&name, "name", "", "A name for this library")
	libraryCreateCmd.MarkFlagRequired("name")
	libraryCreateCmd.Flags().StringVar(&path, "path", "", "Path for this library")
	libraryCreateCmd.MarkFlagRequired("path")
	libraryCreateCmd.Flags().IntVar(&mediaType, "media_type", 0, "Media type, 0 for Movies, 1 for Series")
	libraryCmd.AddCommand(libraryCreateCmd)

	rootCmd.AddCommand(libraryCmd)
}
