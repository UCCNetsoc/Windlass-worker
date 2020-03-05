package host

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"

	"github.com/cenkalti/backoff"

	lxd "github.com/lxc/lxd/client"

	"github.com/Strum355/log"

	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/helpers"
	"github.com/UCCNetworkingSociety/Windlass-worker/app/models/container"
	"github.com/UCCNetworkingSociety/Windlass-worker/utils/writecloser"
	lxdclient "github.com/lxc/lxd/client"
	"github.com/lxc/lxd/shared/api"
)

var lxdConn lxd.ContainerServer

type lxdHost struct {
	conn       lxdclient.ContainerServer
	dockerConn *docker.Client
	ip         string

	// certs for interacting with the host's Docker daemon
	clientKeyPEM  []byte
	clientCertPEM []byte
	caPEM         []byte
}

func getLXD() (lxd.ContainerServer, error) {
	if lxdConn != nil {
		return lxdConn, nil
	}

	lxdconn, err := lxd.ConnectLXDUnix(viper.GetString("lxd.socket"), &lxd.ConnectionArgs{
		UserAgent: "Windlass",
	})
	if err != nil {
		return nil, fmt.Errorf("couldnt connect to LXD socket: %v", err)
	}

	lxdConn = lxdconn

	return lxdConn, nil
}

// TODO context tiemouts
func NewLXDRepository() ContainerHostRepository {
	lxdHostConn, err := getLXD()
	if err != nil {
		panic(fmt.Sprintf("error getting LXD host: %v", err))
	}

	return &lxdHost{
		conn: lxdHostConn,
	}
}

func (lxd *lxdHost) createDockerConn() error {
	if lxd.dockerConn == nil {
		client, err := docker.NewTLSClientFromBytes("https://"+lxd.ip, lxd.clientCertPEM, lxd.clientKeyPEM, lxd.caPEM)
		if err != nil {
			return err
		}
		lxd.dockerConn = client
	}
	return nil
}

func (lxd *lxdHost) Ping(ctx context.Context) error {
	if err := lxd.createDockerConn(); err != nil {
		return err
	}
	/* log.WithFields({
		"ip": lxd.ip,
	}).Info("pinging docker endpoint") */
	return lxd.dockerConn.PingWithContext(ctx)
}

func (lxd *lxdHost) parseError(err error) error {
	if err == nil {
		return nil
	}
	if strings.HasSuffix(err.Error(), "This container already exists") {
		return ErrHostExists
	}
	return err
}

func (lxd *lxdHost) CreateContainerHost(ctx context.Context, opts ContainerHostCreateOptions) error {
	log.WithFields(log.Fields{
		"containerHost": opts.Name,
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

func (lxd *lxdHost) DeleteContainerHost(ctx context.Context, opts ContainerHostDeleteOptions) error {
	op, err := lxd.conn.DeleteContainer(opts.Name)
	if err != nil {
		return err
	}
	return helpers.OperationTimeout(ctx, op)
}

func (lxd *lxdHost) StartContainerHost(ctx context.Context, opts ContainerHostStartOptions) error {
	op, err := lxd.conn.UpdateContainerState(opts.Name, api.ContainerStatePut{
		Action:  "start",
		Timeout: -1,
	}, "")
	if err != nil {
		return err
	}

	return helpers.OperationTimeout(ctx, op)
}

func (lxd *lxdHost) StopContainerHost(ctx context.Context, opts ContainerHostStopOptions) error {
	op, err := lxd.conn.UpdateContainerState(opts.Name, api.ContainerStatePut{
		Action:  "stop",
		Timeout: -1,
	}, "")
	if err != nil {
		return err
	}

	return helpers.OperationTimeout(ctx, op)
}

func (lxd *lxdHost) GetContainerHostIP(ctx context.Context, name string) (string, error) {
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
				lxd.ip = ip
				return nil
			}
		}
		return errors.New("failed to find ipv4 address for container")
	}

	return ip, backoff.Retry(f, retry)
}

func (lxd *lxdHost) PushAuthCerts(ctx context.Context, opts ContainerPushCertsOptions, caPEM, serverKeyPEM, serverCertPEM []byte) error {
	var err *multierror.Error

	multierror.Append(err,
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

	return err.ErrorOrNil()
}

func (lxd *lxdHost) UseCerts(clientKeyPEM, clientCertPEM, caPEM []byte) {
	lxd.clientKeyPEM = clientKeyPEM
	lxd.clientCertPEM = clientCertPEM
	lxd.caPEM = caPEM
}

func (lxd *lxdHost) RestartNGINX(ctx context.Context, name string) error {
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

func (lxd *lxdHost) CreateContainer(ctx context.Context, ctr container.Container) error {
	if err := lxd.createDockerConn(); err != nil {
		return err
	}

	mounts := make([]docker.Mount, 0, len(ctr.Mounts))

	for _, mount := range ctr.Mounts {
		mounts = append(mounts, docker.Mount{
			Destination: mount.Destination,
			Source:      mount.Source,
			RW:          mount.RW,
		})
	}

	env := make([]string, 0, len(ctr.Env))

	for k, v := range ctr.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	ports := make(map[docker.Port][]docker.PortBinding)

	for _, portMap := range ctr.Ports {
		ports[docker.Port(fmt.Sprintf("%d/tcp", portMap.ContainerPort))] = []docker.PortBinding{
			{HostPort: fmt.Sprintf("%d", portMap.HostPort)},
		}
	}

	splitImage := strings.Split(ctr.Image, ":")
	if err := lxd.dockerConn.PullImage(docker.PullImageOptions{
		Repository: ctr.Image,
		Tag:        splitImage[len(splitImage)-1],
	}, docker.AuthConfiguration{}); err != nil {
		return fmt.Errorf("error pulling image: %w", err)
	}

	newCtr, err := lxd.dockerConn.CreateContainer(docker.CreateContainerOptions{
		Context: ctx,
		Name:    ctr.Name,
		Config: &docker.Config{
			Image:  ctr.Image,
			Cmd:    []string{ctr.Command},
			Labels: ctr.Labels,
			Mounts: mounts,
			Env:    env,
		},
		HostConfig: &docker.HostConfig{
			PortBindings: ports,
		},
	})
	if err != nil {
		return fmt.Errorf("error creating container: %w", err)
	}

	log.WithFields(log.Fields{
		"container": fmt.Sprintf("%#v", newCtr),
	}).Info("created new container")

	if err := lxd.dockerConn.StartContainer(newCtr.ID, nil); err != nil {
		return fmt.Errorf("error starting container: %w", err)
	}

	return nil
}
