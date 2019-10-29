package host

import (
	"context"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type ContainerHostRepository interface {
	Pinger
	GetContainerHostIP(ctx context.Context, name string) (string, error)
	CreateContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error
	DeleteContainerHost(ctx context.Context, opts ContainerHostDeleteOptions) error
	StartContainerHost(ctx context.Context, opts ContainerHostStartOptions) error
	StopContainerHost(ctx context.Context, opts ContainerHostStopOptions) error
	PushAuthCerts(ctx context.Context, opts ContainerPushCertsOptions, caPEM, serverKeyPEM, serverCertPEM []byte) error
	RestartNGINX(ctx context.Context, name string) error
}

type ContainerName struct {
	Name string
}

type ContainerHostCreateOptions struct {
	ContainerName
}

type ContainerHostDeleteOptions struct {
	ContainerName
}

type ContainerHostStopOptions struct {
	ContainerName
}

type ContainerHostStartOptions struct {
	ContainerName
}

type ContainerPushCertsOptions struct {
	ContainerName
}
