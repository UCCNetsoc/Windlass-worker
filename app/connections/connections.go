package connections

import (
	consul "github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"

	lxd "github.com/lxc/lxd/client"

	"github.com/Strum355/log"
	"github.com/spf13/viper"
)

type Connections struct {
	lxd    lxd.ContainerServer
	consul *ConsulProvider
	vault  *VaultProvider
}

var group Connections

func EstablishConnections() error {
	var err error

	if _, err = GetConsul(); err != nil {
		return err
	}

	if _, err = GetVault(); err != nil {
		return err
	}

	if err := group.consul.GetAndSetSharedSecret(); err != nil {
		return err
	}

	if _, err = GetLXD(); err != nil {
		return err
	}

	log.Debug("connections established")
	return nil
}

func Close() {

}

func GetVault() (*VaultProvider, error) {
	if group.vault != nil {
		return group.vault, nil
	}

	config := vault.Config{
		Address: viper.GetString("vault.url"),
	}

	provider, err := newVaultProvider(&config)
	if err != nil {
		return nil, err
	}

	provider.client.SetToken(viper.GetString("vault.token"))

	return provider, nil
}

func GetConsul() (*ConsulProvider, error) {
	if group.consul != nil {
		return group.consul, nil
	}

	config := consul.Config{
		Address: viper.GetString("consul.url"),
		Token:   viper.GetString("consul.token"),
	}

	provider, err := newConsulProvider(&config)
	if err != nil {
		return nil, NewConnectionError(err, "Consul")
	}

	if err := provider.Register(); err != nil {
		return nil, NewConnectionError(err, "Consul")
	}

	group.consul = provider

	return group.consul, nil
}

func GetLXD() (lxd.ContainerServer, error) {
	if group.lxd != nil {
		return group.lxd, nil
	}

	lxdConn, err := lxd.ConnectLXDUnix(viper.GetString("lxd.socket"), &lxd.ConnectionArgs{
		UserAgent: "Windlass",
	})
	if err != nil {
		return nil, NewConnectionError(err, "LXD")
	}

	group.lxd = lxdConn

	return lxdConn, nil
}
