/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/oracle/coherence-go-client/v2/coherence/discovery"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	exceptedConnection       = "excepted"
	verifiedConnection       = "verified"
	manualDiscoveryType      = "manual"
	nameServiceDiscoveryType = "nslookup"
)

func TestSelectedConnectionTLSRequests(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	plain := connectionRequestServer(t, false)
	falseValue, trueValue := false, true
	Config = CoherenceCLIConfig{
		CurrentContext: exceptedConnection, IgnoreInvalidCerts: true,
		Clusters: []ClusterConnection{
			{Name: exceptedConnection, ConnectionType: httpType, ConnectionURL: server.URL, IgnoreInvalidCerts: &trueValue},
			{Name: verifiedConnection, ConnectionType: httpType, ConnectionURL: server.URL, IgnoreInvalidCerts: &falseValue},
			{Name: "unset", ConnectionType: httpType, ConnectionURL: server.URL},
			{Name: "plain", ConnectionType: httpType, ConnectionURL: plain.URL, IgnoreInvalidCerts: &trueValue},
		},
	}
	for _, test := range []struct {
		selection, context string
		success            bool
	}{
		{"", exceptedConnection, true},
		{verifiedConnection, exceptedConnection, false},
		{"unset", exceptedConnection, false},
		{exceptedConnection, verifiedConnection, true},
		{"plain", exceptedConnection, true},
	} {
		t.Run(test.selection+"/"+test.context, func(t *testing.T) {
			clusterConnection, Config.CurrentContext = test.selection, test.context
			name, err := GetConnectionNameFromContextOrArg()
			if err != nil {
				t.Fatal(err)
			}
			f, err := GetDataFetcher(name)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.GetClusterDetailsJSON()
			if (err == nil) != test.success {
				t.Fatalf("selected %q via context %q: %v", name, test.context, err)
			}
			if !test.success && !strings.Contains(err.Error(), "certificate") {
				t.Fatalf("expected certificate verification failure, got %v", err)
			}
		})
	}
	// An existing fetcher's choice cannot follow a later saved-config edit.
	f, err := GetDataFetcher(exceptedConnection)
	if err != nil {
		t.Fatal(err)
	}
	trueValue = false
	if _, err = f.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("construction-bound policy changed: %v", err)
	}
	f, err = GetDataFetcher(exceptedConnection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err == nil {
		t.Fatal("new fetcher ignored saved false")
	}
}

func TestAdHocDiscoveryTLSRequests(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	trueValue := true
	Config = CoherenceCLIConfig{CurrentContext: exceptedConnection, IgnoreInvalidCerts: true,
		Clusters: []ClusterConnection{{Name: exceptedConnection, ConnectionType: httpType, ConnectionURL: server.URL, IgnoreInvalidCerts: &trueValue}}}
	discover := func(_ *cobra.Command, addresses []string) ([]discovery.DiscoveredCluster, []int, error) {
		if len(addresses) != 1 || addresses[0] != "127.0.0.1:7574" {
			t.Fatalf("unexpected discovery request: %v", addresses)
		}
		return []discovery.DiscoveredCluster{{ClusterName: "fixture", ManagementURLs: []string{server.URL}}}, nil, nil
	}
	// Only the Name Service lookup is stubbed; actual selection, construction and
	// the TLS request execute, with the discovered URL equal to the current context.
	f, err := getDataFetcher("127.0.0.1:7574", discover)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("ad hoc discovery inherited the saved exception: %v", err)
	}
}

func TestConcurrentSelectedConnectionAndDownloadTLSIsolation(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	file := writeMigrationFixture(t, fmt.Sprintf(`currentContext: excepted
ignoreInvalidCerts: false
clusters:
  - name: excepted
    connectiontype: http
    connectionurl: %[1]s
    ignoreinvalidcerts: true
  - name: verified
    connectiontype: http
    connectionurl: %[1]s
    ignoreinvalidcerts: false
  - name: unset
    connectiontype: http
    connectionurl: %[1]s
`, server.URL))
	_, Config = readConnectionConfig(t, file)
	t.Setenv("IGNOREINVALIDCERTS", stringTrue)
	type requestCase struct {
		name    string
		request func() ([]byte, error)
		success bool
	}
	var requests []requestCase
	// Resolve context/overrides and construct serially: legacy configuration,
	// credential and TLS initialization are deliberately outside this race test.
	for _, selection := range []string{"", verifiedConnection, certificateUnsetName} {
		clusterConnection = selection
		name, err := GetConnectionNameFromContextOrArg()
		if err != nil {
			t.Fatal(err)
		}
		f, err := GetDataFetcher(name)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, requestCase{name, f.GetClusterDetailsJSON, name == exceptedConnection})
	}
	discover := func(_ *cobra.Command, _ []string) ([]discovery.DiscoveredCluster, []int, error) {
		return []discovery.DiscoveredCluster{{ClusterName: migrationConnectionName, ManagementURLs: []string{server.URL}}}, nil, nil
	}
	adHoc, err := getDataFetcher("127.0.0.1:7574", discover)
	if err != nil {
		t.Fatal(err)
	}
	requests = append(requests,
		requestCase{"ad hoc", adHoc.GetClusterDetailsJSON, false},
		requestCase{"general download", func() ([]byte, error) { return GetURLContents(server.URL) }, false})
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		for _, request := range requests {
			wg.Go(func() {
				<-start
				body, requestErr := request.request()
				if request.success {
					if requestErr != nil || string(body) != "{}" {
						t.Errorf("%s: %q, %v", request.name, body, requestErr)
					}
				} else {
					// The public cluster API formats errors without preserving their
					// type. Require the actual certificate rejection, not any error.
					if requestErr == nil || !strings.Contains(requestErr.Error(), "x509: certificate signed by unknown authority") {
						t.Errorf("%s inherited an exception or failed for another reason: %v", request.name, requestErr)
					}
				}
			})
		}
	}
	close(start)
	wg.Wait()
}

func TestMigratedConnectionTLSRequests(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	contents := fmt.Sprintf(`version: fixture
currentContext: migrated
ignoreInvalidCerts: true
clusters:
  - name: migrated
    connectiontype: http
    connectionurl: %s
  - name: explicit-false
    connectiontype: http
    connectionurl: %s
    ignoreinvalidcerts: false
`, server.URL, server.URL)
	file := writeMigrationFixture(t, contents)
	live, _ := readConnectionConfig(t, file)
	live.AutomaticEnv()
	t.Setenv("IGNOREINVALIDCERTS", stringTrue)
	var stderr bytes.Buffer
	var err error
	Config, err = loadExistingConfig(live, &stderr, defaultConfigMigrationOps())
	if err != nil || Config.IgnoreInvalidCerts || live.GetBool(ignoreCertsContextKey) {
		t.Fatalf("migration failed to mask legacy global: %v", err)
	}
	for _, name := range []string{"migrated", "explicit-false"} {
		f, constructionErr := GetDataFetcher(name)
		if constructionErr != nil {
			t.Fatal(constructionErr)
		}
		_, requestErr := f.GetClusterDetailsJSON()
		if (requestErr == nil) != (name == "migrated") {
			t.Fatalf("migrated connection %s request: %v", name, requestErr)
		}
	}
	// Freshly reloaded persisted choices must have the same request behavior.
	_, Config = readConnectionConfig(t, file)
	f, err := GetDataFetcher(Config.CurrentContext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.GetClusterDetailsJSON(); err != nil {
		t.Fatalf("reloaded migrated exception: %v", err)
	}
}

func TestAddClusterTLSVerifiedByDefault(t *testing.T) {
	resetConnectionRequestState(t)
	server := connectionRequestServer(t, true)
	trueValue := true
	Config = CoherenceCLIConfig{CurrentContext: exceptedConnection, IgnoreInvalidCerts: true,
		Clusters: []ClusterConnection{{Name: exceptedConnection, ConnectionType: httpType, ConnectionURL: server.URL, IgnoreInvalidCerts: &trueValue}}}
	for _, discoveryType := range []string{manualDiscoveryType, nameServiceDiscoveryType} {
		command := &cobra.Command{}
		var output bytes.Buffer
		command.SetOut(&output)
		err := addCluster(command, "new-connection", server.URL, discoveryType, "", false)
		if err == nil || !strings.Contains(err.Error(), "certificate") || len(Config.Clusters) != 1 || output.Len() != 0 {
			t.Fatalf("initial %s probe inherited a bypass or persisted on failure: %v, %s", discoveryType, err, output.String())
		}
	}
}

func resetConnectionRequestState(t *testing.T) {
	t.Helper()
	oldConfig, oldConnection, oldUsername, oldUsingContext := Config, clusterConnection, Username, UsingContext
	oldURL, oldCluster := httpManagementURL, httpManagementCluster
	oldDebug, oldTimeout := fetcher.DebugEnabled, fetcher.RequestTimeout
	oldLogger := utils.Logger
	t.Cleanup(func() {
		Config, clusterConnection, Username, UsingContext = oldConfig, oldConnection, oldUsername, oldUsingContext
		httpManagementURL, httpManagementCluster = oldURL, oldCluster
		fetcher.DebugEnabled, fetcher.RequestTimeout = oldDebug, oldTimeout
		utils.Logger = oldLogger
	})
	clusterConnection, Username = "", ""
	fetcher.DebugEnabled, fetcher.RequestTimeout = false, 5
	utils.Logger = zap.NewNop()
	for _, key := range []string{"COHERENCE_TLS_CERTS_PATH", "COHERENCE_TLS_CLIENT_CERT", "COHERENCE_TLS_CLIENT_KEY"} {
		t.Setenv(key, "")
	}
}

func connectionRequestServer(t *testing.T, useTLS bool) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
