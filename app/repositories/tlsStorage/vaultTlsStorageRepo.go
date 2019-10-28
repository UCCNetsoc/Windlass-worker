package tlsstorage

import (
	"context"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"
	"github.com/spf13/viper"
)

type VaultTLSStorageRepo struct {
	vault *connections.VaultProvider
}

func NewVaultTLSStorageRepo() (*VaultTLSStorageRepo, error) {
	vault, err := connections.GetVault()
	if err != nil {
		return nil, err
	}

	return &VaultTLSStorageRepo{
		vault: vault,
	}, nil
}

func (v *VaultTLSStorageRepo) PushAuthCerts(ctx context.Context, key string, caPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM []byte) error {
	return v.vault.PushSecrets(viper.GetString("vault.path")+key, map[string]interface{}{
		"ca_cert": caPEM, "server_key": serverKeyPEM, "server_cert": serverCertPEM, "client_key": clientKeyPEM, "client_cert": clientCertPEM,
	})
}
