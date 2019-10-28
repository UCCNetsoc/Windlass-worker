package tlsstorage

import (
	"context"
)

type TLSStorageRepo interface {
	PushAuthCerts(ctx context.Context, key string, caPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM []byte) error
}
