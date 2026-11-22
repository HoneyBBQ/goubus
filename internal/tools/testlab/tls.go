package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func runTLS(args []string) error {
	fs := flag.NewFlagSet("tls", flag.ContinueOnError)
	dir := fs.String("dir", "", "output directory for ca.pem, server.pem, server.key")
	err := fs.Parse(args)
	if err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("%w: -dir", errMissingFlag)
	}

	caCertPEM, serverCertPEM, serverKeyPEM, err := generateCertBundle()
	if err != nil {
		return err
	}

	return writeCertBundle(*dir, caCertPEM, serverCertPEM, serverKeyPEM)
}

func generateCertBundle() ([]byte, []byte, []byte, error) {
	now := time.Now()
	caKey, caTemplate, err := buildCATemplate(now)
	if err != nil {
		return nil, nil, nil, err
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, nil, nil, err
	}

	serverKey, serverTemplate, err := buildServerTemplate(now, caCert.SubjectKeyId)
	if err != nil {
		return nil, nil, nil, err
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		return nil, nil, nil, err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return caPEM, serverPEM, keyPEM, nil
}

func buildCATemplate(now time.Time) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&caKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	ski := sha256.Sum256(pubDER)

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "goubus live test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(48 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SubjectKeyId:          ski[:],
	}

	return caKey, tmpl, nil
}

func buildServerTemplate(now time.Time, authorityKeyID []byte) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber:   serial,
		Subject:        pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:      now.Add(-time.Hour),
		NotAfter:       now.Add(48 * time.Hour),
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:       []string{"localhost"},
		IPAddresses:    []net.IP{net.IPv4(127, 0, 0, 1)},
		AuthorityKeyId: authorityKeyID,
	}

	return serverKey, tmpl, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)

	return rand.Int(rand.Reader, limit)
}

func writeCertBundle(dir string, caPEM, serverPEM, keyPEM []byte) error {
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}

	writeErr := errors.Join(
		root.WriteFile("ca.pem", caPEM, 0o600),
		root.WriteFile("server.pem", serverPEM, 0o600),
		root.WriteFile("server.key", keyPEM, 0o600),
	)

	return errors.Join(writeErr, root.Close())
}
