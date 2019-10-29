package services

import (
	"context"
	"fmt"

	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"
	tlsstorage "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/tlsStorage"
	"github.com/spf13/viper"
)

type ContainerHostService struct {
	repo           host.ContainerHostRepository
	tlsService     *TLSCertService
	tlsStorageRepo tlsstorage.TLSStorageRepo
}

func NewContainerHostService() *ContainerHostService {
	hostService := &ContainerHostService{
		tlsService: NewTLSCertService(),
	}

	hostProvider := viper.GetString("containerHost.type")

	if hostProvider == "lxd" {
		hostService.repo = host.NewLXDRepository()
	} else {
		panic(fmt.Sprintf("invalid container host %s", hostProvider))
	}

	if viper.GetBool("vault.enabled") {
		repo, err := tlsstorage.NewVaultTLSStorageRepo()
		if err != nil {
			panic(fmt.Errorf("failed to get TLS storage repo: %w", err))
		}
		hostService.tlsStorageRepo = repo
	} else {
		panic("vault currently required")
	}

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

	if err := service.repo.PushAuthCerts(ctx, host.ContainerPushCertsOptions{containerName}, pems.CAPEM, pems.ServerKeyPEM, pems.ServerCertPEM); err != nil {
		return err
	}

	if err := service.repo.RestartNGINX(ctx, name); err != nil {
		return err
	}

	if err := service.tlsStorageRepo.PushAuthCerts(ctx, containerName.Name, pems.CAPEM, pems.ServerKeyPEM, pems.ServerCertPEM, pems.ClientKeyPEM, pems.ClientCertPEM); err != nil {
		return err
	}

	return nil
}
