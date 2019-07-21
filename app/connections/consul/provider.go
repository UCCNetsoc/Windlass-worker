package consul

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/UCCNetworkingSociety/Windlass-worker/utils/logging"

	"github.com/spf13/viper"

	"github.com/hashicorp/consul/api"
)

type Provider struct {
	id                 string
	address            string
	port               int
	client             *api.Client
	deregisterCritical time.Duration
	ttl                time.Duration
	refreshTTL         time.Duration
}

func NewProvider(conf *api.Config) (*Provider, error) {
	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:     client,
		ttl:        time.Second * 10,
		refreshTTL: time.Second * 5,
	}, nil
}

func (p *Provider) Client() *api.Client {
	return p.client
}

// Register registers the worker and its associated projects with Consul.
// See registerWorker() and registerProjects() for specific details
func (p *Provider) Register() error {
	p.id = fmt.Sprintf("windlass-worker@%s:%s", viper.GetString("http.address"), strconv.Itoa(viper.GetInt("http.port")))
	p.port = viper.GetInt("http.port")
	p.address = viper.GetString("http.address")

	if err := p.registerWorker(); err != nil {
		return err
	}

	if err := p.registerProjects(); err != nil {
		return err
	}

	p.udpateWorkerTTL()

	p.updateProjectTTL()

	return nil
}

// Registers this worker with Consul using a TTL based health check
func (p *Provider) registerWorker() error {
	service := &api.AgentServiceRegistration{
		ID:      p.id,
		Name:    "windlass-worker",
		Address: p.address,
		Port:    p.port,
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: p.deregisterCritical.String(),
			TTL:                            p.ttl.String(),
		},
	}

	return p.client.Agent().ServiceRegister(service)
}

// Registers the projects associated with this worker with Consul using a TTL based health check
// It pings the Docker Daemon on each project, see https://docs.docker.com/engine/api/v1.37/#operation/SystemPing for details
func (p *Provider) registerProjects() error {

	return nil
}

// Updates the TTL for this worker
func (p *Provider) udpateWorkerTTL() {
	go func() {
		ticker := time.NewTicker(p.ttl / 2)
		for range ticker.C {
			if err := p.client.Agent().UpdateTTL("service:"+p.id, "cool and well", api.HealthPassing); err != nil {
				log.Error(err, "failed to update TTL")
			}
		}
	}()
}

// Updates the TTL for each project associated with this worker
func (p *Provider) updateProjectTTL() {

}

func (p *Provider) getIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("couldnt find IP")
}
