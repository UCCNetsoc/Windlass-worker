package api

import (
	"net/http"

	"github.com/99designs/basicauth-go"
	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/middleware"

	v1 "github.com/UCCNetworkingSociety/Windlass-worker/app/api/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/api/models"
	"github.com/go-chi/render"

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
	api.routes.Use(middlechi.DefaultLogger)
	api.routes.Use(middleware.Recoverer)
	api.routes.Use(render.SetContentType(render.ContentTypeJSON))

	api.routes.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.Render(w, r, &models.APIResponse{
			Status: http.StatusOK,
		})
	})

	api.routes.Handle("/metrics", basicauth.New("banana", map[string][]string{
		viper.GetString("http.basicauth.user"): {viper.GetString("http.basicauth.pass")},
	})(promhttp.Handler()))

	api.routes.Route("/v1", func(r chi.Router) {
		v1.NewProjectEndpoints(r)
	})
}
