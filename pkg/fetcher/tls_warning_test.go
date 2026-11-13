/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	bypassLogMessage       = "TLS certificate verification disabled"
	warningAlternateOrigin = "https://example.test:8443"
)

// Both the handler and request goroutines inspect these sinks during requests.
type warningTestBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *warningTestBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *warningTestBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func captureTLSWarnings(t *testing.T) (*warningTestBuffer, *warningTestBuffer) {
	t.Helper()
	oldState, oldLogger := commandTLSWarnings.Load(), Logger
	stderr, logs := &warningTestBuffer{}, &warningTestBuffer{}
	BeginCommandTLSWarnings(stderr)
	Logger = zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(logs), zap.WarnLevel))
	t.Cleanup(func() {
		commandTLSWarnings.Store(oldState)
		Logger = oldLogger
	})
	return stderr, logs
}

func requireTLSWarningCounts(t *testing.T, stderr, logs *warningTestBuffer, warnings, records int) {
	t.Helper()
	if got := strings.Count(stderr.String(), "WARNING:"); got != warnings {
		t.Fatalf("stderr warnings %d, want %d: %s", got, warnings, stderr.String())
	}
	if got := strings.Count(logs.String(), bypassLogMessage); got != records {
		t.Fatalf("log records %d, want %d: %s", got, records, logs.String())
	}
}

func TestTLSBypassWarningRequests(t *testing.T) {
	resetRequestTestState(t)
	stderr, logs := captureTLSWarnings(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The first warning and log entry must be visible before the server
		// handles even the first request, including concurrent callers.
		if stderr.String() == "" || logs.String() == "" {
			t.Error("request arrived before TLS diagnostics")
		}
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()
	h := connectionTestFetcher(t, server.URL, true)
	copyOfH := h
	requireTLSWarningCounts(t, stderr, logs, 0, 0)
	for _, f := range []HTTPFetcher{h, copyOfH, connectionTestFetcher(t, server.URL, true)} {
		if _, err := f.GetClusterDetailsJSON(); err != nil {
			t.Fatal(err)
		}
	}
	const concurrentRequests = 12
	var wg sync.WaitGroup
	for range concurrentRequests {
		wg.Go(func() {
			if _, err := copyOfH.GetClusterDetailsJSON(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	requireTLSWarningCounts(t, stderr, logs, 1, 3+concurrentRequests)
	// Separate names share no terminal suppression, even at the same origin.
	other, err := GetFetcherForConnection(HTTP, server.URL, "", "", "another-connection", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = other.GetClusterDetailsJSON(); err != nil {
		t.Fatal(err)
	}
	// A separately bound destination under the original name warns again.
	second := requestTestServer(t, true)
	if _, err = connectionTestFetcher(t, second.URL, true).GetClusterDetailsJSON(); err != nil {
		t.Fatal(err)
	}
	requireTLSWarningCounts(t, stderr, logs, 3, 5+concurrentRequests)
	// Attempts that fail after dispatch still use the exception and are logged.
	server.Close()
	if _, err = h.GetClusterDetailsJSON(); err == nil {
		t.Fatal("request to closed listener succeeded")
	}
	requireTLSWarningCounts(t, stderr, logs, 3, 6+concurrentRequests)
}

func TestTLSBypassWarningConcurrentFirstUse(t *testing.T) {
	resetRequestTestState(t)
	stderr, logs := captureTLSWarnings(t)
	server := requestTestServer(t, true)
	h := connectionTestFetcher(t, server.URL, true)
	const count = 16
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			copyOfH := h
			if _, err := copyOfH.GetClusterDetailsJSON(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	requireTLSWarningCounts(t, stderr, logs, 1, count)
}

func TestTLSBypassWarningVerifiedSilence(t *testing.T) {
	resetRequestTestState(t)
	stderr, logs := captureTLSWarnings(t)
	server, other := requestTestServer(t, true), requestTestServer(t, true)
	plain := requestTestServer(t, false)
	h := connectionTestFetcher(t, server.URL, true)
	secure, err := GetFetcherOrError(HTTP, server.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = secure.GetClusterDetailsJSON(); err == nil {
		t.Fatal("default request accepted untrusted TLS")
	}
	if _, err = connectionTestFetcher(t, server.URL, false).GetClusterDetailsJSON(); err == nil {
		t.Fatal("explicit false accepted untrusted TLS")
	}
	if _, err = httpGetRequestAbsolute(h, other.URL+"/absolute"); err == nil {
		t.Fatal("unrelated absolute request accepted untrusted TLS")
	}
	if _, _, err = httpGetRequestAbsoluteWithHeader(h, other.URL+"/ready"); err == nil {
		t.Fatal("unrelated health request accepted untrusted TLS")
	}
	if _, err = connectionTestFetcher(t, plain.URL, true).GetClusterDetailsJSON(); err != nil {
		t.Fatal(err)
	}
	if _, err = httpGetRequestAbsolute(h, plain.URL); err != nil {
		t.Fatal(err)
	}
	requireTLSWarningCounts(t, stderr, logs, 0, 0)
}

func TestTLSBypassWarningRedaction(t *testing.T) {
	resetRequestTestState(t)
	stderr, logs := captureTLSWarnings(t)
	server := requestTestServer(t, true)
	name := "fixture\nFORGED\r\x1b[31m\u2028"
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("private-user", "private-password")
	u.Path, u.RawQuery, u.Fragment = "/private-path", "private-query=value", "private-fragment"
	f, err := GetFetcherForConnection(HTTP, u.String(), "", "", name, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err != nil {
		t.Fatal(err)
	}
	requireTLSWarningCounts(t, stderr, logs, 1, 1)
	for _, sink := range []string{stderr.String(), logs.String()} {
		for _, secret := range []string{"private-user", "private-password", "private-path", "private-query", "private-fragment"} {
			if strings.Contains(sink, secret) {
				t.Fatalf("diagnostic leaked %q: %s", secret, sink)
			}
		}
		if strings.Count(sink, "\n") != 1 || strings.ContainsAny(sink, "\r\x1b\u2028") {
			t.Fatalf("unsafe diagnostic fields: %q", sink)
		}
	}
	if !strings.Contains(stderr.String(), strconv.QuoteToASCII(name)) || !strings.Contains(stderr.String(), server.URL) {
		t.Fatalf("missing safe connection/origin: %s", stderr.String())
	}
	var record map[string]interface{}
	if err = json.Unmarshal([]byte(logs.String()), &record); err != nil {
		t.Fatal(err)
	}
	if record["connection"] != strconv.QuoteToASCII(name) || record["origin"] != strconv.QuoteToASCII(server.URL) {
		t.Fatalf("unsafe log fields: %#v", record)
	}
}

// Pure key/format checks complement actual-request tests without requiring a
// privileged port, DNS resolution, or native IPv6 listener availability.
func TestTLSBypassWarningNormalizedKeys(t *testing.T) {
	resetRequestTestState(t)
	stderr, logs := captureTLSWarnings(t)
	for _, address := range []string{
		"HTTPS://EXAMPLE.TEST/private", "https://example.test:443/other",
		"https://example.test:00443", warningAlternateOrigin,
		"https://[2001:DB8::1]", "https://[2001:db8:0:0:0:0:0:1]:443/path",
		"https://[2001:db8::1]:8443", "https://[fe80::1%25eth0]:8443",
	} {
		h := connectionTestFetcher(t, address, true)
		h.recordTLSBypass()
	}
	requireTLSWarningCounts(t, stderr, logs, 5, 8)
	for _, origin := range []string{"https://example.test:443", warningAlternateOrigin, "https://[2001:db8::1]:443", "https://[2001:db8::1]:8443", "https://[fe80::1%25eth0]:8443"} {
		if !strings.Contains(stderr.String(), strconv.QuoteToASCII(origin)) {
			t.Fatalf("missing normalized origin %s: %s", origin, stderr.String())
		}
	}
}
