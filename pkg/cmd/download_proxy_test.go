/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oracle/coherence-cli/internal/testtls"
)

const downloadProxyHelperEnv = "COHCTL_TEST_DOWNLOAD_PROXY_HELPER"

type downloadProxyResult struct {
	Body  string
	Error string
}

// A fresh process avoids Go's cached environment proxy selection. No CLI
// startup or real configuration is involved; the parent owns every endpoint.
func TestDownloadProxyHelper(t *testing.T) {
	if os.Getenv(downloadProxyHelperEnv) == "" {
		return
	}
	resetConnectionRequestState(t)
	t.Setenv("COHERENCE_TLS_CERTS_PATH", os.Getenv("COHCTL_TEST_DOWNLOAD_PROXY_CA"))
	t.Setenv("COHERENCE_TLS_CLIENT_CERT", os.Getenv("COHCTL_TEST_DOWNLOAD_PROXY_CLIENT"))
	t.Setenv("COHERENCE_TLS_CLIENT_KEY", os.Getenv("COHCTL_TEST_DOWNLOAD_PROXY_KEY"))
	optIn := true
	Config = CoherenceCLIConfig{IgnoreInvalidCerts: true, CurrentContext: exceptedConnection,
		Clusters: []ClusterConnection{{Name: exceptedConnection, ConnectionType: httpType,
			ConnectionURL: os.Getenv("COHCTL_TEST_DOWNLOAD_PROXY_TARGET"), IgnoreInvalidCerts: &optIn}}}
	body, err := GetURLContents(os.Getenv("COHCTL_TEST_DOWNLOAD_PROXY_TARGET"))
	result := downloadProxyResult{Body: string(body)}
	if err != nil {
		result.Error = err.Error()
	}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestDownloadTLSProxyVerification(t *testing.T) {
	for _, test := range []struct {
		name                                     string
		plainProxy, plainTarget, wrongProxyName  bool
		trustProxy, trustTarget, wrongTargetName bool
		mutualTLS, wantSuccess, wantProxy        bool
	}{
		{name: "untrusted HTTPS proxy", trustTarget: true},
		{name: "hostname-invalid HTTPS proxy", trustProxy: true, trustTarget: true, wrongProxyName: true},
		{name: "untrusted target via HTTPS proxy", trustProxy: true, wantProxy: true},
		{name: "hostname-invalid target via HTTPS proxy", trustProxy: true, trustTarget: true, wrongTargetName: true, wantProxy: true},
		{name: "HTTPS proxy and target verified with authentication", trustProxy: true, trustTarget: true, wantProxy: true, wantSuccess: true},
		{name: "HTTP CONNECT and target verified with authentication", plainProxy: true, trustTarget: true, wantProxy: true, wantSuccess: true},
		{name: "HTTP proxy and HTTP target", plainProxy: true, plainTarget: true, wantProxy: true, wantSuccess: true},
		{name: "HTTPS proxy forwarding HTTP", trustProxy: true, plainTarget: true, wantProxy: true, wantSuccess: true},
		{name: "untrusted HTTPS proxy forwarding HTTP", plainTarget: true},
		{name: "verified mutual TLS on proxy and target", trustProxy: true, trustTarget: true, mutualTLS: true, wantProxy: true, wantSuccess: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			proxyCA, targetCA := testtls.NewCA(t, "proxy-ca"), testtls.NewCA(t, "target-ca")
			var clientRoots *x509.CertPool
			clientFile, keyFile := "", ""
			if test.mutualTLS {
				clientCA := testtls.NewCA(t, "client-ca")
				clientRoots = x509.NewCertPool()
				clientRoots.AddCert(clientCA.Certificate)
				client := clientCA.Issue(t, "proxy-download-client", true)
				key, err := x509.MarshalPKCS8PrivateKey(client.PrivateKey)
				if err != nil {
					t.Fatal(err)
				}
				clientFile = writeDownloadFixture(t, "client.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: client.Certificate[0]}))
				keyFile = writeDownloadFixture(t, "key.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
			}
			var proxyReached, targetReached atomic.Bool
			target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targetReached.Store(true)
				if test.mutualTLS && len(r.TLS.VerifiedChains) == 0 {
					t.Error("target did not verify client certificate")
				}
				_, _ = io.WriteString(w, "{}")
			}))
			hostname := "download.fixture.invalid"
			if test.wrongTargetName {
				hostname = "wrong.download.invalid"
			}
			startDownloadProxyServer(t, target, test.plainTarget, targetCA.Issue(t, hostname, false), clientRoots)
			localTarget, err := url.Parse(target.URL)
			if err != nil {
				t.Fatal(err)
			}
			probe := localTarget.Scheme + "://" + net.JoinHostPort("download.fixture.invalid", localTarget.Port())
			forward := httputil.NewSingleHostReverseProxy(localTarget)
			proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				proxyReached.Store(true)
				if test.mutualTLS && len(r.TLS.VerifiedChains) == 0 {
					t.Error("proxy did not verify client certificate")
				}
				if r.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte("download-user:download-password")) ||
					r.Host != net.JoinHostPort("download.fixture.invalid", localTarget.Port()) {
					t.Error("proxy authentication or destination changed")
					w.WriteHeader(http.StatusProxyAuthRequired)
					return
				}
				if r.Method != http.MethodConnect {
					forward.ServeHTTP(w, r)
					return
				}
				serveDownloadTunnel(t, w, target.Listener.Addr().String())
			}))
			proxyName := localHost
			if test.wrongProxyName {
				proxyName = "wrong.proxy.invalid"
			}
			startDownloadProxyServer(t, proxy, test.plainProxy, proxyCA.Issue(t, proxyName, false), clientRoots)
			proxyURL, err := url.Parse(proxy.URL)
			if err != nil {
				t.Fatal(err)
			}
			proxyURL.User = url.UserPassword("download-user", "download-password")
			roots := testtls.NewCA(t, "unrelated-ca").PEM
			if test.trustProxy {
				roots = append(roots, proxyCA.PEM...)
			}
			if test.trustTarget {
				roots = append(roots, targetCA.PEM...)
			}
			result := runDownloadProxyHelper(t, proxyURL.String(), probe, writeDownloadFixture(t, "roots.pem", roots), clientFile, keyFile)
			if test.wantSuccess {
				if result.Error != "" || result.Body != "{}" {
					t.Fatalf("proxy download: %+v", result)
				}
			} else if !strings.Contains(result.Error, "certificate") {
				t.Fatalf("expected certificate rejection, got %+v", result)
			}
			if proxyReached.Load() != test.wantProxy || targetReached.Load() != test.wantSuccess {
				t.Fatalf("unexpected request delivery: proxy %v, target %v", proxyReached.Load(), targetReached.Load())
			}
		})
	}
}

func startDownloadProxyServer(t *testing.T, server *httptest.Server, plain bool, certificate tls.Certificate, clientRoots *x509.CertPool) {
	t.Helper()
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	if plain {
		server.Start()
	} else {
		server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}, ClientCAs: clientRoots}
		if clientRoots != nil {
			server.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		}
		server.StartTLS()
	}
	t.Cleanup(server.Close)
}

func runDownloadProxyHelper(t *testing.T, proxy, target, roots, clientFile, keyFile string) downloadProxyResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// #nosec G204 G702 -- Fixed selector in this test binary, using only parent-owned local endpoints, CA files and temporary homes.
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDownloadProxyHelper$")
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		switch strings.ToUpper(key) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "REQUEST_METHOD", "HOME", "USERPROFILE", "COHCTL_HOME":
			continue
		}
		command.Env = append(command.Env, item)
	}
	home := t.TempDir()
	command.Env = append(command.Env, downloadProxyHelperEnv+"=1", "HTTP_PROXY="+proxy, "HTTPS_PROXY="+proxy,
		"COHCTL_TEST_DOWNLOAD_PROXY_TARGET="+target, "COHCTL_TEST_DOWNLOAD_PROXY_CA="+roots, "COHCTL_TEST_DOWNLOAD_PROXY_CLIENT="+clientFile, "COHCTL_TEST_DOWNLOAD_PROXY_KEY="+keyFile,
		"HOME="+home, "USERPROFILE="+home, "COHCTL_HOME="+home)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("download helper: %v, %s", err, output)
	}
	var result downloadProxyResult
	if err = json.Unmarshal(output, &result); err != nil {
		t.Fatalf("download helper output: %s, %v", output, err)
	}
	return result
}

func serveDownloadTunnel(t *testing.T, w http.ResponseWriter, address string) {
	t.Helper()
	// #nosec G704 -- Only the parent-owned target listener is dialed; request-supplied names are never resolved.
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
