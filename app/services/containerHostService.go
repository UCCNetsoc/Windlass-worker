package services

import (
	"context"
	"fmt"

	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"
	"github.com/spf13/viper"
)

type ContainerHostService struct {
	repo host.ContainerHostRepository
}

func NewContainerHostService() *ContainerHostService {
	hostService := &ContainerHostService{}

	provider := viper.GetString("containerHost.type")

	if provider == "lxd" {
		hostService.repo = host.NewLXDRepository()
	} else {
		panic(fmt.Sprintf("invalid container host %s", provider))
	}

	return hostService
}

func (service *ContainerHostService) WithContext(ctx context.Context) *ContainerHostService {
	service.repo.WithContext(ctx)
	return service
}

func (service *ContainerHostService) CreateHost(context context.Context, name string) error {
	options := host.ContainerHostCreateOptions{
		Name: name,
	}

	if err := service.repo.CreateContainerHost(context, options); err != nil {
		return err
	}

	if err := service.repo.StartContainerHost(context, options); err != nil {
		return err
	}

	return nil
}
