package v1

import (
	"net/http"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/services"
	"github.com/go-chi/chi"
)

type ProjectEndpoint struct {
	hostService *services.ContainerHostService
}

func NewProjectEndpoints(r chi.Router) {
	projectEndpoint := ProjectEndpoint{
		hostService: services.NewContainerHostService(),
	}

	r.Route("/projects", func(r chi.Router) {
		r.Post("/", projectEndpoint.createProject)
	})
}

func (p *ProjectEndpoint) createProject(w http.ResponseWriter, r *http.Request) {

}
