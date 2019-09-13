package host

import (
	"bytes"
	"context"
	_ "errors"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/cenkalti/backoff"

	"go.uber.org/multierr"

	"github.com/Strum355/log"

	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"
	"github.com/UCCNetworkingSociety/Windlass-worker/app/helpers"
	"github.com/UCCNetworkingSociety/Windlass-worker/utils/writecloser"
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

func (lxd *LXDHost) PushAuthCerts(ctx context.Context, opts ContainerHostCreateOptions, caPEM, serverKeyPEM, serverCertPEM []byte) error {
	err := multierr.Combine(
		errors.WithMessage(lxd.conn.CreateContainerFile(opts.Name, "/nginx/ca-cert.pem", lxdclient.ContainerFileArgs{
			UID: 0, GID: 0, Content: bytes.NewReader(caPEM), Mode: 400, Type: "file", WriteMode: "overwrite",
		}), "failed to push /nginx/ca-cert.pem"),
		errors.WithMessage(lxd.conn.CreateContainerFile(opts.Name, "/nginx/server-key.pem", lxdclient.ContainerFileArgs{
			UID: 0, GID: 0, Content: bytes.NewReader(serverKeyPEM), Mode: 400, Type: "file", WriteMode: "overwrite",
		}), "failed to push /nginx/server-key.pem"),
		errors.WithMessage(lxd.conn.CreateContainerFile(opts.Name, "/nginx/server-cert.pem", lxdclient.ContainerFileArgs{
			UID: 0, GID: 0, Content: bytes.NewReader(serverCertPEM), Mode: 400, Type: "file", WriteMode: "overwrite",
		}), "failed to push /nginx/server-cert.pem"),
	)

	return err
}

func (lxd *LXDHost) RestartNGINX(ctx context.Context, name string) error {
	exec := api.ContainerExecPost{
		Command:   []string{"systemctl", "restart", "nginx"},
		WaitForWS: true,
	}

	buf := &writecloser.BytesBuffer{bytes.NewBuffer(nil)}
	op, err := lxd.conn.ExecContainer(name, exec, &lxdclient.ContainerExecArgs{
		Stderr: buf,
	})
	if err != nil {
		return err
	}

	err = helpers.OperationTimeout(ctx, op)
	if err == context.DeadlineExceeded {
		return err
	}
	return errors.WithMessage(err, fmt.Sprintf("error restarting nginx: %s", buf.String()))
}
