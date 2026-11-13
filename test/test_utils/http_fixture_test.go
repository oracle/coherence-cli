// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package test_utils

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fixtureStatusPath = "/status"

func assignFixtureOrigins(t *testing.T, context *TestContext) {
	t.Helper()
	previous := currentTestContext
	SetTestContext(context)
	t.Cleanup(func() { SetTestContext(previous) })
}

func TestFixtureGetAssignedOrigins(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "cluster ready") }))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "federation ready") }))
	defer second.Close()
	// Explicit assignment supports non-default ports and named remote origins;
	// the local listener supplies an independently controlled live destination.
	named := strings.Replace(first.URL, "127.0.0.1", "localhost", 1)
	assignFixtureOrigins(t, &TestContext{Url: named + "/management/coherence/cluster", RestUrl: named, AdditionalHTTPOrigins: []string{second.URL}})
	for target, want := range map[string]string{named + "/version": "cluster ready", second.URL + "/populateFederation": "federation ready"} {
		body, err := IssueGetRequest(target)
		if err != nil || string(body) != want {
			t.Fatalf("%s: %s %v", target, body, err)
		}
	}
	if got, err := fixtureOrigin("https://assigned.example/path"); err != nil || got != "https://assigned.example:443" {
		t.Fatalf("remote origin lost: %q %v", got, err)
	}
	if got, err := fixtureOrigin("http://[::1]:30000/management"); err != nil || got != "http://[::1]:30000" {
		t.Fatalf("IPv6 origin lost: %q %v", got, err)
	}
}

func TestFixtureGetRejectsUnassignedTargets(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	assignFixtureOrigins(t, &TestContext{RestUrl: server.URL})
	for _, target := range []string{"file:///etc/passwd", "ftp://127.0.0.1:21/file", "//127.0.0.1:80/", "http://169.254.169.254/latest", "http://localhost:0/", "http://localhost:65536/", "http://localhost:/", "http://user:password@" + strings.TrimPrefix(server.URL, "http://"), server.URL + "/#fragment", strings.Replace(server.URL, "http:", "https:", 1), "http://localhost:1/"} {
		t.Run(target, func(t *testing.T) {
			body, err := IssueGetRequest(target)
			if err == nil || len(body) != 0 {
				t.Fatalf("rejected target returned %q %v", body, err)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatal("rejected input reached HTTP transport")
	}
	SetTestContext(nil)
	if _, err := IssueGetRequest(server.URL); err == nil {
		t.Fatal("missing assignment permitted HTTP")
	}
}

func TestFixtureGetRedirectAndErrors(t *testing.T) {
	var outside atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { outside.Add(1); fmt.Fprint(w, "unassigned") }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/outside":
			http.Redirect(w, r, destination.URL, http.StatusFound)
		case "/inside":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case fixtureStatusPath:
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		case "/truncated":
			w.Header().Set("Content-Length", "10")
			fmt.Fprint(w, "short")
		default:
			fmt.Fprint(w, "OK")
		}
	}))
	defer server.Close()
	assignFixtureOrigins(t, &TestContext{RestUrl: server.URL})
	body, err := IssueGetRequest(server.URL + "/inside")
	if err != nil || string(body) != "OK" {
		t.Fatalf("same-origin redirect: %s %v", body, err)
	}
	for _, path := range []string{"/outside", fixtureStatusPath, "/truncated"} {
		body, err := IssueGetRequest(server.URL + path)
		if err == nil || len(body) != 0 {
			t.Fatalf("%s lost error/empty-body contract: %q %v", path, body, err)
		}
		if path == fixtureStatusPath && err.Error() != "Did not receive a 200 response code: 503 Service Unavailable" {
			t.Fatalf("status error changed: %v", err)
		}
	}
	if outside.Load() != 0 {
		t.Fatal("cross-origin redirect reached unassigned listener")
	}
}

func TestFixtureGetClosesHTTPResources(t *testing.T) {
	var active atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fixtureStatusPath {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/short" {
			w.Header().Set("Content-Length", "20")
		}
		fmt.Fprint(w, "body")
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			active.Add(1)
		}
		if state == http.StateClosed {
			active.Add(-1)
		}
	}
	server.Start()
	defer server.Close()
	assignFixtureOrigins(t, &TestContext{RestUrl: server.URL})
	for _, path := range []string{"/ok", fixtureStatusPath, "/short"} {
		for range 5 {
			_, err := IssueGetRequest(server.URL + path)
			if (err != nil) != (path != "/ok") {
				t.Fatalf("%s error=%v", path, err)
			}
		}
		deadline := time.Now().Add(2 * time.Second)
		for active.Load() != 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if active.Load() != 0 {
			t.Fatalf("response resources left open after %s: %d", path, active.Load())
		}
	}
}
