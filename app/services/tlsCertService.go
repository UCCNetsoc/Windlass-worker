package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

type PEMContainer struct {
	CAPEM, ServerKeyPEM, ServerCertPEM, ClientKeyPEM, ClientCertPEM []byte
}

type TLSCertService struct {
	serverIP           string
	clientCertTemplate *x509.Certificate
	serverCertTemplate *x509.Certificate
}

func NewTLSCertService(serverIP string) *TLSCertService {
	return &TLSCertService{serverIP: serverIP}
}

func (t *TLSCertService) certificateTemplate() *x509.Certificate {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(((time.Hour) * 365) * 10),
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP(t.serverIP)},
	}
}

func (t *TLSCertService) generateCertTemplates() {
	certTemplate := t.certificateTemplate()

	serverCertTemplate := *certTemplate
	serverCertTemplate.IsCA = true
	serverCertTemplate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	serverCertTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	t.serverCertTemplate = &serverCertTemplate

	clientCertTemplate := *certTemplate
	clientCertTemplate.KeyUsage = x509.KeyUsageDigitalSignature
	clientCertTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	t.clientCertTemplate = &clientCertTemplate
}

func (t *TLSCertService) generateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}

func (t *TLSCertService) generateKeyPEM() ([]byte, *rsa.PrivateKey, error) {
	serverKey, err := t.generateKey()
	if err != nil {
		return nil, nil, err
	}
	serverKeyBlock := pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)}
	return pem.EncodeToMemory(&serverKeyBlock), serverKey, nil
}

func (t *TLSCertService) generateCertPEM(template, parent *x509.Certificate, pub, priv interface{}) ([]byte, error) {
	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, err
	}
	caCertBlock := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	return pem.EncodeToMemory(&caCertBlock), nil
}

func (t *TLSCertService) generateTLSCertPEM(certPEM, keyPEM []byte) ([]byte, error) {
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	tlsCertBlock := pem.Block{Type: "CERTIFICATE", Bytes: tlsCert.Certificate[0]}
	return pem.EncodeToMemory(&tlsCertBlock), nil
}

func (t *TLSCertService) CreatePEMs() (*PEMContainer, error) {
	t.generateCertTemplates()

	// Create Server Key PEM file for server side
	serverKeyPEM, serverKey, err := t.generateKeyPEM()
	if err != nil {
		return nil, err
	}

	// Create CA PEM file for client and server side
	caCertPEM, err := t.generateCertPEM(t.serverCertTemplate, t.serverCertTemplate, &serverKey.PublicKey, serverKey)
	if err != nil {
		return nil, err
	}

	// Create Server Cert PEM file for server side
	serverCertPEM, err := t.generateTLSCertPEM(caCertPEM, serverKeyPEM)
	if err != nil {
		return nil, err
	}

	// Create Client Key PEM file for client side
	clientKeyPEM, clientKey, err := t.generateKeyPEM()
	if err != nil {
		return nil, err
	}

	clientCACertPEM, err := t.generateCertPEM(t.clientCertTemplate, t.serverCertTemplate, &clientKey.PublicKey, serverKey)
	if err != nil {
		return nil, err
	}

	clientCertPEM, err := t.generateTLSCertPEM(clientCACertPEM, clientKeyPEM)
	if err != nil {
		return nil, err
	}

	return &PEMContainer{caCertPEM, serverKeyPEM, serverCertPEM, clientKeyPEM, clientCertPEM}, nil
}
