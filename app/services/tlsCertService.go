package services

// Automated TLS cert generation with client and server auth
// Adapted from https://github.com/Shyp/generate-tls-cert/blob/master/generate.go
// with changes to Root Template from https://trac.nginx.org/nginx/ticket/1760 specifically:
// "remove the X509v3 Extended Key Usage extension from the root certificate."

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

const (
	tenYears = time.Hour * 24 * 365 * 10
)

type PEMContainer struct {
	ServerCAPEM, ClientCAPEM, ServerKeyPEM, ServerCertPEM, ClientKeyPEM, ClientCertPEM []byte
}

type TLSCertService struct {
	serverIP           string
	clientCertTemplate *x509.Certificate
	serverCertTemplate *x509.Certificate
}

func NewTLSCertService() *TLSCertService {
	return &TLSCertService{}
}

func (t *TLSCertService) certificateTemplate() *x509.Certificate {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tenYears),
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP(t.serverIP)},
	}
}

func (t *TLSCertService) generateRootParts() (rootKeyPEM []byte, rootCertPEM []byte, rootKeyPart *ecdsa.PrivateKey, rootTemplate *x509.Certificate, err error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rootKeyBytes, err := t.encodeKey(rootKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rootTemplate = &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tenYears),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rootCertBytes, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rootCertBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCertBytes})

	return rootKeyBytes, rootCertBytes, rootKey, rootTemplate, err
}

func (t *TLSCertService) generateServerParts(rootTemplate *x509.Certificate, rootKey *ecdsa.PrivateKey) (serverKeyPEM []byte, serverCertPEM []byte, err error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serverKeyBytes, err := t.encodeKey(serverKey)
	if err != nil {
		return nil, nil, err
	}

	serverTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tenYears),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		IPAddresses:           []net.IP{net.ParseIP(t.serverIP)},
	}

	serverCertBytes, err := x509.CreateCertificate(rand.Reader, &serverTemplate, rootTemplate, &serverKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}

	serverCertBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes})

	return serverKeyBytes, serverCertBytes, nil
}

func (t *TLSCertService) generateClientParts(rootTemplate *x509.Certificate, rootKey *ecdsa.PrivateKey) (clientKeyPEM []byte, clientCertPEM []byte, err error) {
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	clientKeyBytes, err := t.encodeKey(clientKey)
	if err != nil {
		return nil, nil, err
	}

	clientTemplate := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(4),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tenYears),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	clientCertBytes, err := x509.CreateCertificate(rand.Reader, &clientTemplate, rootTemplate, &clientKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}

	clientCertBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertBytes})

	return clientKeyBytes, clientCertBytes, nil
}

func (t *TLSCertService) encodeKey(key *ecdsa.PrivateKey) ([]byte, error) {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), nil
}

func (t *TLSCertService) CreatePEMs(serverIP string) (*PEMContainer, error) {
	t.serverIP = serverIP
	rootKeyPEM, rootCertPEM, rootKey, rootTemplate, err := t.generateRootParts()
	if err != nil {
		return nil, err
	}

	serverKeyPEM, serverCertPEM, err := t.generateServerParts(rootTemplate, rootKey)
	if err != nil {
		return nil, err
	}

	clientKeyPEM, clientCertPEM, err := t.generateClientParts(rootTemplate, rootKey)
	if err != nil {
		return nil, err
	}

	return &PEMContainer{
		ServerCAPEM:   rootKeyPEM,
		ClientCAPEM:   rootCertPEM,
		ServerKeyPEM:  serverKeyPEM,
		ClientKeyPEM:  clientKeyPEM,
		ServerCertPEM: serverCertPEM,
		ClientCertPEM: clientCertPEM,
	}, nil
}
