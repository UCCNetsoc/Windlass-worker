package lxd

import (
	"context"
	"fmt"

	"github.com/spf13/viper"

	log "github.com/UCCNetworkingSociety/Windlass-worker/utils/logging"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"
	"github.com/UCCNetworkingSociety/Windlass-worker/app/helpers"
	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"
	lxdclient "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

type LXDHost struct {
	ctx  context.Context
	conn lxdclient.ContainerServer
}

// TODO context tiemouts
func NewLXDRepository() host.ContainerHostRepository {
	lxdHost, err := connections.GetLXD()
	if err != nil {
		panic(fmt.Sprintf("error getting LXD host: %v", err))
	}

	return &LXDHost{
		ctx:  context.Background(),
		conn: lxdHost,
	}
}

func (lxd *LXDHost) WithContext(ctx context.Context) host.ContainerHostRepository {
	lxd.ctx = ctx
	return lxd
}

func (lxd *LXDHost) Ping() error {
	return nil
}

func (lxd *LXDHost) CreateContainerHost(context context.Context, opts host.ContainerHostCreateOptions) error {
	log.WithFields(log.Fields{
		"containerHostName": opts.Name,
	}).Debug("create container host request")

	op, err := lxd.conn.CreateContainer(api.ContainersPost{
		ContainerPut: api.ContainerPut{
			Devices: map[string]map[string]string{
				"eth0": map[string]string{
					"type":         "nic",
					"nictype":      "bridged",
					"name":         "eth0",
					"parent":       "windlassbr0",
					"ipv4.address": "10.69.1.5", // sample data
				},
			},
			Config: map[string]string{
				"security.nesting": "true",
			},
		},
		Name: opts.Name,
		Source: api.ContainerSource{
			Type:        "image",
			Fingerprint: viper.GetString("lxd.baseImage"),
		},
	})
	if err != nil {
		return err
	}

	err = helpers.OperationTimeout(lxd.ctx, op)
	return err
}

func (lxd *LXDHost) StartContainerHost(context context.Context, opts host.ContainerHostCreateOptions) error {
	op, err := lxd.conn.UpdateContainerState(opts.Name, api.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}, "")
	if err != nil {
		return err
	}

	return helpers.OperationTimeout(lxd.ctx, op)
}
