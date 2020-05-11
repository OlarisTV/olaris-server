package resolvers

//go:generate sh -c "printf 'package resolvers\n\nvar schemaTxt = `%s`\n' \"$(cat schema.graphql)\" > schematxt.go"

import (
	"github.com/graph-gophers/graphql-go"
	"gitlab.com/olaris/olaris-server/metadata/app"
)

// InitSchema inits the graphql schema.
func InitSchema(env *app.MetadataContext) *graphql.Schema {
	schema := graphql.MustParseSchema(schemaTxt, NewResolver(env))
	return schema
}
