package web

import "github.com/gorilla/mux"

// Controller wraps a set of routers that can be registered on a router.
type Controller interface {
	RegisterRoutes(r *mux.Router)
}
