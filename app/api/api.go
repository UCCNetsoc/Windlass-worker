package api

import (
	"encoding/json"
	"net/http"

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
	//api.routes.Use(middlechi.DefaultLogger)

	api.routes.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"apiVersion": "v1",
			"message":    "cool and well",
		})
	})

	api.routes.Route("/v1", func(r chi.Router) {

	})
}
