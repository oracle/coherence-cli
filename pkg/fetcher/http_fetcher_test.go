/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oracle/coherence-cli/internal/testtls"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/utils"
	"go.uber.org/zap"
)

const (
	testHTTPSOrigin    = "https://example.test"
	invalidHTTPSOrigin = "https://[invalid]"
	certificatePEMType = "CERTIFICATE"
)

func TestConnectionTLSOrigins(t *testing.T) {
	for _, test := range []struct {
		name, allowed, destination string
		want                       bool
	}{
		{"same", "https://example.test/base", "https://example.test/other?q=1", true},
		{"default port", testHTTPSOrigin, "https://example.test:443/path", true},
		{"explicit default", "https://example.test:443", testHTTPSOrigin, true},
		{"case", "HTTPS://EXAMPLE.TEST:443/base", "https://example.test/", true},
		{"userinfo", "https://user:secret@example.test", testHTTPSOrigin, true},
		{"ipv6", "https://[2001:DB8::1]", "https://[2001:db8:0:0:0:0:0:1]:443/a", true},
		{"ipv6 port", "https://[::1]:8443", "https://[::1]:8443/a", true},
		{"ipv6 different port", "https://[::1]:8443", "https://[::1]:443/a", false},
		{"different port", testHTTPSOrigin, "https://example.test:8443", false},
		{"different host", testHTTPSOrigin, "https://other.test", false},
		{"prefix host", testHTTPSOrigin, "https://example.test.evil.test", false},
		{"prefix credentials", testHTTPSOrigin, "https://example.test@evil.test", false},
		{"no DNS aliases", "https://localhost", "https://127.0.0.1", false},
		{"HTTP destination", testHTTPSOrigin, "http://example.test:443", false},
		{"HTTP base", "http://example.test", testHTTPSOrigin, false},
		{"empty base", "", testHTTPSOrigin, false},
		{"opaque base", "https:example.test", testHTTPSOrigin, false},
		{"opaque destination", testHTTPSOrigin, "https:example.test", false},
		{"missing host", "https://", testHTTPSOrigin, false},
		{"missing destination host", testHTTPSOrigin, "https:///a", false},
		{"relative", testHTTPSOrigin, "//example.test/a", false},
		{"malformed", "https://[::1", testHTTPSOrigin, false},
		{"bad destination", testHTTPSOrigin, "https://example.test:bad", false},
		{"bad base port", "https://example.test:bad", testHTTPSOrigin, false},
		{"empty port", "https://example.test:", testHTTPSOrigin, false},
		{"invalid IPv6", invalidHTTPSOrigin, invalidHTTPSOrigin, false},
		{"unbracketed IPv6", "https://2001:db8::1:443", "https://[2001:db8::1]:443", false},
		{"bracketed IPv4", "https://[127.0.0.1]", "https://127.0.0.1", false},
		{"out of range", "https://example.test:65536", "https://example.test:65536", false},
		{"zero port", "https://example.test:0", "https://example.test:0", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := HTTPFetcher{insecureOrigin: parseHTTPSOrigin(test.allowed)}
			u, err := url.Parse(test.destination)
			if err != nil {
				u = nil
			}
			if got := h.permitsInvalidCertificate(u); got != test.want {
				t.Fatalf("origin %q permits %q = %v, want %v", test.allowed, test.destination, got, test.want)
			}
		})
	}
}

func TestConnectionTLSRequests(t *testing.T) {
	resetRequestTestState(t)
	a, b := requestTestServer(t, true), requestTestServer(t, true)
	secure, err := GetFetcherOrError(HTTP, a.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = secure.GetClusterDetailsJSON(); err == nil {
		t.Fatal("default constructor accepted an untrusted certificate")
	}
	h := connectionTestFetcher(t, a.URL, true)
	if h.connectionName != "fixture" {
		t.Fatal("constructor did not retain the connection name")
	}
	if _, err = h.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("opted-in same-origin request: %v", err)
	}
	if _, err = httpGetRequestAbsolute(h, a.URL+"/absolute"); err != nil {
		t.Fatalf("same-origin absolute request: %v", err)
	}
	// Both servers have the same loopback hostname but distinct ports.
	_, err = httpGetRequestAbsolute(h, b.URL+"/absolute")
	requireCertificateRejection(t, err)
	_, err = httpGetRequestAbsolute(h, strings.Replace(a.URL, "127.0.0.1", "localhost", 1))
	requireCertificateRejection(t, err)
	if _, err = httpGetRequestAbsolute(h, "https://[::1"); err == nil {
		t.Fatal("malformed request URL accepted")
	}
	if status, _ := h.GetResponseCodeAndNodeID(b.URL + "/live"); status != "Refused" {
		t.Fatalf("different-origin health request = %s", status)
	}
	if status, node := h.GetResponseCodeAndNodeID(a.URL + "/live"); status != "200" || node != "7" {
		t.Fatalf("same-origin health = %s, %s", status, node)
	}
	// The empty-base health helper and direct zero-policy values remain secure.
	empty, err := GetFetcherOrError(HTTP, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if status, _ := empty.GetResponseCodeAndNodeID(a.URL + "/ready"); status != "Refused" {
		t.Fatalf("empty-base health constructor = %s", status)
	}
	_, err = httpGetRequest(HTTPFetcher{URL: a.URL}, "/")
	requireCertificateRejection(t, err)
	// Changing the public base URL cannot retarget a construction-bound exception.
	h.URL = b.URL
	_, err = httpGetRequest(h, "/")
	requireCertificateRejection(t, err)
	for _, base := range []string{"", "https:opaque", "https://", invalidHTTPSOrigin, "http://127.0.0.1"} {
		invalid := connectionTestFetcher(t, base, true)
		_, err = httpGetRequestAbsolute(invalid, a.URL)
		requireCertificateRejection(t, err)
	}
	if _, err = GetFetcherOrError("invalid", a.URL, "", ""); err == nil {
		t.Fatal("invalid connection type accepted")
	}
}

func TestConnectionTLSExpiredCertificateRejected(t *testing.T) {
	resetRequestTestState(t)
	ca := testtls.NewCA(t, "expiry-ca")
	expired := ca.IssueExpiredServer(t, "127.0.0.1")
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "{}")
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{expired}}
	server.StartTLS()
	defer server.Close()
	caFile := filepath.Join(t.TempDir(), "roots.pem")
	writeTLSFixture(t, caFile, ca.PEM)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", caFile)
	defaultFetcher, err := GetFetcherOrError(HTTP, server.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	verified := connectionTestFetcher(t, server.URL, false)
	for _, h := range []HTTPFetcher{defaultFetcher.(HTTPFetcher), verified} {
		// Use the common request path before the public management API formats
		// the error. The valid issuing CA and matching IP SAN isolate expiry.
		_, err = httpGetRequest(h, "/")
		var invalid x509.CertificateInvalidError
		if !errors.As(err, &invalid) || invalid.Reason != x509.Expired || invalid.Cert == nil ||
			!bytes.Equal(invalid.Cert.Raw, expired.Certificate[0]) {
			t.Fatalf("expected expired cluster leaf rejection, got %v", err)
		}
	}
}

func TestConnectionTLSIsolation(t *testing.T) {
	resetRequestTestState(t)
	a, b := requestTestServer(t, true), requestTestServer(t, true)
	allowed := connectionTestFetcher(t, a.URL, true)
	verified := connectionTestFetcher(t, b.URL, false)
	// Construction is sequential: Init still writes existing package TLS globals.
	// Requests here do not mutate those globals or use the global credentials.
	request := func(h HTTPFetcher, destination string, wantSuccess bool) error {
		_, err := httpGetRequestAbsolute(h, destination)
		if wantSuccess && err != nil {
			return err
		}
		if !wantSuccess {
			var certificateError *tls.CertificateVerificationError
			if !errors.As(err, &certificateError) {
				return fmt.Errorf("expected certificate rejection, got %v", err)
			}
		}
		return nil
	}
	for range 3 {
		if err := request(allowed, a.URL, true); err != nil {
			t.Fatal(err)
		}
		if err := request(verified, b.URL, false); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for range 8 {
		for _, test := range []struct {
			h       HTTPFetcher
			url     string
			success bool
		}{{allowed, a.URL, true}, {verified, b.URL, false}, {allowed, b.URL, false}} {
			wg.Go(func() {
				if err := request(test.h, test.url, test.success); err != nil {
					failures <- err
				}
			})
		}
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestConnectionTLSCoherenceWebLinks(t *testing.T) {
	resetRequestTestState(t)
	other := requestTestServer(t, true)
	for _, outside := range []bool{false, true} {
		t.Run(fmt.Sprintf("other-origin=%v", outside), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/webApplications" {
					link := "https://" + r.Host + "/webapp"
					if outside {
						link = other.URL + "/webapp"
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(config.ItemLinks{Links: []config.Links{{Links: []config.Link{{Rel: "self", Href: link}}}}})
					return
				}
				_, _ = io.WriteString(w, `{"items":[{"nodeId":7}]}`)
			}))
			defer server.Close()
			h := connectionTestFetcher(t, server.URL, true)
			result, err := h.GetHTTPSessionDetailsJSON()
			if outside {
				if err == nil || !strings.Contains(err.Error(), "certificate") {
					t.Fatalf("Coherence*Web absolute link inherited exception: %v", err)
				}
			} else if err != nil || !strings.Contains(string(result), `"nodeId":7`) {
				t.Fatalf("same-origin Coherence*Web link failed: %s, %v", result, err)
			}
		})
	}
}

func TestConnectionTLSHTTPAndRedirects(t *testing.T) {
	resetRequestTestState(t)
	server := requestTestServer(t, false)
	for _, enabled := range []bool{false, true} {
		h := connectionTestFetcher(t, server.URL, enabled)
		if _, err := h.GetClusterDetailsJSON(); err != nil {
			t.Fatalf("HTTP request: %v", err)
		}
		if h.insecureOrigin != (httpsOrigin{}) {
			t.Fatal("HTTP acquired a TLS exception")
		}
	}
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		followed.Add(1)
		_, _ = io.WriteString(w, "{}")
	}))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()
	h := connectionTestFetcher(t, redirect.URL, true)
	if _, err := h.GetClusterDetailsJSON(); err == nil || !strings.Contains(err.Error(), "302") || followed.Load() != 0 {
		t.Fatalf("redirect behavior changed: %v, followed %d", err, followed.Load())
	}
}

func TestConnectionTLSTrustAndClientCertificate(t *testing.T) {
	resetRequestTestState(t)
	server := requestTestServer(t, true)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	writeTLSFixture(t, caFile, pem.EncodeToMemory(&pem.Block{Type: certificatePEMType, Bytes: server.Certificate().Raw}))
	t.Setenv("COHERENCE_TLS_CERTS_PATH", caFile)
	h := connectionTestFetcher(t, server.URL, false)
	if _, err := h.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("trusted CA with verification: %v", err)
	}
	// This certificate has the IP SAN, but no localhost SAN. Trusting it must
	// still enforce hostnames rather than treating a custom CA as an exception.
	_, err := httpGetRequestAbsolute(h, strings.Replace(server.URL, "127.0.0.1", "localhost", 1))
	requireCertificateRejection(t, err)

	clientCert, clientKey := requestTestClientCertificate(t)
	clientPool := x509.NewCertPool()
	clientPool.AppendCertsFromPEM(clientCert)
	var verifiedClient atomic.Bool
	mutual := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifiedClient.Store(len(r.TLS.VerifiedChains) > 0)
		_, _ = io.WriteString(w, "{}")
	}))
	mutual.Config.ErrorLog = log.New(io.Discard, "", 0)
	mutual.TLS = &tls.Config{MinVersion: tls.VersionTLS12, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientPool}
	mutual.StartTLS()
	defer mutual.Close()
	writeTLSFixture(t, caFile, pem.EncodeToMemory(&pem.Block{Type: certificatePEMType, Bytes: mutual.Certificate().Raw}))
	h = connectionTestFetcher(t, mutual.URL, false)
	if _, err = h.GetClusterDetailsJSON(); err == nil {
		t.Fatal("mutual TLS accepted a missing client certificate")
	}
	clientFile, keyFile := filepath.Join(t.TempDir(), "client.pem"), filepath.Join(t.TempDir(), "client-key.pem")
	writeTLSFixture(t, clientFile, clientCert)
	writeTLSFixture(t, keyFile, clientKey)
	t.Setenv("COHERENCE_TLS_CLIENT_CERT", clientFile)
	t.Setenv("COHERENCE_TLS_CLIENT_KEY", keyFile)
	h = connectionTestFetcher(t, mutual.URL, false)
	if _, err = h.GetClusterDetailsJSON(); err != nil || !verifiedClient.Load() {
		t.Fatalf("verified mutual TLS: %v, verified client %v", err, verifiedClient.Load())
	}
}

func TestConnectionTLSWebLogicAuthentication(t *testing.T) {
	resetRequestTestState(t)
	username, password = "fixture-user", "fixture-password"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password || r.Header.Get("X-Requested-By") != "Coherence-CLI" ||
			r.URL.Path != "/management/coherence/latest/clusters/fixture/" {
			t.Error("WebLogic path or authentication changed")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()
	f, err := GetFetcherForConnection(HTTP, server.URL+"/management/coherence/latest/clusters", "", "fixture", "fixture", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("WebLogic authenticated TLS request: %v", err)
	}
}

func resetRequestTestState(t *testing.T) {
	t.Helper()
	oldCertificates, oldPool := certificates, certPool
	oldCA, oldCert, oldKey := caCertPath, clientCertPath, clientCertKeyPath
	oldUsername, oldPassword, oldDebug, oldTimeout := username, password, DebugEnabled, RequestTimeout
	oldLogger := utils.Logger
	t.Cleanup(func() {
		certificates, certPool = oldCertificates, oldPool
		caCertPath, clientCertPath, clientCertKeyPath = oldCA, oldCert, oldKey
		username, password, DebugEnabled, RequestTimeout = oldUsername, oldPassword, oldDebug, oldTimeout
		utils.Logger = oldLogger
	})
	username, password, DebugEnabled, RequestTimeout = "", "", false, 5
	utils.Logger = zap.NewNop()
	for _, key := range []string{"COHERENCE_TLS_CERTS_PATH", "COHERENCE_TLS_CLIENT_CERT", "COHERENCE_TLS_CLIENT_KEY"} {
		t.Setenv(key, "")
	}
}

func requestTestServer(t *testing.T, useTLS bool) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Coherence-Node-Id", "7")
		_, _ = io.WriteString(w, "{}")
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	if useTLS {
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	return server
}

func connectionTestFetcher(t *testing.T, baseURL string, enabled bool) HTTPFetcher {
	t.Helper()
	f, err := GetFetcherForConnection(HTTP, baseURL, "", "", "fixture", enabled)
	if err != nil {
		t.Fatal(err)
	}
	return f.(HTTPFetcher)
}

func requireCertificateRejection(t *testing.T, err error) {
	t.Helper()
	var certificateError *tls.CertificateVerificationError
	if !errors.As(err, &certificateError) {
		t.Fatalf("expected certificate rejection, got %v", err)
	}
}

func writeTLSFixture(t *testing.T, file string, data []byte) {
	t.Helper()
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func requestTestClientCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fixture-client"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, public, private)
	if err != nil {
		t.Fatal(err)
	}
	key, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: certificatePEMType, Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
}
