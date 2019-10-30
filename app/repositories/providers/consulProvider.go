package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	host "github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/containerHost"

	"github.com/Strum355/log"

	"github.com/spf13/viper"

	consul "github.com/hashicorp/consul/api"
)

type ConsulProvider struct {
	client           *consul.Client
	ttl              time.Duration
	mu               *sync.Mutex
	secretGetRetry   int
	projectIDtoCheck map[string]projectCheck
}

type projectCheck struct {
	check  func(ip string) (string, bool)
	ticker *time.Ticker
}

type projectMeta struct {
	ID string `json:"id"`
	IP string `json:"ip_address"`
}

func NewConsulProvider() (*ConsulProvider, error) {
	config := consul.Config{
		Address: viper.GetString("consul.url"),
		Token:   viper.GetString("consul.token"),
	}

	client, err := consul.NewClient(&config)
	if err != nil {
		return nil, err
	}

	return &ConsulProvider{
		client:           client,
		ttl:              time.Second * 10,
		secretGetRetry:   5,
		projectIDtoCheck: make(map[string]projectCheck),
		mu:               new(sync.Mutex),
	}, nil
}

func (p *ConsulProvider) id() string {
	return fmt.Sprintf("windlass-worker@%s:%s", viper.GetString("http.hostname"), strconv.Itoa(viper.GetInt("http.port")))
}

func (p *ConsulProvider) address() string {
	return viper.GetString("http.address")
}

func (p *ConsulProvider) port() int {
	return viper.GetInt("http.port")
}

// Register registers the worker and its associated projects with Consul.
// See registerWorker() and registerProjects() for specific details
func (p *ConsulProvider) Register() error {
	if err := p.registerWorker(); err != nil {
		return fmt.Errorf("failed to register worker: %v", err)
	}

	if err := p.registerProjects(); err != nil {
		return fmt.Errorf("failed to register one or more associated projects: %v", err)
	}

	p.udpateWorkerTTL()

	return nil
}

// Registers this worker with Consul using a TTL based health check
func (p *ConsulProvider) registerWorker() error {
	service := &consul.AgentServiceRegistration{
		ID:      p.id(),
		Name:    "windlass_worker",
		Address: p.address(),
		Port:    p.port(),
		Check: &consul.AgentServiceCheck{
			// Probably wanna keep failed workers around
			// TODO: Need to purposely deregister on successful shutdown then
			//DeregisterCriticalServiceAfter: p.deregisterCritical.String(),
			TTL: p.ttl.String(),
		},
	}

	return p.client.Agent().ServiceRegister(service)
}

// Registers the projects associated with this worker with Consul using a TTL based health check
// It pings the Docker Daemon on each project, see https://docs.docker.com/engine/api/v1.37/#operation/SystemPing for details
func (p *ConsulProvider) registerProjects() error {
	pairs, _, err := p.client.KV().List(p.kvPath(), &consul.QueryOptions{})
	if err != nil {
		return errors.New(fmt.Sprintf("failed to load KV at path %s", p.kvPath()))
	}

	for _, pair := range pairs {
		var meta projectMeta
		err := json.Unmarshal(pair.Value, &meta)
		if err != nil {
			return err
		}

		containerHost := host.NewContainerHostRepository()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		projectName := p.projectNameFromKVPath(pair.Key)
		ip, err := containerHost.GetContainerHostIP(ctx, projectName)
		if err != nil {
			return errors.New(fmt.Sprintf("failed to get container IP for host %s: %v", projectName, err))
		}

		vault, err := NewVaultProvider()
		if err != nil {
			return err
		}

		tls, err := vault.Get(viper.GetString("vault.path") + meta.ID)
		if err != nil {
			return err
		}

		clientKey, _ := base64.StdEncoding.DecodeString(tls["client_key"].(string))
		clientCert, _ := base64.StdEncoding.DecodeString(tls["client_cert"].(string))
		clientCA, _ := base64.StdEncoding.DecodeString(tls["client_ca"].(string))

		containerHost.UseCerts([]byte(clientKey), []byte(clientCert), []byte(clientCA))

		p.RegisterProject(p.projectNameFromKVPath(pair.Key), ip, func(ip string) (string, bool) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := containerHost.Ping(ctx)
			if err != nil {
				return err.Error(), false
			}
			return "Remote Docker daemon reachable", true
		})
	}

	return nil
}

// RegisterProject registers a single project
func (p *ConsulProvider) RegisterProject(projectName string, ip string, check func(ip string) (string, bool)) error {
	projectM := projectMeta{ID: projectName, IP: ip}

	projectService := &consul.AgentServiceRegistration{
		ID:      projectName,
		Name:    "windlass_worker_projects",
		Tags:    []string{fmt.Sprintf("Worker:%s", p.kvPath())},
		Address: ip,
		Port:    443,
		Check: &consul.AgentServiceCheck{
			TTL: (p.ttl * 5).String(),
		},
	}

	err := p.client.Agent().ServiceRegister(projectService)
	if err != nil {
		return err
	}

	err = p.SaveProjectMeta(projectM)
	if err != nil {
		return err
	}

	p.updateProjectTTL(projectName, ip, check)

	return nil
}

// Updates the TTL for this worker
func (p *ConsulProvider) udpateWorkerTTL() {
	go func() {
		ticker := time.NewTicker(p.ttl / 2)
		for range ticker.C {
			health := consul.HealthPassing
			if viper.GetString("windlass.secret") == "" {
				health = consul.HealthCritical
			}

			err := p.client.Agent().UpdateTTL("service:"+p.id(), "", health)
			if err != nil {
				p.onFailedWorkerTTL(err)
				log.WithError(err).Error("failed to update TTL")
			}
		}
	}()
}

// Updates the TTL for each project associated with this worker
// runs `check` on each tick which returns a string for health message and a bool true if healthy and false if not
func (p *ConsulProvider) updateProjectTTL(id, ip string, check func(ip string) (string, bool)) {
	ticker := time.NewTicker((p.ttl * 5) / 2)
	p.projectIDtoCheck[id] = projectCheck{
		check: check, ticker: ticker,
	}

	go func() {
		for range ticker.C {
			health := consul.HealthPassing
			msg, healthy := check(ip)
			log.WithFields(log.Fields{
				"msg":     msg,
				"healthy": healthy,
			}).Info("docker daemon health check ping response")
			if !healthy {
				health = consul.HealthCritical
			}
			err := p.client.Agent().UpdateTTL("service:"+id, msg, health)
			if err != nil {

			}
		}
	}()
}

func (p *ConsulProvider) GetAndSetSharedSecret() error {
	fn := func() error {
		path := viper.GetString("consul.path") + "/secret"
		kv, _, err := p.client.KV().Get(path, &consul.QueryOptions{})
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

func (p *ConsulProvider) SaveProjectMeta(projectMetadata projectMeta) error {
	b, err := json.Marshal(projectMetadata)
	if err != nil {
		return err
	}

	p.mu.Lock()
	_, err = p.client.KV().Put(&consul.KVPair{
		Key:   fmt.Sprintf("%s/%s", p.kvPath(), projectMetadata.ID),
		Value: b,
	}, &consul.WriteOptions{})
	return err
}

func (p *ConsulProvider) kvPath() string {
	return fmt.Sprintf("windlass_worker@%s", viper.GetString("http.hostname"))
}

func (p *ConsulProvider) onFailedWorkerTTL(err error) error {
	if strings.HasSuffix(err.Error(), "does not have associated TTL)") {
		return p.registerWorker()
	}

	return nil
}

func (p *ConsulProvider) onFailedProjectTTL(id, ip string, err error) error {
	if strings.HasPrefix(err.Error(), "does not have associated TTL") {
		return p.RegisterProject(id, ip, p.projectIDtoCheck[id].check)
	}
	return nil
}

func (p *ConsulProvider) projectNameFromKVPath(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, p.kvPath()+"/"), "/")
}
