package api

import (
	"net/http"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/api/models"
	"github.com/go-chi/render"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
	api.routes.Use(middlechi.DefaultLogger)
	api.routes.Use(middleware.Recoverer)
	api.routes.Use(render.SetContentType(render.ContentTypeJSON))

	api.routes.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.Render(w, r, &models.APIResponse{
			Status: http.StatusOK,
		})
	})

	api.routes.Route("/v1", func(r chi.Router) {

	})
}
