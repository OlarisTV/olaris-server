package library_create

import (
	"time"

	"github.com/goava/di"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

const defaultTimeOffset = -24 * time.Hour

type LibraryCreateCommand cmd.Command

func RegisterLibraryCreateCommand(rootCommand root.RootCommand, libraryCreateCommand LibraryCreateCommand) {
	rootCommand.GetCobraCommand().AddCommand(libraryCreateCommand.GetCobraCommand())
}

func New() di.Option {
	return di.Options(
		di.Provide(NewLibraryCreateCommand, di.As(new(LibraryCreateCommand))),
		di.Invoke(RegisterLibraryCreateCommand),
	)
}

func NewLibraryCreateCommand() *cmd.CobraCommand {
	var name string
	var filePath string
	var mediaType int
	var backendType int
	var rcloneName string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new library",
		RunE: func(cmd *cobra.Command, args []string) error {
			mctx := app.NewDefaultMDContext()
			defer mctx.Db.Close()

			lib := &db.Library{Name: name, FilePath: filePath, Kind: db.MediaType(mediaType), Backend: backendType, RcloneName: rcloneName}

			// Make sure we don't initialize the library with zero time (issue with strict mode in MySQL)
			lib.RefreshStartedAt = time.Now().Add(defaultTimeOffset)
			lib.RefreshCompletedAt = time.Now().Add(defaultTimeOffset)

			err := db.AddLibrary(lib)
			return err
		},
	}

	c.Flags().StringVar(&name, "name", "", "A name for this library")
	c.MarkFlagRequired("name")

	c.Flags().StringVar(&filePath, "path", "", "Path for this library")
	c.MarkFlagRequired("path")

	c.Flags().IntVar(&mediaType, "media_type", 0, "Media type, 0 for Movies, 1 for Series")
	c.Flags().IntVar(&backendType, "backend_type", 0, "Backend type, 0 for Local, 1 for Rclone")
	c.Flags().StringVar(&rcloneName, "rclone_name", "", "Name for the Rclone remote")

	c.Flags().IntP("port", "p", 8080, "http port")
	c.Flags().BoolP("verbose", "v", true, "verbose logging")
	c.Flags().Bool("db-log", false, "sets whether the database should log queries")
	c.Flags().String("db-conn", "", "sets the database connection string")

	return &cmd.CobraCommand{Command: c}
}
