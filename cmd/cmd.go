package cmd

import (
	"github.com/goava/di"

	"gitlab.com/olaris/olaris-server/cmd/dumpdebug"
	"gitlab.com/olaris/olaris-server/cmd/identify"
	"gitlab.com/olaris/olaris-server/cmd/identify_movie"
	"gitlab.com/olaris/olaris-server/cmd/library"
	"gitlab.com/olaris/olaris-server/cmd/library_create"
	"gitlab.com/olaris/olaris-server/cmd/root"
	"gitlab.com/olaris/olaris-server/cmd/serve"
	"gitlab.com/olaris/olaris-server/cmd/user"
	"gitlab.com/olaris/olaris-server/cmd/user_create"
	"gitlab.com/olaris/olaris-server/cmd/version"
)

func New() di.Option {
	return di.Options(
		root.New(),
		user.New(),
		user_create.New(),
		serve.New(),
		identify.New(),
		identify_movie.New(),
		library.New(),
		library_create.New(),
		dumpdebug.New(),
		version.New(),
	)
}
