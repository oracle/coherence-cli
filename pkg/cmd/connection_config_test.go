/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

const (
	connectionTestStandalone   = "Standalone"
	connectionTestStartupClass = "com.tangosol.net.Coherence"
)

func TestClusterConnectionIgnoreInvalidCertsRoundTrip(t *testing.T) {
	previous := *viper.GetViper()
	t.Cleanup(func() { *viper.GetViper() = previous })
	falseValue, trueValue := false, true
	for _, test := range []struct {
		name  string
		yaml  string
		value *bool
	}{
		{name: "absent"},
		{name: "false", yaml: "    ignoreInvalidCerts: false\n", value: &falseValue},
		{name: stringTrue, yaml: "    ignoreInvalidCerts: true\n", value: &trueValue},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "cohctl.yaml")
			contents := `version: fixture-version
currentContext: secure
debug: true
color: off
requestTimeout: 47
ignoreInvalidCerts: false
defaultBytesFormat: g
defaultHeap: 2g
useGradle: true
defaultStyle: compact
profiles:
  - name: storage
    value: "-Dcoherence.distributed.localstorage=true"
panels:
  - name: custom
    layout: "cluster,members"
extension:
  owner: fixture
clusters:
  - name: secure
    discoverytype: manual
    connectiontype: http
    connectionurl: https://localhost:30000/management/coherence/cluster
    nameservicediscovery: localhost:7574
    clusterversion: 15.1.1
    clustername: example
    clustertype: Standalone
    manuallycreated: true
    baseclasspath: coherence.jar
    additionalclasspath: custom.jar
    arguments: "-Dcoherence.cluster=example"
    managementport: 30000
    persistencemode: active
    loggingdestination: logs
    managementavailable: true
    startupclass: com.tangosol.net.Coherence
` + test.yaml
			if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			reader, got := readConnectionConfig(t, file)
			want := CoherenceCLIConfig{
				Version: "fixture-version", CurrentContext: "secure", Debug: true, Color: "off",
				RequestTimeout: 47, DefaultBytesFormat: "g", DefaultHeap: "2g", UseGradle: true,
				DefaultStyle: "compact",
				Profiles:     []ProfileValue{{Name: "storage", Value: "-Dcoherence.distributed.localstorage=true"}},
				Panels:       []Panel{{Name: "custom", Layout: "cluster,members"}},
				Clusters: []ClusterConnection{{
					Name: "secure", DiscoveryType: "manual", ConnectionType: httpType,
					ConnectionURL:        "https://localhost:30000/management/coherence/cluster",
					NameServiceDiscovery: "localhost:7574", ClusterVersion: "15.1.1", ClusterName: "example",
					ClusterType: connectionTestStandalone, IgnoreInvalidCerts: test.value, ManuallyCreated: true,
					BaseClasspath: "coherence.jar", AdditionalClasspath: "custom.jar",
					Arguments: "-Dcoherence.cluster=example", ManagementPort: 30000, PersistenceMode: "active",
					LoggingDestination: "logs", ManagementAvailable: true, StartupClass: connectionTestStartupClass,
				}},
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("loaded configuration = %#v, want %#v", got, want)
			}

			// Exercise addCluster's configuration encoder as well as Viper's codec.
			// A fresh reader on each pass prevents an old pointer from masking data loss.
			for pass := 1; pass <= 2; pass++ {
				*viper.GetViper() = *reader
				viper.Set(clusterKey, got.Clusters)
				if err := WriteConfig(); err != nil {
					t.Fatal(err)
				}
				reader, got = readConnectionConfig(t, file)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("round trip %d: configuration = %#v, want %#v", pass, got, want)
				}
				if reader.GetString("extension.owner") != "fixture" {
					t.Fatalf("round trip %d: unrelated extension was lost", pass)
				}
				entries, ok := reader.Get(clusterKey).([]any)
				if !ok || len(entries) != 1 {
					t.Fatalf("unexpected serialized clusters: %#v", reader.Get(clusterKey))
				}
				entry, ok := entries[0].(map[string]any)
				if !ok {
					t.Fatalf("unexpected serialized connection: %#v", entries[0])
				}
				value, present := entry["ignoreinvalidcerts"]
				if test.value == nil {
					if present {
						t.Fatalf("unset choice must remain absent, got %#v", value)
					}
				} else if !present || value != *test.value {
					t.Fatalf("serialized choice = %#v (present %v), want %v", value, present, *test.value)
				}
			}
		})
	}
}

func TestClusterConnectionIgnoreInvalidCertsLegacyAndNewDefaults(t *testing.T) {
	if (ClusterConnection{}).IgnoreInvalidCerts != nil {
		t.Fatal("a default connection must leave the certificate exception unset")
	}
	for _, test := range []struct {
		name  string
		yaml  string
		value bool
	}{
		{name: "absent"},
		{name: "false", yaml: "ignoreInvalidCerts: false\n"},
		{name: stringTrue, yaml: "ignoreInvalidCerts: true\n", value: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "cohctl.yaml")
			contents := test.yaml + `currentContext: legacy-http
clusters:
  - name: legacy-http
    connectiontype: http
    connectionurl: http://localhost:30000/management/coherence/cluster
  - name: legacy-https
    connectiontype: http
    connectionurl: https://localhost:30001/management/coherence/cluster
`
			if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			reader, got := readConnectionConfig(t, file)
			for _, scheme := range []string{httpType, "https"} {
				got.Clusters = append(got.Clusters, ClusterConnection{
					Name: "new-" + scheme, ConnectionType: httpType,
					ConnectionURL: scheme + "://localhost:30002/management/coherence/cluster",
				})
			}
			want := got
			reader.Set(clusterKey, got.Clusters)
			if err := reader.WriteConfig(); err != nil {
				t.Fatal(err)
			}
			_, got = readConnectionConfig(t, file)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("legacy/new connections changed after save/reload: got %#v, want %#v", got, want)
			}
			if got.IgnoreInvalidCerts != test.value {
				t.Fatal("saving and reloading must preserve the legacy global value")
			}
			for _, connection := range got.Clusters {
				if connection.IgnoreInvalidCerts != nil {
					t.Fatalf("connection %q acquired a certificate exception: %v", connection.Name, *connection.IgnoreInvalidCerts)
				}
			}
		})
	}
}

func TestClusterConnectionIgnoreInvalidCertsIndependentValues(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cohctl.yaml")
	contents := `ignoreInvalidCerts: true
clusters:
  - name: unset
    connectiontype: http
    connectionurl: https://localhost:30000/management/coherence/cluster
  - name: verified
    connectiontype: http
    connectionurl: https://localhost:30001/management/coherence/cluster
    ignoreInvalidCerts: false
  - name: exception
    connectiontype: http
    connectionurl: https://localhost:30002/management/coherence/cluster
    ignoreInvalidCerts: true
`
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	reader, got := readConnectionConfig(t, file)
	if len(got.Clusters) != 3 || got.Clusters[0].IgnoreInvalidCerts != nil ||
		got.Clusters[1].IgnoreInvalidCerts == nil || *got.Clusters[1].IgnoreInvalidCerts ||
		got.Clusters[2].IgnoreInvalidCerts == nil || !*got.Clusters[2].IgnoreInvalidCerts {
		t.Fatalf("independent choices did not load: %#v", got.Clusters)
	}
	want := got
	reader.Set(clusterKey, got.Clusters)
	if err := reader.WriteConfig(); err != nil {
		t.Fatal(err)
	}
	reader, got = readConnectionConfig(t, file)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("independent choices changed after save/reload: got %#v, want %#v", got, want)
	}
	*got.Clusters[1].IgnoreInvalidCerts = true
	*got.Clusters[2].IgnoreInvalidCerts = false
	want = got
	reader.Set(clusterKey, got.Clusters)
	if err := reader.WriteConfig(); err != nil {
		t.Fatal(err)
	}
	_, got = readConnectionConfig(t, file)
	if !reflect.DeepEqual(got, want) || got.Clusters[1].IgnoreInvalidCerts == nil ||
		!*got.Clusters[1].IgnoreInvalidCerts || got.Clusters[2].IgnoreInvalidCerts == nil || *got.Clusters[2].IgnoreInvalidCerts {
		t.Fatalf("updated choices were not saved independently: %#v", got.Clusters)
	}
}

func readConnectionConfig(t *testing.T, file string) (*viper.Viper, CoherenceCLIConfig) {
	t.Helper()
	reader := viper.New()
	reader.SetConfigFile(file)
	if err := reader.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	var config CoherenceCLIConfig
	if err := reader.Unmarshal(&config); err != nil {
		t.Fatal(err)
	}
	return reader, config
}
