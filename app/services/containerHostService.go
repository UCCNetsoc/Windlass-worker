package services

import (
	"context"
	"fmt"
	"time"

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
		return err
	}

	if err := service.repo.StartContainerHost(ctx, host.ContainerHostStartOptions{containerName}); err != nil {
		return err
	}

	ip, err := service.repo.GetContainerHostIP(ctx, name)
	if err != nil {
		return err
	}

	pems, err := service.tlsService.CreatePEMs(ip)
	if err != nil {
		return err
	}

	if err := service.repo.PushAuthCerts(ctx, host.ContainerPushCertsOptions{containerName}, pems.ClientCAPEM, pems.ServerKeyPEM, pems.ServerCertPEM); err != nil {
		return err
	}

	if err := service.repo.RestartNGINX(ctx, name); err != nil {
		return err
	}

	service.repo.UseCerts(pems.ClientKeyPEM, pems.ClientCertPEM, pems.ClientCAPEM)

	if err := service.tlsStorageRepo.PushAuthCerts(ctx, containerName.Name, pems.ServerCAPEM, pems.ClientCAPEM, pems.ServerKeyPEM, pems.ServerCertPEM, pems.ClientKeyPEM, pems.ClientCertPEM); err != nil {
		return err
	}

	service.consul.RegisterProject(containerName.Name, ip, func(ip string) (string, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		err := service.repo.Ping(ctx)
		if err != nil {
			return err.Error(), false
		}
		return "Remote Docker daemon reachable", true
	})

	return nil
}
