package project

import (
	"errors"
	"regexp"

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
	Name       string               `json:"name"`
	Namespace  string               `json:"namespace"`
	Containers container.Containers `json:"containers"`
}

func (p Project) Validate() error {
	if !projectName.MatchString(p.Namespace + "_" + p.Name) {
		return ErrInvalidFormat
	}

	if (len(p.Namespace) + len(p.Name) + 1) > 62 {
		return ErrNameTooLong
	}

	return nil
}
