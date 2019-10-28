package connections

import (
	vault "github.com/hashicorp/vault/api"
)

type VaultProvider struct {
	client *vault.Client
}

func newVaultProvider(conf *vault.Config) (*VaultProvider, error) {
	client, err := vault.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return &VaultProvider{
		client: client,
	}, nil
}

func (p *VaultProvider) PushSecrets(pathPrefix string, kv map[string]interface{}) error {
	_, err := p.client.Logical().Write(pathPrefix, kv)
	return err
}

func (p *VaultProvider) GetSecrets(pathPrefix string) (map[string]interface{}, error) {
	s, err := p.client.Logical().Read(pathPrefix)
	if err != nil {
		return nil, err
	}

	return s.Data, nil
}
