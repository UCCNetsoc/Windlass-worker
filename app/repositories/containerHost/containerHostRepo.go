package host

import (
	"context"
	"fmt"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/models/container"

	"github.com/spf13/viper"
)

type ContainerHostRepository interface {
	Ping(ctx context.Context) error
	UseCerts(clientKeyPEM, clientCertPEM, caPEM []byte)
	GetContainerHostIP(ctx context.Context, name string) (string, error)
	CreateContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error
	DeleteContainerHost(ctx context.Context, opts ContainerHostDeleteOptions) error
	StartContainerHost(ctx context.Context, opts ContainerHostStartOptions) error
	StopContainerHost(ctx context.Context, opts ContainerHostStopOptions) error
	PushAuthCerts(ctx context.Context, opts ContainerPushCertsOptions, caPEM, serverKeyPEM, serverCertPEM []byte) error
	RestartNGINX(ctx context.Context, name string) error
	CreateContainer(ctx context.Context, ctr container.Container) error
}

func NewContainerHostRepository() ContainerHostRepository {
	hostProvider := viper.GetString("containerHost.type")

	if hostProvider == "lxd" {
		return NewLXDRepository()
	}
	panic(fmt.Sprintf("invalid container host %s", hostProvider))
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
