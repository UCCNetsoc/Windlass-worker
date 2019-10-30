package tlsstorage

import (
	"context"
	"fmt"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/providers"
	"github.com/spf13/viper"
)

type TLSStorageRepo interface {
	PushAuthCerts(ctx context.Context, key string, serverCAPEM, clientCAPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM []byte) error
}

func NewTLSStorageRepo() TLSStorageRepo {
	if viper.GetBool("vault.enabled") {
		vault, err := providers.NewVaultProvider()
		if err != nil {
			panic(fmt.Errorf("failed to create Vault client: %w", err))
		}

		repo, err := NewVaultTLSStorageRepo(vault)
		if err != nil {
			panic(fmt.Errorf("failed to get TLS storage repo: %w", err))
		}
		return repo
	} else {
		// TODO: consul
	}
	panic("vault currently required")
}
