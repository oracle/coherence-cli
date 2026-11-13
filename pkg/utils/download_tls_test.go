/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package utils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/oracle/coherence-cli/internal/testtls"
)

func TestDownloadTLSAdditiveTrust(t *testing.T) {
	resetDownloadTLSEnv(t)
	systemCA, customCA := testtls.NewCA(t, "system-fixture"), testtls.NewCA(t, "custom-fixture")
	roots := x509.NewCertPool()
	roots.AddCert(systemCA.Certificate)
	customFile := filepath.Join(t.TempDir(), "custom.pem")
	if err := os.WriteFile(customFile, customCA.PEM, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envTLSCertPath, customFile)
	config, err := getDownloadTLSConfig(func() (*x509.CertPool, error) { return roots.Clone(), nil })
	if err != nil {
		t.Fatal(err)
	}
	_, clusterRoots, _, _, _, err := GetTLSDetails()
	if err != nil {
		t.Fatal(err)
	}
	// Both distinct real chains work for downloads; cluster custom trust still
	// replaces system trust. An unrelated CA cannot be accepted by either path.
	for _, test := range []struct {
		name    string
		ca      testtls.CA
		cluster bool
	}{
		{"system", systemCA, false},
		{"custom", customCA, true},
		{"unrelated", testtls.NewCA(t, "unrelated-fixture"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "verified")
			}))
			server.Config.ErrorLog = log.New(io.Discard, "", 0)
			server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{test.ca.Issue(t, "127.0.0.1", false)}}
			server.StartTLS()
			defer server.Close()
			for name, tlsConfig := range map[string]*tls.Config{
				"download": config, "cluster": {RootCAs: clusterRoots},
			} {
				transport := &http.Transport{TLSClientConfig: tlsConfig}
				defer transport.CloseIdleConnections()
				client := &http.Client{Transport: transport}
				resp, requestErr := client.Get(server.URL)
				wantSuccess := test.name != "unrelated"
				if name == "cluster" {
					wantSuccess = test.cluster
				}
				if requestErr != nil {
					var certificateError *tls.CertificateVerificationError
					if wantSuccess || !errors.As(requestErr, &certificateError) {
						t.Fatalf("%s request: %v", name, requestErr)
					}
					continue
				}
				body, readErr := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if !wantSuccess || readErr != nil || string(body) != "verified" {
					t.Fatalf("%s response %q, error %v", name, body, readErr)
				}
			}
		})
	}
}

func TestDownloadTLSTrustFailures(t *testing.T) {
	resetDownloadTLSEnv(t)
	ca := testtls.NewCA(t, "root-fixture")
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.pem")
	invalid := filepath.Join(dir, "invalid.pem")
	for file, content := range map[string][]byte{valid: ca.PEM, invalid: []byte("invalid certificate data")} {
		if err := os.WriteFile(file, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	loadError := errors.New("system root fixture failed")
	for _, test := range []struct {
		name, path, message string
		pool                *x509.CertPool
		err                 error
	}{
		{"system error", valid, "system root fixture failed", nil, loadError},
		{"partial system error", valid, "system root fixture failed", x509.NewCertPool(), loadError},
		{"nil system pool", valid, "empty certificate pool", nil, nil},
		{"empty system pool", valid, "empty certificate pool", x509.NewCertPool(), nil},
		{"missing CA", filepath.Join(dir, "missing.pem"), "not a valid file", nil, nil},
		{"unreadable CA directory", dir, "", nil, nil},
		{"malformed CA", invalid, "failed to append certificates", nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(envTLSCertPath, test.path)
			config, err := getDownloadTLSConfig(func() (*x509.CertPool, error) {
				if strings.Contains(test.name, "system") {
					return test.pool, test.err
				}
				pool := x509.NewCertPool()
				pool.AddCert(ca.Certificate)
				return pool, nil
			})
			if config != nil || err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected closed failure %q, config %v, error %v", test.message, config, err)
			}
			if test.err != nil && !errors.Is(err, loadError) {
				t.Fatalf("lost system root cause: %v", err)
			}
		})
	}
}

func TestDownloadTLSDefaultSystemTrust(t *testing.T) {
	resetDownloadTLSEnv(t)
	config, err := getDownloadTLSConfig(func() (*x509.CertPool, error) {
		t.Fatal("no custom CA must retain normal lazy system trust")
		return nil, nil
	})
	if err != nil || config == nil || config.RootCAs != nil || config.InsecureSkipVerify {
		t.Fatalf("default system trust: config %v, error %v", config, err)
	}
	// Exercise the real native loader too: on macOS/Windows the system pool
	// may expose no Subjects while still representing native system trust.
	customFile := filepath.Join(t.TempDir(), "custom.pem")
	t.Setenv(envTLSCertPath, customFile)
	if err = os.WriteFile(customFile, testtls.NewCA(t, "custom-native").PEM, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = GetDownloadTLSConfig(); err != nil {
		t.Fatalf("native system plus custom trust: %v", err)
	}
}

func TestDownloadTLSUnreadableCA(t *testing.T) {
	resetDownloadTLSEnv(t)
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission check; read failures are also covered with a directory")
	}
	file := filepath.Join(t.TempDir(), "unreadable.pem")
	if err := os.WriteFile(file, testtls.NewCA(t, "unreadable-ca").PEM, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(file, 0600); err != nil {
			t.Error(err)
		}
	})
	if handle, err := os.Open(file); err == nil {
		_ = handle.Close()
		t.Skip("current user can read files without permission bits")
	}
	t.Setenv(envTLSCertPath, file)
	config, err := GetDownloadTLSConfig()
	if config != nil || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unreadable CA must fail closed: config %v, error %v", config, err)
	}
}

func resetDownloadTLSEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{envTLSCertPath, envTLSClientCert, envTLSClientKey} {
		t.Setenv(key, "")
	}
}
