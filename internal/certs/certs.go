package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
)

// CertificateGroup a bunch of methods to help to work with certificates.
type CertificateGroup struct {
	CSRDerBytes []byte
	key         *ecdsa.PrivateKey

	CSRPem *bytes.Buffer
	KeyPem *bytes.Buffer
}

func NewCertificateGroup(name string) (*CertificateGroup, error) {
	keyCert, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, err
	}

	subj := pkix.Name{
		CommonName:   name,
		Organization: []string{"k4e"},
	}

	csr := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.ECDSAWithSHA1,
	}

	derBytes, err := x509.CreateCertificateRequest(rand.Reader, &csr, keyCert)
	if err != nil {
		return nil, err
	}

	certificateGroup := &CertificateGroup{
		CSRDerBytes: derBytes,
		key:         keyCert,
	}

	certificateGroup.CreatePem()
	return certificateGroup, nil
}

func (c *CertificateGroup) CreatePem() error {
	csrBytes := new(bytes.Buffer)
	err := pem.Encode(csrBytes, &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: c.CSRDerBytes,
	})
	if err != nil {
		return err
	}

	derKey, err := x509.MarshalECPrivateKey(c.key)
	if err != nil {
		return err
	}

	keyBytes := new(bytes.Buffer)
	err = pem.Encode(keyBytes, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derKey})

	if err != nil {
		return err
	}

	c.KeyPem = keyBytes
	c.CSRPem = csrBytes
	return nil
}

func (c *CertificateGroup) Export(certFile, keyFile string) error {
	err := ioutil.WriteFile(certFile, c.CSRPem.Bytes(), 0)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(keyFile, c.KeyPem.Bytes(), 0)
	return err
}
