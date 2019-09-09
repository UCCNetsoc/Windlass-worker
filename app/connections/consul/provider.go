package consul

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Strum355/log"

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
	ttlError           error
	secretGetRetry     int
}

func NewProvider(conf *api.Config) (*Provider, error) {
	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:         client,
		ttl:            time.Second * 10,
		refreshTTL:     time.Second * 5,
		secretGetRetry: 5,
	}, nil
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
			health := api.HealthPassing
			if viper.GetString("windlass.secret") == "" {
				health = api.HealthCritical
			}

			err := p.client.Agent().UpdateTTL("service:"+p.id, "", health)
			p.ttlError = err
			if err != nil {
				p.onFailedTTL()
				log.WithError(err).Error("failed to update TTL")
			}
		}
	}()
}

// Updates the TTL for each project associated with this worker
func (p *Provider) updateProjectTTL() {

}

func (p *Provider) GetAndSetSharedSecret() error {
	fn := func() error {
		path := viper.GetString("consul.path") + "/secret"
		kv, _, err := p.client.KV().Get(path, &api.QueryOptions{})
		if err != nil {
			return err
		}

		if kv == nil {
			return errors.New(fmt.Sprintf("key %s not set", path))
		}

		viper.Set("windlass.secret", kv.Value)
		return nil
	}

	count := p.secretGetRetry
	var err error
	for ; count > 0; count-- {
		err = fn()
		if err == nil {
			return nil
		}
		log.WithFields(log.Fields{
			"limit": p.secretGetRetry,
			"count": count,
		}).WithError(err).Error("failed to get shared secret")
		time.Sleep(time.Second * 3)
	}
	return err
}

func (p *Provider) onFailedTTL() error {
	if strings.HasSuffix(p.ttlError.Error(), "does not have associated TTL)") {
		return p.registerWorker()
	}

	return nil
}
