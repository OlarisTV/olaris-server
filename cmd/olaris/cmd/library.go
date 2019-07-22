package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
)

var libraryCmd = &cobra.Command{
	Use:   "library",
	Short: "Manage libraries",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Subcommand required")
	},
}

var name string
var filePath string
var mediaType int
var backendType int
var rcloneName string

var libraryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new library",
	RunE: func(cmd *cobra.Command, args []string) error {
		mctx := app.NewDefaultMDContext(true, false)
		defer mctx.Db.Close()

		err := db.AddLibrary(&db.Library{Name: name, FilePath: filePath, Kind: db.MediaType(mediaType), Backend: backendType, RcloneName: rcloneName})
		return err
	},
}

func init() {
	libraryCreateCmd.Flags().StringVar(&name, "name", "", "A name for this library")
	libraryCreateCmd.MarkFlagRequired("name")

	libraryCreateCmd.Flags().StringVar(&filePath, "path", "", "Path for this library")
	libraryCreateCmd.MarkFlagRequired("path")

	libraryCreateCmd.Flags().IntVar(&mediaType, "media_type", 0, "Media type, 0 for Movies, 1 for Series")
	libraryCreateCmd.Flags().IntVar(&backendType, "backend_type", 0, "Backend type, 0 for Local, 1 for Rclone")
	libraryCreateCmd.Flags().StringVar(&rcloneName, "rclone_name", "", "Name for the Rclone remote")

	libraryCmd.AddCommand(libraryCreateCmd)

	rootCmd.AddCommand(libraryCmd)
}
