/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/oracle/coherence-cli/internal/testtls"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadTLSExpiredCertificateRejected(t *testing.T) {
	resetConnectionRequestState(t)
	ca := testtls.NewCA(t, "expiry-ca")
	expired := ca.IssueExpiredServer(t, localHost)
	server := downloadTLSServer(t, expired, nil)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "roots.pem", ca.PEM))
	optIn := true
	Config = CoherenceCLIConfig{IgnoreInvalidCerts: true, CurrentContext: exceptedConnection,
		Clusters: []ClusterConnection{{Name: exceptedConnection, ConnectionType: httpType,
			ConnectionURL: server.URL, IgnoreInvalidCerts: &optIn}}}
	// The explicitly excepted cluster can use this exact endpoint, but the
	// download must still reject its expired certificate afterwards.
	f, err := GetDataFetcher(exceptedConnection)
	if err != nil {
		t.Fatal(err)
	}
	if body, requestErr := f.GetClusterDetailsJSON(); requestErr != nil || string(body) != "{}" {
		t.Fatalf("explicit exception against expired leaf: %q, %v", body, requestErr)
	}
	_, err = GetURLContents(server.URL)
	// The issuer is trusted and the leaf has the correct IP SAN. Check the
	// precise expiry cause and leaf, rather than accepting any TLS failure.
	var invalid x509.CertificateInvalidError
	if !errors.As(err, &invalid) || invalid.Reason != x509.Expired || invalid.Cert == nil ||
		!bytes.Equal(invalid.Cert.Raw, expired.Certificate[0]) {
		t.Fatalf("expected expired download leaf rejection, got %v", err)
	}
}

func TestDownloadTLSIgnoresClusterExceptions(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	optIn := true
	Config = CoherenceCLIConfig{IgnoreInvalidCerts: true, CurrentContext: exceptedConnection,
		Clusters: []ClusterConnection{{Name: exceptedConnection, ConnectionType: httpType,
			ConnectionURL: server.URL, IgnoreInvalidCerts: &optIn}}}
	f, err := GetDataFetcher(exceptedConnection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("opted-in cluster: %v", err)
	}
	_, err = GetURLContents(server.URL)
	var certificateError *tls.CertificateVerificationError
	if !errors.As(err, &certificateError) {
		t.Fatalf("download inherited a cluster/global exception: %v", err)
	}
}

func TestDownloadTLSConfiguredTrust(t *testing.T) {
	resetConnectionRequestState(t)
	ca := testtls.NewCA(t, "download-ca")
	t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "roots.pem", ca.PEM))
	for _, test := range []struct {
		name, hostname string
		success        bool
	}{
		{"configured CA", localHost, true},
		{"hostname mismatch", "wrong.download.invalid", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := downloadTLSServer(t, ca.Issue(t, test.hostname, false), nil)
			body, err := GetURLContents(server.URL)
			if test.success {
				if err != nil || string(body) != "{}" {
					t.Fatalf("configured CA download: %q, %v", body, err)
				}
			} else {
				var hostnameError x509.HostnameError
				if !errors.As(err, &hostnameError) {
					t.Fatalf("expected hostname rejection: %v", err)
				}
			}
		})
	}
}

func TestDownloadTLSClientCertificate(t *testing.T) {
	resetConnectionRequestState(t)
	ca, clientCA := testtls.NewCA(t, "download-ca"), testtls.NewCA(t, "client-ca")
	clientRoots := x509.NewCertPool()
	clientRoots.AddCert(clientCA.Certificate)
	server := downloadTLSServer(t, ca.Issue(t, localHost, false), clientRoots)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "roots.pem", ca.PEM))
	if _, err := GetURLContents(server.URL); err == nil {
		t.Fatal("server requiring a client certificate accepted none")
	}
	client := clientCA.Issue(t, "download-client", true)
	key, err := x509.MarshalPKCS8PrivateKey(client.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("COHERENCE_TLS_CLIENT_CERT", writeDownloadFixture(t, "client.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: client.Certificate[0]})))
	t.Setenv("COHERENCE_TLS_CLIENT_KEY", writeDownloadFixture(t, "key.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})))
	body, err := GetURLContents(server.URL)
	if err != nil || string(body) != "{}" {
		t.Fatalf("verified mutual TLS download: %q, %v", body, err)
	}
	t.Setenv("COHERENCE_TLS_CLIENT_KEY", writeDownloadFixture(t, "bad-key.pem", []byte("invalid key")))
	if _, err = GetURLContents(server.URL); err == nil {
		t.Fatal("invalid client key was ignored")
	}
}

func TestDownloadHTTPRedirectAndTimeout(t *testing.T) {
	resetConnectionRequestState(t)
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Store(true)
		_, _ = io.WriteString(w, "redirect destination")
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.SetCookie(w, &http.Cookie{Name: "download-cookie", Value: "cookie", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, target.URL, http.StatusFound)
		case "/timeout":
			<-r.Context().Done()
		default:
			_, _ = io.WriteString(w, "plain HTTP")
		}
	}))
	defer server.Close()
	body, err := GetURLContents(server.URL)
	if err != nil || string(body) != "plain HTTP" {
		t.Fatalf("HTTP download: %q, %v", body, err)
	}
	if _, err = GetURLContents(server.URL + "/redirect"); err == nil || !strings.Contains(err.Error(), "302") || redirected.Load() {
		t.Fatalf("redirect policy changed: %v, destination reached %v", err, redirected.Load())
	}
	fetcher.RequestTimeout = 1
	started := time.Now()
	_, err = GetURLContents(server.URL + "/timeout")
	var timeoutError net.Error
	if !errors.As(err, &timeoutError) || !timeoutError.Timeout() || time.Since(started) > 5*time.Second {
		t.Fatalf("request timeout not preserved: %v, elapsed %v", err, time.Since(started))
	}
}

func TestDownloadTLSInvalidCAFailsBeforeRequest(t *testing.T) {
	resetConnectionRequestState(t)
	var reached atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { reached.Store(true) }))
	defer server.Close()
	for _, file := range []string{filepath.Join(t.TempDir(), "missing.pem"), t.TempDir(), writeDownloadFixture(t, "bad.pem", []byte("invalid CA"))} {
		t.Setenv("COHERENCE_TLS_CERTS_PATH", file)
		if _, err := GetURLContents(server.URL); err == nil || reached.Load() {
			t.Fatalf("invalid CA did not fail before request: %v, reached %v", err, reached.Load())
		}
	}
}

func downloadTLSServer(t *testing.T, certificate tls.Certificate, clientRoots *x509.CertPool) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clientRoots != nil && len(r.TLS.VerifiedChains) == 0 {
			t.Error("client certificate was not verified")
		}
		_, _ = io.WriteString(w, "{}")
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}, ClientCAs: clientRoots}
	if clientRoots != nil {
		server.TLS.ClientAuth = tls.RequireAndVerifyClientCert
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func writeDownloadFixture(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
