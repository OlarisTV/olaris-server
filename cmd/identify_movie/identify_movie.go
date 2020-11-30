package identify_movie

import (
	"fmt"
	"strings"

	"github.com/goava/di"
	"github.com/spf13/cobra"

	"gitlab.com/olaris/olaris-server/cmd/identify"
	"gitlab.com/olaris/olaris-server/filesystem"
	"gitlab.com/olaris/olaris-server/metadata/agents"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"gitlab.com/olaris/olaris-server/metadata/managers/metadata"
	"gitlab.com/olaris/olaris-server/pkg/cmd"
)

type identifyMovieCommand cmd.Command

func RegisterIdentifyMovieCommand(rootCommand identify.IdentifyCommand, identifyMovieCommand identifyMovieCommand) {
	rootCommand.GetCobraCommand().AddCommand(identifyMovieCommand.GetCobraCommand())
}

func New() di.Option {
	return di.Options(
		di.Provide(NewIdentifyMovieCommand, di.As(new(identifyMovieCommand))),
		di.Invoke(RegisterIdentifyMovieCommand),
	)
}

func NewIdentifyMovieCommand() *cmd.CobraCommand {
	var filePath string
	var agent string
	var id int
	var dbConn string
	var dbLog bool

	c := &cobra.Command{
		Use:   "movie",
		Short: "Identify a movie",
		RunE: func(cmd *cobra.Command, args []string) error {
			var a agents.MetadataRetrievalAgent
			switch strings.ToLower(agent) {
			case "tmdb":
				a = agents.NewTmdbAgent()
			default:
				return fmt.Errorf("unknown agent: %s", agent)
			}

			fl, err := filesystem.ParseFileLocator(filePath)
			if err != nil {
				return err
			}

			n, err := filesystem.GetNodeFromFileLocator(fl)
			if err != nil {
				return err
			}

			dbOptions := db.DatabaseOptions{
				Connection: dbConn,
				LogMode:    dbLog,
			}
			mctx := app.NewMDContext(dbOptions, a)
			defer mctx.Db.Close()

			f, err := db.FindMovieFileByPath(n)
			if err != nil {
				return err
			}

			mm := metadata.NewMetadataManager(a)
			movie, err := mm.GetOrCreateMovieByTmdbID(id)
			if err != nil {
				return err
			}

			f.Movie = *movie
			db.SaveMovieFile(f)
			return nil
		},
	}

	c.Flags().StringVar(&filePath, "path", "", "Path of the movie file")
	c.MarkFlagRequired("path")

	c.Flags().IntVar(&id, "id", 0, "ID of movie from agent")
	c.MarkFlagRequired("id")

	c.Flags().StringVar(&agent, "agent", "tmdb", "Agent, defaults to tmdb")
	c.Flags().StringVar(&dbConn, "db-conn", "", "sets the database connection string")
	c.Flags().BoolVar(&dbLog, "db-log", false, "sets whether the database should log queries")

	return &cmd.CobraCommand{Command: c}
}
