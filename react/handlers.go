// Package react is a handler for the olaris-react application.
package react

//go:generate go-bindata-assetfs -pkg $GOPACKAGE build/...

import (
	"github.com/rs/cors"
	"net/http"
)

// GetHandler implements a handler that serves up the compiled olaris-react code.
func GetHandler() http.Handler {
	handler := cors.AllowAll().Handler(http.FileServer(assetFS()))

	return handler
}
