/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	proxyHelperEnv = "COHCTL_TEST_TLS_PROXY_HELPER"
	proxyTargetDNS = "cluster.fixture.invalid"
	proxyTestUser  = "proxy-user"
	proxyTestPass  = "proxy-password"
)

type proxyRequestResult struct {
	Body  string
	Error string
}

// Each child has a fresh http.ProxyFromEnvironment cache. All endpoints and CA
// files come from the parent test's loopback fixtures, with no user configuration.
func TestConnectionTLSProxyHelper(t *testing.T) {
	if os.Getenv(proxyHelperEnv) == "" {
		return
	}
	resetRequestTestState(t)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", os.Getenv("COHCTL_TEST_TLS_PROXY_CA"))
	f, err := GetFetcherForConnection(HTTP, os.Getenv("COHCTL_TEST_TLS_PROXY_TARGET"), "", "", "proxy-fixture", os.Getenv("COHCTL_TEST_TLS_PROXY_OPT_IN") == "true")
	if err != nil {
		t.Fatal(err)
	}
	count, err := strconv.Atoi(os.Getenv("COHCTL_TEST_TLS_PROXY_REQUEST_COUNT"))
	if err != nil || count < 1 || count > 4 {
		t.Fatal("invalid helper request count")
	}
	results := make([]proxyRequestResult, count)
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			var body []byte
			var requestErr error
			if absolute := os.Getenv("COHCTL_TEST_TLS_PROXY_ABSOLUTE"); absolute != "" {
				body, requestErr = httpGetRequestAbsolute(f.(HTTPFetcher), absolute)
			} else {
				body, requestErr = f.GetClusterDetailsJSON()
			}
			results[i].Body = string(body)
			if requestErr != nil {
				results[i].Error = requestErr.Error()
			}
		})
	}
	wg.Wait()
	if err = json.NewEncoder(os.Stdout).Encode(results); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestConnectionTLSProxyRequests(t *testing.T) {
	for _, test := range []struct {
		name                                     string
		plainProxy, plainTarget, wrongProxyName  bool
		trustProxy, trustTarget, optIn, absolute bool
		proxyAuth                                bool
		count                                    int
		wantSuccess, wantProxyRequest            bool
	}{
		{name: "untrusted HTTPS proxy", optIn: true},
		{name: "hostname-invalid HTTPS proxy", trustProxy: true, wrongProxyName: true, optIn: true},
		{name: "trusted HTTPS proxy and opted-in target", trustProxy: true, optIn: true, wantSuccess: true, wantProxyRequest: true},
		{name: "trusted HTTPS proxy and untrusted verified target", trustProxy: true, wantProxyRequest: true},
		{name: "trusted HTTPS proxy and trusted verified target", trustProxy: true, trustTarget: true, wantSuccess: true, wantProxyRequest: true},
		{name: "other-origin absolute target stays verified", trustProxy: true, optIn: true, absolute: true, wantProxyRequest: true},
		{name: "HTTP CONNECT and opted-in target", plainProxy: true, optIn: true, wantSuccess: true, wantProxyRequest: true},
		{name: "HTTP CONNECT and untrusted verified target", plainProxy: true, wantProxyRequest: true},
		{name: "HTTPS proxy authentication", trustProxy: true, optIn: true, proxyAuth: true, wantSuccess: true, wantProxyRequest: true},
		{name: "HTTP CONNECT authentication", plainProxy: true, optIn: true, proxyAuth: true, wantSuccess: true, wantProxyRequest: true},
		{name: "HTTPS proxy forwarding HTTP", trustProxy: true, plainTarget: true, optIn: true, wantSuccess: true, wantProxyRequest: true},
		{name: "untrusted HTTPS proxy with HTTP target", plainTarget: true, optIn: true},
		{name: "concurrent HTTPS proxy requests", trustProxy: true, optIn: true, count: 4, wantSuccess: true, wantProxyRequest: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			proxyCA, targetCA := newProxyTestCA(t, "proxy-ca"), newProxyTestCA(t, "target-ca")
			var targetRequests, proxyRequests atomic.Int32
			target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				targetRequests.Add(1)
				_, _ = io.WriteString(w, "{}")
			}))
			target.Config.ErrorLog = log.New(io.Discard, "", 0)
			if test.plainTarget {
				target.Start()
			} else {
				target.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{targetCA.serverCertificate(t, proxyTargetDNS)}}
				target.StartTLS()
			}
			t.Cleanup(target.Close)
			localTarget, err := url.Parse(target.URL)
			if err != nil {
				t.Fatal(err)
			}
			probeURL := localTarget.Scheme + "://" + net.JoinHostPort(proxyTargetDNS, localTarget.Port())
			forward := httputil.NewSingleHostReverseProxy(localTarget)
			proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxyRequests.Add(1)
				if test.proxyAuth && r.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(proxyTestUser+":"+proxyTestPass)) {
					t.Error("proxy credentials were not preserved")
					w.WriteHeader(http.StatusProxyAuthRequired)
					return
				}
				if r.Host != net.JoinHostPort(proxyTargetDNS, localTarget.Port()) {
					t.Errorf("unexpected proxy destination: %s", r.Host)
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				if r.Method != http.MethodConnect {
					forward.ServeHTTP(w, r)
					return
				}
				serveProxyTestTunnel(t, w, target.Listener.Addr().String())
			}))
			proxy.Config.ErrorLog = log.New(io.Discard, "", 0)
			if test.plainProxy {
				proxy.Start()
			} else {
				proxyHost := "127.0.0.1"
				if test.wrongProxyName {
					proxyHost = "wrong.proxy.invalid"
				}
				proxy.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{proxyCA.serverCertificate(t, proxyHost)}}
				proxy.StartTLS()
			}
			t.Cleanup(proxy.Close)
			proxyURL, err := url.Parse(proxy.URL)
			if err != nil {
				t.Fatal(err)
			}
			if test.proxyAuth {
				proxyURL.User = url.UserPassword(proxyTestUser, proxyTestPass)
			}
			// Always trust a third, unrelated CA so untrusted cases have a valid
			// nonempty CA file and cannot use an accidental system trust fallback.
			roots := newProxyTestCA(t, "unrelated-ca").pem
			if test.trustProxy {
				roots = append(roots, proxyCA.pem...)
			}
			if test.trustTarget {
				roots = append(roots, targetCA.pem...)
			}
			caFile := filepath.Join(t.TempDir(), "proxy-roots.pem")
			writeTLSFixture(t, caFile, roots)
			allowedURL, absoluteURL := probeURL, ""
			if test.absolute {
				allowedURL = "https://different.fixture.invalid:443"
				absoluteURL = probeURL + "/absolute"
			}
			count := test.count
			if count == 0 {
				count = 1
			}
			results := runProxyRequestHelper(t, proxyURL.String(), allowedURL, absoluteURL, caFile, test.optIn, count)
			for _, result := range results {
				if test.wantSuccess {
					if result.Error != "" || result.Body != "{}" {
						t.Errorf("proxy request failed: %+v", result)
					}
				} else if !strings.Contains(result.Error, "certificate") {
					t.Errorf("expected certificate rejection, got %+v", result)
				}
			}
			wantProxy, wantTarget := int32(0), int32(0)
			if test.wantProxyRequest {
				wantProxy = 1
			}
			if test.wantSuccess {
				wantTarget = 1
			}
			if int(proxyRequests.Load()) != count*int(wantProxy) || int(targetRequests.Load()) != count*int(wantTarget) {
				t.Fatalf("proxy requests %d, target requests %d (count %d)", proxyRequests.Load(), targetRequests.Load(), count)
			}
		})
	}
}

func runProxyRequestHelper(t *testing.T, proxy, target, absolute, caFile string, optIn bool, count int) []proxyRequestResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// #nosec G204 G702 -- Fixed test selector in this test binary; the parent supplies only task-owned CA files and loopback proxy/target fixtures.
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestConnectionTLSProxyHelper$")
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		switch strings.ToUpper(key) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "REQUEST_METHOD":
			continue
		}
		command.Env = append(command.Env, item)
	}
	command.Env = append(command.Env, proxyHelperEnv+"=1", "HTTP_PROXY="+proxy, "HTTPS_PROXY="+proxy,
		"COHCTL_TEST_TLS_PROXY_TARGET="+target, "COHCTL_TEST_TLS_PROXY_ABSOLUTE="+absolute, "COHCTL_TEST_TLS_PROXY_CA="+caFile,
		"COHCTL_TEST_TLS_PROXY_OPT_IN="+strconv.FormatBool(optIn), "COHCTL_TEST_TLS_PROXY_REQUEST_COUNT="+strconv.Itoa(count))
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("proxy helper: %v, stdout %s, stderr %s", err, output, stderr.String())
	}
	var results []proxyRequestResult
	if err = json.Unmarshal(output, &results); err != nil || len(results) != count {
		t.Fatalf("proxy helper output: %s, %v", output, err)
	}
	return results
}

func serveProxyTestTunnel(t *testing.T, w http.ResponseWriter, address string) {
	t.Helper()
	// #nosec G704 -- This address comes only from the task-owned target listener; the proxy never resolves a request-supplied hostname.
	upstream, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Error(err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer upstream.Close()
	client, buffered, err := w.(http.Hijacker).Hijack()
	if err != nil {
		t.Error(err)
		return
	}
	defer client.Close()
	if _, err = io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		t.Error(err)
		return
	}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(upstream, buffered)
		_ = upstream.Close()
		close(done)
	}()
	_, _ = io.Copy(client, upstream)
	_ = client.Close()
	<-done
}

type proxyTestCA struct {
	certificate *x509.Certificate
	key         ed25519.PrivateKey
	pem         []byte
}

func newProxyTestCA(t *testing.T, name string) proxyTestCA {
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
	return proxyTestCA{certificate: certificate, key: private, pem: pem.EncodeToMemory(&pem.Block{Type: certificatePEMType, Bytes: der})}
}

func (ca proxyTestCA) serverCertificate(t *testing.T, hostname string) tls.Certificate {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: hostname},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(hostname); ip != nil {
		certificate.IPAddresses = []net.IP{ip}
	} else {
		certificate.DNSNames = []string{hostname}
	}
	der, err := x509.CreateCertificate(rand.Reader, certificate, ca.certificate, public, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: private}
}
