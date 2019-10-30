package connections

import (
	"github.com/Strum355/log"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/repositories/providers"
)

func TestConnections() error {
	var err error

	if err = testConsul(); err != nil {
		return err
	}

	if err = testVault(); err != nil {
		return err
	}

	log.Debug("connections tested successfully")
	return nil
}

func testVault() error {
	_, err := providers.NewVaultProvider()
	return err
}

func testConsul() error {
	p, err := providers.NewConsulProvider()
	if err != nil {
		return err
	}

	if err := p.Register(); err != nil {
		return err
	}
	return p.GetAndSetSharedSecret()
}
