package providers

import (
	"errors"

	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
)

type VaultProvider struct {
	client *vault.Client
}

func NewVaultProvider() (*VaultProvider, error) {
	config := vault.Config{
		Address: viper.GetString("vault.url"),
	}
	client, err := vault.NewClient(&config)
	if err != nil {
		return nil, err
	}

	client.SetToken(viper.GetString("vault.token"))

	return &VaultProvider{
		client: client,
	}, nil
}

func (p *VaultProvider) Put(pathPrefix string, kv map[string]interface{}) error {
	_, err := p.client.Logical().Write(pathPrefix, kv)
	return err
}

func (p *VaultProvider) Get(pathPrefix string) (map[string]interface{}, error) {
	s, err := p.client.Logical().Read(pathPrefix)
	if err != nil {
		return nil, err
	}

	if s == nil {
		return nil, errors.New("nothing found in Vault at given prefix")
	}

	return s.Data, nil
}
