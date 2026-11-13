/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

// Package testtls provides independent, short-lived CA chains for local TLS tests.
package testtls

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"
)

// CA is a test-only certificate authority.
type CA struct {
	Certificate *x509.Certificate
	PEM         []byte
	key         ed25519.PrivateKey
}

// NewCA creates a new independent authority.
func NewCA(t *testing.T, name string) CA {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, public, private)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err = x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return CA{Certificate: certificate, PEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), key: private}
}

// Issue creates a leaf certificate, using a DNS/IP SAN for servers.
func (ca CA) Issue(t *testing.T, name string, client bool) tls.Certificate {
	t.Helper()
	return ca.issue(t, name, client, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
}

// IssueExpiredServer creates an expired server leaf under the still-valid CA.
func (ca CA) IssueExpiredServer(t *testing.T, name string) tls.Certificate {
	t.Helper()
	return ca.issue(t, name, false, time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))
}

func (ca CA) issue(t *testing.T, name string, client bool, notBefore, notAfter time.Time) tls.Certificate {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: name},
		NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if client {
		certificate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else if ip := net.ParseIP(name); ip != nil {
		certificate.IPAddresses = []net.IP{ip}
	} else {
		certificate.DNSNames = []string{name}
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, ca.Certificate, public, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: private}
}
