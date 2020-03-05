package services

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/models/project"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/providers"

	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"
	tlsstorage "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/tlsStorage"
)

type ContainerHostService struct {
	repo           host.ContainerHostRepository
	consul         *providers.ConsulProvider
	tlsService     *TLSCertService
	tlsStorageRepo tlsstorage.TLSStorageRepo
}

func NewContainerHostService() *ContainerHostService {
	hostService := &ContainerHostService{
		tlsService: NewTLSCertService(),
	}

	hostService.repo = host.NewContainerHostRepository()

	hostService.tlsStorageRepo = tlsstorage.NewTLSStorageRepo()

	consul, err := providers.NewConsulProvider()
	if err != nil {
		panic(fmt.Sprintf("failed to get consul provider: %v", err))
	}
	hostService.consul = consul

	return hostService
}

// TODO: more to be part of ContainerHostCreateOptions
// TODO: better error handling, rollback changes on failure etc
func (service *ContainerHostService) CreateHost(ctx context.Context, name string) error {
	containerName := host.ContainerName{name}

	if err := service.repo.CreateContainerHost(ctx, host.ContainerHostCreateOptions{containerName}); err != nil {
		return fmt.Errorf("error creating host: %w", err)
	}

	if err := service.repo.StartContainerHost(ctx, host.ContainerHostStartOptions{containerName}); err != nil {
		return fmt.Errorf("error starting host: %w", err)
	}

	ip, err := service.repo.GetContainerHostIP(ctx, name)
	if err != nil {
		return fmt.Errorf("error getting host IP: %w", err)
	}

	pems, err := service.tlsService.CreatePEMs(ip)
	if err != nil {
		return fmt.Errorf("error creating TLS certs: %w", err)
	}

	if err := service.repo.PushAuthCerts(ctx, host.ContainerPushCertsOptions{containerName}, pems.ClientCAPEM, pems.ServerKeyPEM, pems.ServerCertPEM); err != nil {
		return fmt.Errorf("error pushing TLS certs to host: %w", err)
	}

	if err := service.repo.RestartNGINX(ctx, name); err != nil {
		return fmt.Errorf("error restarting nginx: %w", err)
	}

	service.repo.UseCerts(pems.ClientKeyPEM, pems.ClientCertPEM, pems.ClientCAPEM)

	if err := service.tlsStorageRepo.PushAuthCerts(ctx, containerName.Name, pems.ServerCAPEM, pems.ClientCAPEM, pems.ServerKeyPEM, pems.ServerCertPEM, pems.ClientKeyPEM, pems.ClientCertPEM); err != nil {
		return fmt.Errorf("error pushing TLS certs to storage: %w", err)
	}

	err = service.consul.RegisterProject(containerName.Name, ip, func(ip string) (string, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		err := service.repo.Ping(ctx)
		if err != nil {
			return err.Error(), false
		}
		return "Remote Docker daemon reachable", true
	})
	if err != nil {
		return fmt.Errorf("error registering project and/or health check: %w", err)
	}

	return nil
}

func (service *ContainerHostService) CreateServices(ctx context.Context, name string, data project.Project) error {
	/* pems, err := service.tlsStorageRepo.GetAuthCerts(ctx, name)
	if err != nil {
		return err
	} */

	var err *multierror.Error

	for _, container := range data.Containers {
		multierror.Append(err, service.repo.CreateContainer(ctx, container))
	}

	return err.ErrorOrNil()
}
