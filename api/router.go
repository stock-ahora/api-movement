// api/router.go
package api

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *MovementHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/movements", handler.GetMovements)
	r.Get("/movements/{id}", handler.GetMovementByID)

	return r
}