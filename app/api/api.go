package api

import (
	"github.com/go-chi/chi"
	middlechi "github.com/go-chi/chi/middleware"
)

type API struct {
	routes chi.Router
}

func NewAPI(router chi.Router) *API {
	return &API{
		routes: router,
	}
}

func (api *API) Init() {
	api.routes.Use(middlechi.RealIP)
	api.routes.Route("/v1", func(r chi.Router) {

	})
}
