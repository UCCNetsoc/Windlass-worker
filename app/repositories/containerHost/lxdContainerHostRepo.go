package host

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"

	"go.uber.org/multierr"

	"github.com/Strum355/log"

	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"
	"github.com/UCCNetworkingSociety/Windlass-worker/app/helpers"
	lxdclient "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

type LXDHost struct {
	conn lxdclient.ContainerServer
}

// TODO context tiemouts
func NewLXDRepository() ContainerHostRepository {
	lxdHost, err := connections.GetLXD()
	if err != nil {
		panic(fmt.Sprintf("error getting LXD host: %v", err))
	}

	return &LXDHost{
		conn: lxdHost,
	}
}

func (lxd *LXDHost) Ping(ctx context.Context) error {
	return nil
}

func (lxd *LXDHost) parseError(err error) error {
	if err == nil {
		return nil
	}
	if strings.HasSuffix(err.Error(), "This container already exists") {
		return ErrHostExists
	}
	return err
}

func (lxd *LXDHost) CreateContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error {
	log.WithFields(log.Fields{
		"containerHostName": opts.Name,
	}).Debug("create container host request")

	op, err := lxd.conn.CreateContainer(api.ContainersPost{
		ContainerPut: api.ContainerPut{
			Devices: map[string]map[string]string{
				"eth0": {
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
		return lxd.parseError(err)
	}

	err = helpers.OperationTimeout(ctx, op)
	return lxd.parseError(err)
}

func (lxd *LXDHost) StartContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error {
	op, err := lxd.conn.UpdateContainerState(opts.Name, api.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}, "")
	if err != nil {
		return err
	}

	return helpers.OperationTimeout(ctx, op)
}

func (lxd *LXDHost) GetContainerHostIP(ctx context.Context, name string) (string, error) {
	var ip string
	retry := backoff.WithContext(backoff.NewConstantBackOff(time.Millisecond*5), ctx)
	f := func() error {
		state, _, err := lxd.conn.GetContainerState(name)
		if err != nil {
			return backoff.Permanent(err)
		}

		for _, addr := range state.Network["eth0"].Addresses {
			if addr.Family == "inet" {
				ip = addr.Address
				return nil
			}
		}
		return errors.New("failed to find ipv4 address for container")
	}

	return ip, backoff.Retry(f, retry)
}

func (lxd *LXDHost) PushAuthCerts(ctx context.Context, opts ContainerHostCreateOptions, caPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM []byte) error {
	err := multierr.Combine(
		lxd.conn.CreateContainerFile(opts.Name, "/nginx", lxdclient.ContainerFileArgs{
			UID: 33, GID: 33, Content: bytes.NewReader(caPEM), Mode: 400, Type: "file", WriteMode: "file",
		}),
		lxd.conn.CreateContainerFile(opts.Name, "/nginx", lxdclient.ContainerFileArgs{
			UID: 33, GID: 33, Content: bytes.NewReader(serverKeyPEM), Mode: 400, Type: "file", WriteMode: "file",
		}),
		lxd.conn.CreateContainerFile(opts.Name, "/nginx", lxdclient.ContainerFileArgs{
			UID: 33, GID: 33, Content: bytes.NewReader(serverCertPEM), Mode: 400, Type: "file", WriteMode: "file",
		}),
		lxd.conn.CreateContainerFile(opts.Name, "/nginx", lxdclient.ContainerFileArgs{
			UID: 33, GID: 33, Content: bytes.NewReader(clientKeyPEM), Mode: 400, Type: "file", WriteMode: "file",
		}),
		lxd.conn.CreateContainerFile(opts.Name, "/nginx", lxdclient.ContainerFileArgs{
			UID: 33, GID: 33, Content: bytes.NewReader(clientCertPEM), Mode: 400, Type: "file", WriteMode: "file",
		}),
	)

	return err
}
