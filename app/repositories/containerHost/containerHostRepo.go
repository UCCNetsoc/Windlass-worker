package host

import (
	"context"
)

type ContainerHostRepository interface {
	Ping(ctx context.Context) error
	GetContainerHostIP(ctx context.Context, name string) (string, error)
	CreateContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error
	StartContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error
	PushAuthCerts(ctx context.Context, opts ContainerHostCreateOptions, caPEM, serverKeyPEM, serverCertPEM []byte) error
	RestartNGINX(ctx context.Context, name string) error
}

type ContainerHostCreateOptions struct {
	Name string
}
