// Package app is a handler for the olaris-react application.
package app

//go:generate go-bindata-assetfs -pkg $GOPACKAGE build/...

import (
	"github.com/rs/cors"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"net/http"
)

// GetHandler implements a handler that serves up the compiled olaris-react code.
func GetHandler(menv *db.MetadataContext) http.Handler {
	handler := cors.AllowAll().Handler(http.FileServer(assetFS()))

	return handler
}
