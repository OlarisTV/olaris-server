// Package react is a handler for the olaris-react application.
package react

import (
	"embed"
	"fmt"
	"github.com/rs/cors"
	"io/fs"
	"net/http"
)

//go:embed build/*
var embedded embed.FS

// GetHandler implements a handler that serves up the compiled olaris-react code.
func GetHandler() http.Handler {
	embeddedFS, err := fs.Sub(embedded, "build")

	if err != nil {
		panic(fmt.Sprintf("Failed to read embedded react files: %s", err.Error()))
	}

	handler := cors.AllowAll().Handler(http.FileServer(http.FS(embeddedFS)))

	return handler
}
