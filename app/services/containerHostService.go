package services

import (
	"context"
	"fmt"

	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"
	"github.com/spf13/viper"
)

type ContainerHostService struct {
	repo       host.ContainerHostRepository
	tlsService *TLSCertService
}

func NewContainerHostService() *ContainerHostService {
	hostService := &ContainerHostService{
		tlsService: NewTLSCertService(),
	}

	provider := viper.GetString("containerHost.type")

	if provider == "lxd" {
		hostService.repo = host.NewLXDRepository()
	} else {
		panic(fmt.Sprintf("invalid container host %s", provider))
	}

	return hostService
}

func (service *ContainerHostService) CreateHost(ctx context.Context, name string) error {
	options := host.ContainerHostCreateOptions{
		Name: name,
	}

	if err := service.repo.CreateContainerHost(ctx, options); err != nil {
		return err
	}

	if err := service.repo.StartContainerHost(ctx, options); err != nil {
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

	if err := service.repo.PushAuthCerts(ctx, options, pems.CAPEM, pems.ServerKeyPEM, pems.ServerCertPEM); err != nil {
		return err
	}

	if err := service.repo.RestartNGINX(ctx, name); err != nil {
		return err
	}

	return nil
}
