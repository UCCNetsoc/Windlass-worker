package host

import (
	"context"
)

type ContainerHostRepository interface {
	WithContext(context.Context) ContainerHostRepository
	Ping() error
	CreateContainerHost(context.Context, ContainerHostCreateOptions) error
	StartContainerHost(context.Context, ContainerHostCreateOptions) error
}

type ContainerHostCreateOptions struct {
	Name string
}
