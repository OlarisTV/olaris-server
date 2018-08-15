package app

//go:generate go-bindata-assetfs -pkg $GOPACKAGE build/...

import (
	_ "github.com/gorilla/mux"
	"github.com/rs/cors"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"net/http"
)

func GetHandler(menv *db.MetadataContext) http.Handler {
	handler := cors.AllowAll().Handler(http.FileServer(assetFS()))

	return handler
}
