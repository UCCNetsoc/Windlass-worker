package project

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/models/container"
)

var (
	ErrInvalidFormat = errors.New("project name bad format")
	ErrNameTooLong   = errors.New("project name too long")
)

var (
	projectName = regexp.MustCompile(`^[A-Za-z]([A-Za-z0-9\-])*[A-Za-z0-9]$`)
)

type Project struct {
	Name         string               `json:"name"`
	Namespace    string               `json:"namespace"`
	Containers   container.Containers `json:"containers"`
	CreationDate time.Time            `json:"createdAt"`
	UpdatedDate  time.Time            `json:"updatedAt"`
}

func (p *Project) Bind(r *http.Request) error {
	if !projectName.MatchString(p.Namespace + "-" + p.Name) {
		return ErrInvalidFormat
	}

	if (len(p.Namespace) + len(p.Name) + 1) > 62 {
		return ErrNameTooLong
	}

	return nil
}
