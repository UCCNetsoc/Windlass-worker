package tlsstorage

import (
	"context"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/providers"

	"github.com/spf13/viper"
)

type vaultTLSStorageRepo struct {
	vault *providers.VaultProvider
}

func NewVaultTLSStorageRepo(vault *providers.VaultProvider) (TLSStorageRepo, error) {
	return &vaultTLSStorageRepo{
		vault: vault,
	}, nil
}

func (v *vaultTLSStorageRepo) PushAuthCerts(ctx context.Context, key string, serverCAPEM, clientCAPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM []byte) error {
	return v.vault.Put(viper.GetString("vault.path")+key, map[string]interface{}{
		"server_ca": serverCAPEM, "client_ca": clientCAPEM, "server_key": serverKeyPEM, "server_cert": serverCertPEM, "client_key": clientKeyPEM, "client_cert": clientCertPEM,
	})
}
