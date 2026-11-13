/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/oracle/coherence-cli/internal/testtls"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	certificateCommand          = "ignore-certs"
	certificateFlag             = "--ignore-certs"
	certificateFalseFlag        = "--ignore-certs=false"
	certificateGetAction        = "get"
	certificateSetAction        = "set"
	certificateAddAction        = "add"
	certificateClusterCommand   = "cluster"
	certificateUnsetName        = "unset"
	certificatePlainName        = "plain"
	certificateDiscoveryAddress = "127.0.0.1:7574"
	certificateJSONFormat       = "json"
	certificateTOMLFormat       = "toml"
	certificateExtensionKey     = "Extension"
	certificateTargetName       = "target"
)

const certificateCommandFixture = `version: fixture
color: off
currentContext: unset
ignoreInvalidCerts: false
extension:
  owner: fixture
clusters:
  - name: unset
    connectiontype: http
    connectionurl: https://unreachable.invalid:30000/
  - name: verified
    connectiontype: http
    connectionurl: https://unreachable.invalid:30001/
    ignoreinvalidcerts: false
  - name: excepted
    connectiontype: http
    connectionurl: https://unreachable.invalid:30002/
    ignoreinvalidcerts: true
  - name: plain
    connectiontype: http
    connectionurl: http://unreachable.invalid:30000/
    ignoreinvalidcerts: true
`

// Each invocation models a fresh CLI parse/load while staying in one process.
// Cobra commands are package singletons: reset their local flags explicitly so
// a previous command cannot make a default-choice test pass with stale options.
func runCertificateCommand(t *testing.T, file string, args ...string) (string, error) {
	t.Helper()
	for _, command := range []*cobra.Command{addClusterCmd, getConfigCmd, getClustersCmd} {
		for _, name := range []string{certificateCommand, "url", "type", "verbose"} {
			flag := command.Flags().Lookup(name)
			if flag != nil {
				if err := flag.Value.Set(flag.DefValue); err != nil {
					t.Fatal(err)
				}
				flag.Changed = false
			}
		}
	}
	command := Initialize(nil)
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs(append([]string{"--config", file}, args...))
	err := command.Execute()
	return stdout.String(), err
}

func certificateCommandConfig(t *testing.T, contents string) string {
	t.Helper()
	resetConnectionRequestState(t)
	home := t.TempDir()
	t.Setenv("COHCTL_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("IGNOREINVALIDCERTS", stringTrue)
	file := filepath.Join(home, "cohctl.yaml")
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		viper.Reset()
		_ = addClusterCmd.Flags().Set(certificateCommand, stringFalse)
		_ = addClusterCmd.Flags().Set("url", "")
		_ = addClusterCmd.Flags().Set("type", httpType)
		verboseOutput = false
		cfgDirectory, cfgFile = "", ""
	})
	return file
}

func TestIgnoreCertsCommands(t *testing.T) {
	file := certificateCommandConfig(t, certificateCommandFixture)
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"absent context", []string{certificateGetAction, certificateCommand}, getIgnoreCertsMsg + stringFalse + "\n"},
		{"explicit false", []string{certificateGetAction, certificateCommand, "-c", "verified"}, getIgnoreCertsMsg + stringFalse + "\n"},
		{"explicit true overrides context", []string{certificateGetAction, certificateCommand, "--connection", exceptedConnection}, getIgnoreCertsMsg + stringTrue + "\n"},
		{"HTTP", []string{certificateGetAction, certificateCommand, "-c", certificatePlainName}, "Current setting: n/a\n"},
		{"set override", []string{certificateSetAction, certificateCommand, stringFalse, "-c", exceptedConnection}, "Value is now set to false\n"},
		{"reload override", []string{certificateGetAction, certificateCommand, "-c", exceptedConnection}, getIgnoreCertsMsg + stringFalse + "\n"},
		{"set context", []string{certificateSetAction, certificateCommand, stringTrue}, "Value is now set to true\n"},
		{"reload context", []string{certificateGetAction, certificateCommand}, getIgnoreCertsMsg + stringTrue + "\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := runCertificateCommand(t, file, test.args...)
			if err != nil || output != test.want {
				t.Fatalf("output %q, err %v; want %q", output, err, test.want)
			}
		})
	}
	live, reloaded := readConnectionConfig(t, file)
	if reloaded.IgnoreInvalidCerts || live.GetBool(ignoreCertsContextKey) {
		t.Fatal("setter reintroduced a global exception")
	}
	if reloaded.CurrentContext != certificateUnsetName || live.GetString("extension.owner") != migrationConnectionName {
		t.Fatal("unrelated configuration changed")
	}
	want := []bool{true, false, false, true}
	for i, connection := range reloaded.Clusters {
		if connection.IgnoreInvalidCerts == nil || *connection.IgnoreInvalidCerts != want[i] {
			t.Fatalf("unexpected saved choice: %#v", connection)
		}
	}
}

func TestIgnoreCertsSelectionAndArgumentErrors(t *testing.T) {
	for _, test := range []struct {
		name, fixture string
		args          []string
		want          string
	}{
		{"missing get", strings.Replace(certificateCommandFixture, "currentContext: unset", "currentContext: ''", 1), []string{certificateGetAction, certificateCommand}, "provide a connection name"},
		{"missing set", strings.Replace(certificateCommandFixture, "currentContext: unset", "currentContext: ''", 1), []string{certificateSetAction, certificateCommand, stringTrue}, "provide a connection name"},
		{"unknown get", certificateCommandFixture, []string{certificateGetAction, certificateCommand, "-c", "unknown"}, UnableToFindClusterMsg},
		{"unknown context", strings.Replace(certificateCommandFixture, "currentContext: unset", "currentContext: unknown", 1), []string{certificateSetAction, certificateCommand, stringTrue}, UnableToFindClusterMsg},
		{"ad hoc get", certificateCommandFixture, []string{certificateGetAction, certificateCommand, "-c", certificateDiscoveryAddress}, "saved connection name"},
		{"ad hoc set", certificateCommandFixture, []string{certificateSetAction, certificateCommand, stringTrue, "-c", certificateDiscoveryAddress}, "saved connection name"},
		{"HTTP set true", certificateCommandFixture, []string{certificateSetAction, certificateCommand, stringTrue, "-c", certificatePlainName}, "valid HTTPS URL"},
		{"HTTP set false", certificateCommandFixture, []string{certificateSetAction, certificateCommand, stringFalse, "-c", certificatePlainName}, "valid HTTPS URL"},
		{"invalid value", certificateCommandFixture, []string{certificateSetAction, certificateCommand, "TRUE"}, setIgnoreCertsError},
		{"missing value", certificateCommandFixture, []string{certificateSetAction, certificateCommand}, "provide either true or false"},
		{"extra value", certificateCommandFixture, []string{certificateSetAction, certificateCommand, stringTrue, stringFalse}, "provide either true or false"},
		{"get argument", certificateCommandFixture, []string{certificateGetAction, certificateCommand, stringTrue}, "accepts 0 arg(s)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := certificateCommandConfig(t, test.fixture)
			// Normalize startup-only version writes before checking rejected commands.
			if _, err := runCertificateCommand(t, file, certificateGetAction, "context"); err != nil {
				t.Fatal(err)
			}
			before := readMigrationBytes(t, file)
			output, err := runCertificateCommand(t, file, test.args...)
			if err == nil || !strings.Contains(err.Error(), test.want) || strings.Contains(output, setIgnoreCertsMsg) {
				t.Fatalf("output %q, error %v", output, err)
			}
			if !bytes.Equal(before, readMigrationBytes(t, file)) {
				t.Fatal("rejected command changed saved configuration")
			}
		})
	}
}

func TestIgnoreCertsURLEligibility(t *testing.T) {
	for _, address := range []string{
		"http://host/", "ftp://host/", "https:opaque", "https:///path", "https://[not-ip]/", "https://host]/", "https://host:bad/", "https://host:65536/", "https://host:0/", "https://host:000/", "https://host:/", "https://::1/", "https://%zz/",
	} {
		t.Run(address, func(t *testing.T) {
			fixture := strings.Replace(certificateCommandFixture, "https://unreachable.invalid:30000/", address, 1)
			file := certificateCommandConfig(t, fixture)
			output, err := runCertificateCommand(t, file, certificateGetAction, certificateCommand)
			if err != nil || output != getIgnoreCertsMsg+na+"\n" {
				t.Fatalf("get: %q, %v", output, err)
			}
			for _, value := range []string{stringTrue, stringFalse} {
				if _, err = runCertificateCommand(t, file, certificateSetAction, certificateCommand, value); err == nil {
					t.Fatal("ineligible URL accepted")
				}
			}
		})
	}
	for _, address := range []string{"https://host/", "HTTPS://HOST:443/", "https://[::1]:30000/", "https://host:00443/"} {
		if !fetcher.IsValidHTTPSURL(address) {
			t.Fatalf("valid HTTPS URL rejected: %s", address)
		}
	}
}

func TestIgnoreCertsCommandsDoNotRequestNetwork(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			requests.Add(1)
		}
	}
	server.Start()
	defer server.Close()
	file := certificateCommandConfig(t, strings.ReplaceAll(certificateCommandFixture, "https://unreachable.invalid:30000/", strings.Replace(server.URL, "http:", "https:", 1)))
	for _, args := range [][]string{{certificateGetAction, certificateCommand}, {certificateSetAction, certificateCommand, stringTrue}, {certificateSetAction, certificateCommand, stringFalse}} {
		if _, err := runCertificateCommand(t, file, args...); err != nil {
			t.Fatal(err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("get/set issued network requests")
	}
}

func TestAddClusterCertificateChoices(t *testing.T) {
	file := certificateCommandConfig(t, "version: fixture\ncolor: off\nignoreinvalidcerts: false\n")
	if _, err := runCertificateCommand(t, file, certificateGetAction, "context"); err != nil {
		t.Fatal(err)
	}
	untrusted := connectionRequestServer(t, true)
	plain := connectionRequestServer(t, false)
	ca := testtls.NewCA(t, "add-trusted")
	trusted := downloadTLSServer(t, ca.Issue(t, localHost, false), nil)
	for _, test := range []struct {
		name, address string
		flags         []string
		success       bool
		choice        *bool
	}{
		{"default-untrusted", untrusted.URL, nil, false, nil},
		{exceptedConnection, untrusted.URL, []string{certificateFlag}, true, boolChoice(true)},
		{"default-after-exception", untrusted.URL, nil, false, nil},
		{"explicit-false", untrusted.URL, []string{certificateFalseFlag}, false, nil},
		{"HTTP-exception", plain.URL, []string{certificateFlag}, false, nil},
		{"HTTP-explicit-false", plain.URL, []string{certificateFalseFlag}, false, nil},
		{"host-port-exception", strings.TrimPrefix(plain.URL, "http://"), []string{certificateFlag}, false, nil},
		{"HTTP", plain.URL, nil, true, nil},
		{"host-port", strings.TrimPrefix(plain.URL, "http://"), nil, true, nil},
		{"trusted", trusted.URL, nil, true, boolChoice(false)},
		{"trusted-explicit-false", trusted.URL, []string{certificateFalseFlag}, true, boolChoice(false)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "trusted.pem", ca.PEM))
			_, before := readConnectionConfig(t, file)
			args := append([]string{certificateAddAction, certificateClusterCommand, test.name, "--url", test.address}, test.flags...)
			output, err := runCertificateCommand(t, file, args...)
			if (err == nil) != test.success {
				t.Fatalf("output %q, error %v", output, err)
			}
			_, after := readConnectionConfig(t, file)
			if !test.success {
				if strings.Contains(output, "Added cluster") || strings.Contains(output, setContextMsg) || !reflect.DeepEqual(before, after) {
					t.Fatalf("failed add persisted or reported success: %q", output)
				}
				return
			}
			if !strings.Contains(output, "Added cluster "+test.name) || !strings.Contains(output, setContextMsg+test.name) || after.CurrentContext != test.name {
				t.Fatalf("add/context output %q, config %#v", output, after)
			}
			connection := after.Clusters[len(after.Clusters)-1]
			if connection.Name != test.name || !reflect.DeepEqual(connection.IgnoreInvalidCerts, test.choice) || after.IgnoreInvalidCerts {
				t.Fatalf("saved choice: %#v", connection)
			}
		})
	}
	// Retain a previous manual flag in the singleton: discovery passes its own
	// explicit false and cannot inherit that flag, the context or the global env.
	if err := addClusterCmd.Flags().Set(certificateCommand, stringTrue); err != nil {
		t.Fatal(err)
	}
	before := readMigrationBytes(t, file)
	err := addCluster(&cobra.Command{}, "discovered", untrusted.URL, nameServiceDiscoveryType, certificateDiscoveryAddress, false)
	if err == nil || !strings.Contains(err.Error(), "certificate") || !bytes.Equal(before, readMigrationBytes(t, file)) {
		t.Fatalf("discovery inherited an exception: %v", err)
	}
}

func boolChoice(value bool) *bool { return &value }

func TestCertificateConfigDisplayAndFlagScope(t *testing.T) {
	file := certificateCommandConfig(t, certificateCommandFixture)
	output, err := runCertificateCommand(t, file, certificateGetAction, "config")
	if err != nil || strings.Contains(output, "Ignore Invalid Certs") {
		t.Fatalf("global setting still displayed: %q, %v", output, err)
	}
	output, err = runCertificateCommand(t, file, certificateGetAction, "config", "--verbose")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(output, "CLUSTER CONNECTIONS")
	if len(parts) != 2 || strings.Contains(parts[0], "Ignore Invalid Certs") {
		t.Fatalf("global setting in verbose output: %q", output)
	}
	for _, expected := range []string{stringFalse, stringFalse, stringTrue, na} {
		index := strings.Index(parts[1], "Ignore Invalid Certs")
		if index < 0 {
			t.Fatalf("missing connection setting: %q", output)
		}
		line, rest, _ := strings.Cut(parts[1][index:], "\n")
		if !strings.HasSuffix(line, ":  "+expected) {
			t.Fatalf("setting line %q; want %s", line, expected)
		}
		parts[1] = rest
	}
	output, err = runCertificateCommand(t, file, certificateGetAction, clusterKey)
	if err != nil || strings.Contains(output, "CERT") {
		t.Fatalf("cluster table changed: %q, %v", output, err)
	}
	if got := strings.Fields(strings.Split(output, "\n")[0]); !reflect.DeepEqual(got, []string{"CONNECTION", "TYPE", "URL", "VERSION", "CLUSTER", "NAME", "TYPE", "CTX", "CREATED"}) {
		t.Fatalf("table columns: %v", got)
	}
	root := Initialize(nil)
	if root.PersistentFlags().Lookup(certificateCommand) != nil || addClusterCmd.PersistentFlags().Lookup(certificateCommand) != nil {
		t.Fatal("exception flag has persistent scope")
	}
	for _, args := range [][]string{{certificateGetAction, clusterKey, certificateFlag}, {"discover", clusterKey, certificateFlag}} {
		if _, err = runCertificateCommand(t, file, args...); err == nil || !strings.Contains(err.Error(), "unknown flag") {
			t.Fatalf("flag accepted outside add cluster: %v", err)
		}
	}
}

func TestCertificateWriteFailures(t *testing.T) {
	file := certificateCommandConfig(t, certificateCommandFixture)
	if _, err := runCertificateCommand(t, file, certificateGetAction, "context"); err != nil {
		t.Fatal(err)
	}
	before := readMigrationBytes(t, file)
	oldConnections := append([]ClusterConnection(nil), Config.Clusters...)
	oldViper := viper.Get(clusterKey)
	// Writing to a directory deterministically fails on POSIX and Windows, even
	// for privileged test users. Keep the real fixture intact for comparison.
	viper.SetConfigFile(filepath.Dir(file))
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	if err := setIgnoreCertsCmd.RunE(command, []string{stringTrue}); err == nil || !strings.Contains(err.Error(), "write config") {
		t.Fatalf("set write failure: %v", err)
	}
	if output.Len() != 0 || !reflect.DeepEqual(Config.Clusters, oldConnections) || !reflect.DeepEqual(viper.Get(clusterKey), oldViper) {
		t.Fatal("failed setter published success/state")
	}
	server := connectionRequestServer(t, false)
	if err := addCluster(command, "failed-save", server.URL, "manual", "", false); err == nil {
		t.Fatal("add reported success on failed save")
	}
	if output.Len() != 0 || !reflect.DeepEqual(Config.Clusters, oldConnections) || !bytes.Equal(before, readMigrationBytes(t, file)) {
		t.Fatal("failed add persisted or reported success")
	}
}

func TestIgnoreCertsMigrationRemainsConnectionScoped(t *testing.T) {
	file := certificateCommandConfig(t, strings.Replace(certificateCommandFixture, "ignoreInvalidCerts: false", "ignoreInvalidCerts: true", 1))
	if output, err := runCertificateCommand(t, file, certificateGetAction, certificateCommand); err != nil || output != getIgnoreCertsMsg+"true\n" {
		t.Fatalf("migration: %q, %v", output, err)
	}
	if _, err := runCertificateCommand(t, file, certificateSetAction, certificateCommand, stringFalse); err != nil {
		t.Fatal(err)
	}
	if output, err := runCertificateCommand(t, file, certificateGetAction, certificateCommand); err != nil || output != getIgnoreCertsMsg+"false\n" {
		t.Fatalf("reload: %q, %v", output, err)
	}
	if backups := migrationBackups(t, file); len(backups) != 1 {
		t.Fatalf("setter triggered another migration: %v", backups)
	}
	_, loaded := readConnectionConfig(t, file)
	if loaded.IgnoreInvalidCerts {
		t.Fatalf("legacy true restored: %#v", loaded)
	}
}

func TestAddClusterWebLogicCertificateDefaults(t *testing.T) {
	file := certificateCommandConfig(t, "version: fixture\ncolor: off\n")
	ca := testtls.NewCA(t, "add-weblogic")
	t.Setenv("COHERENCE_TLS_CERTS_PATH", writeDownloadFixture(t, "weblogic.pem", ca.PEM))
	var requests atomic.Int32
	for _, useTLS := range []bool{false, true} {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "fixture-user" || pass != "fixture-password" || r.URL.Path != "/management/coherence/latest/clusters/" {
				t.Errorf("WebLogic request: user %q, pass %q, path %q", user, pass, r.URL.Path)
			}
			requests.Add(1)
			_, _ = io.WriteString(w, `{"items":[{"clusterName":"wls-fixture","version":"15.1.1"}]}`)
		}))
		if useTLS {
			server.TLS = &tls.Config{Certificates: []tls.Certificate{ca.Issue(t, localHost, false)}, MinVersion: tls.VersionTLS12}
			server.StartTLS()
		} else {
			server.Start()
		}
		defer server.Close()
		passwordFile := filepath.Join(t.TempDir(), "password")
		if err := os.WriteFile(passwordFile, []byte("fixture-password\n"), 0600); err != nil {
			t.Fatal(err)
		}
		input, err := os.Open(passwordFile)
		if err != nil {
			t.Fatal(err)
		}
		oldStdin := os.Stdin
		os.Stdin = input
		name := "weblogic-http"
		if useTLS {
			name = "weblogic-https"
		}
		output, err := runCertificateCommand(t, file, certificateAddAction, certificateClusterCommand, name, "-u", server.URL+"/management/coherence/latest/clusters", "--username", "fixture-user", "--stdin")
		os.Stdin = oldStdin
		_ = input.Close()
		if err != nil {
			t.Fatalf("WebLogic add: %s, %v", output, err)
		}
		_, loaded := readConnectionConfig(t, file)
		connection := loaded.Clusters[len(loaded.Clusters)-1]
		if connection.ClusterType != "WebLogic" || connection.ClusterName != "wls-fixture" || connection.ClusterVersion != "15.1.1" || loaded.CurrentContext != name {
			t.Fatalf("saved WebLogic connection: %#v", connection)
		}
		if useTLS && (connection.IgnoreInvalidCerts == nil || *connection.IgnoreInvalidCerts) {
			t.Fatal("trusted WebLogic HTTPS did not save verified choice")
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("WebLogic probes: %d", requests.Load())
	}
}

func TestAddClusterRejectedOptionsAndResponses(t *testing.T) {
	file := certificateCommandConfig(t, "version: fixture\ncolor: off\n")
	if _, err := runCertificateCommand(t, file, certificateGetAction, "context"); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, "invalid JSON")
	}))
	defer server.Close()
	before := readMigrationBytes(t, file)
	for _, args := range [][]string{
		{certificateAddAction, certificateClusterCommand},
		{certificateAddAction, certificateClusterCommand, "extra", "args", "-u", server.URL},
		{certificateAddAction, certificateClusterCommand, "bad-value", "-u", server.URL, "--ignore-certs=maybe"},
		{certificateAddAction, certificateClusterCommand, "bad-url", "-u", "https://host:0/", certificateFlag},
		{certificateAddAction, certificateClusterCommand, "bad-json", "-u", server.URL},
	} {
		output, err := runCertificateCommand(t, file, args...)
		if err == nil || strings.Contains(output, "Added cluster") || !bytes.Equal(before, readMigrationBytes(t, file)) {
			t.Fatalf("rejected add: %q, %v", output, err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("rejected options sent requests: %d", requests.Load())
	}
}

func TestIgnoreCertsSetterPreservesSavedEntries(t *testing.T) {
	file := certificateCommandConfig(t, migrationFixture)
	if _, err := runCertificateCommand(t, file, certificateGetAction, certificateCommand); err != nil {
		t.Fatal(err)
	}
	before := decodeMigrationFixture(t, "yaml", readMigrationBytes(t, file))
	if _, err := runCertificateCommand(t, file, certificateSetAction, certificateCommand, stringFalse); err != nil {
		t.Fatal(err)
	}
	after := decodeMigrationFixture(t, "yaml", readMigrationBytes(t, file))
	before[clusterKey].([]any)[0].(map[string]any)["ignoreinvalidcerts"] = false
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("setter changed unrelated saved data:\nbefore %#v\nafter %#v", before, after)
	}
	if backups := migrationBackups(t, file); len(backups) != 1 {
		t.Fatalf("setter repeated migration: %v", backups)
	}
}

func TestIgnoreCertsSetterRejectsChangedSavedTarget(t *testing.T) {
	file := certificateCommandConfig(t, certificateCommandFixture)
	if _, err := runCertificateCommand(t, file, certificateGetAction, certificateCommand); err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(readMigrationBytes(t, file), []byte("https://unreachable.invalid:30000/"), []byte("http://changed.invalid/"), 1)
	if err := os.WriteFile(file, changed, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	err := setIgnoreCertsCmd.RunE(command, []string{stringTrue})
	if err == nil || !strings.Contains(err.Error(), "changed; reload") || output.Len() != 0 || !bytes.Equal(changed, readMigrationBytes(t, file)) || Config.Clusters[0].IgnoreInvalidCerts != nil {
		t.Fatalf("externally changed target accepted: %s, %v", &output, err)
	}
}

// All formats use regular files. Inject the external edit after the initial read
// has captured its bytes but before validation/mutation can use that snapshot.
// This controls the interleaving without timing, FIFOs or platform-specific APIs.
func TestIgnoreCertsSetterRejectsConcurrentEdit(t *testing.T) {
	for _, format := range []string{configType, certificateJSONFormat, certificateTOMLFormat} {
		for _, change := range []string{"revoke-other", "change-target"} {
			t.Run(format+"/"+change, func(t *testing.T) {
				file := certificateUpdateFixture(t, format)
				before := append([]ClusterConnection(nil), Config.Clusters...)
				beforeViper := viper.Get(clusterKey)
				external := decodeMigrationFixture(t, format, readMigrationBytes(t, file))
				entries := external["Clusters"].([]any)
				if change == "revoke-other" {
					entries[1].(map[string]any)["IgnoreInvalidCerts"] = false
				} else {
					entries[0].(map[string]any)["ConnectionURL"] = "https://changed.invalid/"
				}
				external[certificateExtensionKey].(map[string]any)["Owner"] = "external-editor"
				encoder, err := viper.NewCodecRegistry().Encoder(format)
				if err != nil {
					t.Fatal(err)
				}
				edited, err := encoder.Encode(external)
				if err != nil {
					t.Fatal(err)
				}
				ops := defaultConfigMigrationOps()
				read, injected := ops.readFile, false
				ops.readFile = func(path string) ([]byte, error) {
					contents, readErr := read(path)
					if readErr == nil && !injected {
						injected = true
						readErr = os.WriteFile(file, edited, 0600)
					}
					return contents, readErr
				}
				err = saveConnectionCertificateChoiceWithOps(before[0], true, ops)
				if !injected || err == nil {
					t.Fatalf("stale setter was not rejected: injected %t, error %v", injected, err)
				}
				if !bytes.Equal(edited, readMigrationBytes(t, file)) {
					t.Fatal("setter overwrote the external edit")
				}
				if !reflect.DeepEqual(Config.Clusters, before) || !reflect.DeepEqual(viper.Get(clusterKey), beforeViper) {
					t.Fatal("rejected setter published live state")
				}
				staged, globErr := filepath.Glob(filepath.Join(filepath.Dir(file), "."+filepath.Base(file)+".config-stage-*"))
				if globErr != nil || len(staged) != 1 {
					t.Fatalf("missing recoverable staging file: %v, %v", staged, globErr)
				}
				want := fmt.Sprintf("write setting %q in %q; staging file %q may be retained: configuration changed during update; retry after reviewing it", clusterKey, file, staged[0])
				if err.Error() != want {
					t.Fatalf("conflict diagnostic: %v; want %s", err, want)
				}
				assertMigrationPermissions(t, staged[0], 0600)
			})
		}
	}
}

func TestIgnoreCertsSetterWrapsPersistenceError(t *testing.T) {
	file := certificateUpdateFixture(t, configType)
	injected := errors.New("injected config restriction failure")
	var staged string
	ops := defaultConfigMigrationOps()
	ops.restrict = func(file migrationFile, _ os.FileMode) error {
		staged = file.Name()
		return injected
	}
	err := saveConnectionCertificateChoiceWithOps(Config.Clusters[0], true, ops)
	if staged == "" || !errors.Is(err, injected) {
		t.Fatalf("persistence failure lost its stage or cause: %q, %v", staged, err)
	}
	want := fmt.Sprintf("write setting %q in %q; staging file %q may be retained: %v", clusterKey, file, staged, injected)
	if err.Error() != want {
		t.Fatalf("persistence diagnostic: %v; want %s", err, want)
	}
	if _, statErr := os.Stat(staged); statErr != nil {
		t.Fatalf("missing retained staging file: %v", statErr)
	}
}

func TestIgnoreCertsSetterLatePersistenceFailures(t *testing.T) {
	for _, phase := range []string{migrationReplace, migrationAfterSync} {
		t.Run(phase, func(t *testing.T) {
			file := certificateUpdateFixture(t, configType)
			original := readMigrationBytes(t, file)
			want := decodeMigrationFixture(t, configType, original)
			want["Clusters"].([]any)[0].(map[string]any)["IgnoreInvalidCerts"] = true
			// Serialize both live representations before the call so later map,
			// slice or pointer mutations cannot also change the expected snapshot.
			before, err := json.Marshal([]any{Config, viper.AllSettings()})
			if err != nil {
				t.Fatal(err)
			}
			injected := errors.New("injected ordinary " + phase + " failure")
			ops := defaultConfigMigrationOps()
			replace := ops.replace
			var staged string
			ops.replace = func(stage, target string) error {
				staged = stage
				if phase == migrationReplace {
					return injected
				}
				return replace(stage, target)
			}
			syncDir, calls := ops.syncDir, 0
			ops.syncDir = func(dir string) error {
				calls++
				if phase == migrationAfterSync && calls == 2 {
					return injected
				}
				return syncDir(dir)
			}
			err = saveConnectionCertificateChoiceWithOps(Config.Clusters[0], true, ops)
			if staged == "" || !errors.Is(err, injected) {
				t.Fatalf("late failure lost its stage or cause: %q, %v", staged, err)
			}
			diagnostic := fmt.Sprintf("write setting %q in %q; staging file %q may be retained: %v", clusterKey, file, staged, injected)
			if err.Error() != diagnostic {
				t.Fatalf("late failure diagnostic: %v; want %s", err, diagnostic)
			}
			after, marshalErr := json.Marshal([]any{Config, viper.AllSettings()})
			if marshalErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("failed setter published Config/Viper state: before %s, after %s, error %v", before, after, marshalErr)
			}
			candidate := file
			if phase == migrationReplace {
				if calls != 1 || !bytes.Equal(original, readMigrationBytes(t, file)) {
					t.Fatal("replacement failure changed the original or reached the final directory sync")
				}
				candidate = staged
			} else if calls != 2 || bytes.Equal(original, readMigrationBytes(t, file)) {
				t.Fatal("expected installed replacement before final directory sync failure")
			}
			// After replacement the stage pathname need not exist. Check the
			// installed file in that phase; a sync error does not imply rollback.
			assertMigrationPermissions(t, candidate, 0600)
			if got := decodeMigrationFixture(t, configType, readMigrationBytes(t, candidate)); !reflect.DeepEqual(got, want) {
				t.Fatalf("candidate changed unrelated saved data: got %#v, want %#v", got, want)
			}
			if backups := migrationBackups(t, file); len(backups) != 0 {
				t.Fatalf("ordinary setter created migration backups: %v", backups)
			}
		})
	}
}

func TestIgnoreCertsSetterPreservesFormats(t *testing.T) {
	for _, format := range []string{configType, certificateJSONFormat, certificateTOMLFormat} {
		t.Run(format, func(t *testing.T) {
			file := certificateUpdateFixture(t, format)
			before := decodeMigrationFixture(t, format, readMigrationBytes(t, file))
			output, err := runCertificateCommand(t, file, certificateSetAction, certificateCommand, stringTrue)
			if err != nil || output != setIgnoreCertsMsg+"true\n" {
				t.Fatalf("setter: %q, %v", output, err)
			}
			before["Clusters"].([]any)[0].(map[string]any)["IgnoreInvalidCerts"] = true
			after := decodeMigrationFixture(t, format, readMigrationBytes(t, file))
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("setter changed unrelated saved data: before %#v, after %#v", before, after)
			}
			assertMigrationPermissions(t, file, 0600)
		})
	}
}

func TestStartupConfigValuePreservesFormats(t *testing.T) {
	for _, format := range []string{configType, certificateJSONFormat, certificateTOMLFormat} {
		t.Run(format, func(t *testing.T) {
			file := certificateUpdateFixture(t, format)
			before := decodeMigrationFixture(t, format, readMigrationBytes(t, file))
			if err := writeStartupConfigValue(viper.GetViper(), colorContextKey, on); err != nil {
				t.Fatal(err)
			}
			if err := writeStartupConfigValue(viper.GetViper(), migrationVersionKey, migrationStartupVersion); err != nil {
				t.Fatal(err)
			}
			before[colorContextKey], before[migrationVersionKey] = on, migrationStartupVersion
			after := decodeMigrationFixture(t, format, readMigrationBytes(t, file))
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("startup changed unrelated saved data: before %#v, after %#v", before, after)
			}
		})
	}
}

func certificateUpdateFixture(t *testing.T, format string) string {
	t.Helper()
	file := certificateCommandConfig(t, "")
	file = filepath.Join(filepath.Dir(file), "selected."+format)
	extension := map[string]any{"Owner": migrationConnectionName, "literal.dot": []any{"kept", 47}}
	if format != certificateTOMLFormat {
		extension["Empty"] = nil
	}
	data := map[string]any{
		migrationVersionKey: Version, colorContextKey: off, "currentContext": certificateTargetName, "ignoreInvalidCerts": false,
		certificateExtensionKey: extension,
		"Clusters": []any{
			map[string]any{"Name": certificateTargetName, "ConnectionURL": "https://target.invalid/", "IgnoreInvalidCerts": false, "FutureData": extension},
			map[string]any{"Name": "other", "ConnectionURL": "https://other.invalid/", "IgnoreInvalidCerts": true},
		},
	}
	encoder, err := viper.NewCodecRegistry().Encoder(format)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := encoder.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(file, contents, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = runCertificateCommand(t, file, certificateGetAction, certificateCommand); err != nil {
		t.Fatal(err)
	}
	return file
}
