package resolvers

import (
	"gitlab.com/olaris/olaris-server/metadata/app"
	"testing"
)

func TestInitSchema(t *testing.T) {
	env := app.NewTestingMDContext(nil)
	InitSchema(env)
}
