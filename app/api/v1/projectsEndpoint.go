package v1

import (
	"net/http"
	"time"

	"github.com/UCCNetworkingSociety/Windlass-worker/middleware"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/api/models"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/models/project"
	"github.com/go-chi/render"

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
		r.Post("/", middleware.WithContext(projectEndpoint.createProject, time.Second*20))
	})
}

func (p *ProjectEndpoint) createProject(w http.ResponseWriter, r *http.Request) {
	var newProject project.Project
	if err := render.Bind(r, &newProject); err != nil {
		render.Render(w, r, models.APIResponse{
			Status:  http.StatusBadRequest,
			Content: err.Error(),
		})
		return
	}

	if err := p.hostService.CreateHost(r.Context(), newProject.HostName()); err != nil {
		render.Render(w, r, models.APIResponse{
			Status:  http.StatusInternalServerError,
			Content: err.Error(),
		})
		return
	}

}
