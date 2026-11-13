/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	persistenceRenamePhase = "rename"

	persistenceWritePhase     = "write"
	persistenceSyncPhase      = "sync"
	persistenceClosePhase     = "close"
	persistenceDanglingPhase  = "dangling-link"
	persistenceDirectoryPhase = "directory-target"
)

func managedPersistenceState(t *testing.T) {
	t.Helper()
	// Preserve the complete Viper value, including explicit type and bindings.
	// managedTestState resets the singleton before any fixture changes it.
	previous := viper.GetViper()
	t.Cleanup(func() { *viper.GetViper() = *previous })
	managedTestState(t)
	create, rename := createManagedConfigTemp, renameManagedConfig
	t.Cleanup(func() { createManagedConfigTemp, renameManagedConfig = create, rename })
}

func readManagedPersistence(t *testing.T, filename, format string) (*viper.Viper, CoherenceCLIConfig) {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if format == managementTestJSON && !json.Valid(data) {
		t.Fatalf("selected JSON file is no longer JSON: %s", data)
	}
	reader := viper.New()
	reader.SetConfigFile(filename)
	reader.SetConfigType(format)
	if err := reader.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	var loaded CoherenceCLIConfig
	if err := reader.Unmarshal(&loaded); err != nil {
		t.Fatal(err)
	}
	return reader, loaded
}

func persistenceFixture(t *testing.T, filename, format, explicitType string) ClusterConnection {
	t.Helper()
	external := ClusterConnection{Name: "external", ConnectionURL: "https://remote.example/management", ClusterName: "remote"}
	settings := map[string]any{
		clusterKey: []ClusterConnection{external}, currentContextKey: external.Name,
		"preferences": map[string]any{"theme": "green", "pins": []string{"one", "two"}},
		"profiles":    []ProfileValue{{Name: "portable", Value: "-Dcustom=true"}}, "requestTimeout": 41,
	}
	var data []byte
	var err error
	data, err = yaml.Marshal(settings)
	if err == nil && format == managementTestJSON {
		// Existing CLI configuration uses connectionurl/clusterversion, unlike
		// the public JSON output's url/clusterVersionParam field names.
		var canonical map[string]any
		if err = yaml.Unmarshal(data, &canonical); err == nil {
			data, err = json.Marshal(canonical)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	viper.SetConfigFile(filename)
	viper.SetConfigType(explicitType)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if err := viper.Unmarshal(&Config); err != nil {
		t.Fatal(err)
	}
	return external
}

func assertNoManagedReplacement(t *testing.T, filename string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(filename), ".cohctl-management-*"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("stale replacements: %v (%v)", paths, err)
	}
}

// initConfig creates the default file after its initial search failed. The
// ordinary writer must retain Viper's discovered filename for a managed save
// in that same first CLI invocation.
func TestConfigurationWriterFirstUseSelection(t *testing.T) {
	for _, format := range []string{configType, managementTestJSON} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			dir := t.TempDir()
			filename := filepath.Join(dir, configName+"."+format)
			viper.AddConfigPath(dir)
			viper.SetConfigName(configName)
			viper.SetConfigType(format)
			if err := viper.ReadInConfig(); err == nil {
				t.Fatal("first-use fixture unexpectedly found a configuration")
			}
			if viper.ConfigFileUsed() != "" {
				t.Fatal("failed search selected a file")
			}
			if err := os.WriteFile(filename, nil, 0600); err != nil {
				t.Fatal(err)
			}
			viper.Set(colorContextKey, on)
			viper.Set(clusterKey, []ClusterConnection{})
			if err := WriteConfig(); err != nil {
				t.Fatal(err)
			}
			if viper.ConfigFileUsed() != filename {
				t.Fatalf("ordinary first write lost selected file: got %q want %q", viper.ConfigFileUsed(), filename)
			}
			connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ClusterName: managementTestConnectionName,
				ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{RESTAddress: localHost})
			if err := persistManagedConnection(connection, true); err != nil {
				t.Fatalf("managed save in first invocation: %v", err)
			}
			_, loaded := readManagedPersistence(t, filename, format)
			if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{connection}) || loaded.CurrentContext != connection.Name {
				t.Fatalf("first-use managed record: %+v", loaded)
			}
			assertNoManagedReplacement(t, filename)
		})
	}
}

func TestManagementPersistenceSelectedFormat(t *testing.T) {
	for _, tc := range []struct{ name, filename, format, explicitType string }{
		{configType, "cohctl.yaml", configType, ""},
		{ordinaryYML, "cohctl.yml", configType, ""},
		{managementTestJSON, "cohctl.json", managementTestJSON, ""},
		{"explicit-extensionless", "cohctl", managementTestJSON, managementTestJSON},
		{"explicit-json-over-extension", "cohctl.yaml", managementTestJSON, managementTestJSON},
		{"explicit-yaml-over-extension", "cohctl.json", configType, configType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), tc.filename)
			external := persistenceFixture(t, filename, tc.format, tc.explicitType)
			beforePreferences := viper.Get("preferences")
			connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ClusterName: managementTestConnectionName, ClusterVersion: "14.1.2-0-5", ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{localHost, 9999, localHost, loopbackHostname})
			if err := persistManagedConnection(connection, true); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filename)
			if err != nil || strings.Contains(string(data), "allow-insecure") {
				t.Fatalf("consent saved or unreadable config: %v", err)
			}

			reader, loaded := readManagedPersistence(t, filename, tc.explicitType)
			// Infer the parser type independently when no explicit override exists.
			if tc.format == managementTestJSON {
				data, _ := os.ReadFile(filename)
				if !json.Valid(data) {
					t.Fatalf("JSON replaced with another format: %s", data)
				}
			}
			if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{external, connection}) || loaded.CurrentContext != connection.Name || !reflect.DeepEqual(loaded, Config) {
				t.Fatalf("create round trip: %+v; memory: %+v", loaded, Config)
			}
			if !reflect.DeepEqual(reader.Get("preferences"), beforePreferences) || reader.GetInt("requestTimeout") != 41 || !reflect.DeepEqual(loaded.Profiles, []ProfileValue{{Name: "portable", Value: "-Dcustom=true"}}) {
				t.Fatal("unrelated settings changed")
			}
			// A later ordinary external connection and its active context survive
			// an update of the managed connection, including explicit JMX disable.
			later := ClusterConnection{Name: "later-external", ConnectionURL: "https://later.example/management"}
			Config.Clusters = append(Config.Clusters, later)
			Config.CurrentContext = later.Name
			viper.Set(clusterKey, Config.Clusters)
			viper.Set(currentContextKey, later.Name)
			updated := testManagedConnection(t, connection, ClusterManagement{RESTAddress: managementTestIP})
			if err := persistManagedConnection(updated, false); err != nil {
				t.Fatal(err)
			}
			reader, loaded = readManagedPersistence(t, filename, tc.explicitType)
			if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{external, updated, later}) || loaded.CurrentContext != later.Name || !reflect.DeepEqual(loaded, Config) || !reflect.DeepEqual(reader.Get("preferences"), beforePreferences) {
				t.Fatalf("update round trip: %+v; memory: %+v", loaded, Config)
			}
			if viper.ConfigFileUsed() != filename {
				t.Fatal("save retargeted Viper to its temporary file")
			}
			info, err := os.Stat(filename)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != managementTestWindows && info.Mode().Perm() != 0600 {
				t.Fatalf("replacement permissions: %v", info.Mode())
			}
			assertNoManagedReplacement(t, filename)
		})
	}
}

func TestManagementPersistencePreservesSelectedSymlink(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("Unix symlink replacement regression")
	}
	for _, tc := range []struct {
		name, format string
		relative     bool
	}{
		{"relative-yaml", configType, true},
		{"absolute-json", managementTestJSON, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedPersistenceState(t)
			root := t.TempDir()
			filename := filepath.Join(root, "selected", "cohctl."+tc.format)
			// The selected filename supplies the format, even when the target
			// has a different extension and lives in another directory.
			target := filepath.Join(root, "targets", "settings.data")
			for _, path := range []string{filename, target} {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
			}
			linkTarget := target
			if tc.relative {
				var err error
				linkTarget, err = filepath.Rel(filepath.Dir(filename), target)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(linkTarget, filename); err != nil {
				t.Fatal(err)
			}
			external := persistenceFixture(t, filename, tc.format, "")
			if viper.ConfigFileUsed() != filename {
				t.Fatal("fixture did not select the symlink")
			}
			connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ClusterName: managementTestConnectionName,
				ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{managementTestIP, 9999, localHost, loopbackHostname})
			if err := persistManagedConnection(connection, true); err != nil {
				t.Fatal(err)
			}
			info, err := os.Lstat(filename)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				t.Errorf("selected configuration symlink was replaced: mode %v", info.Mode())
			} else if got, err := os.Readlink(filename); err != nil || got != linkTarget {
				t.Errorf("selected configuration symlink target: got %q (%v), want %q", got, err, linkTarget)
			}
			_, loaded := readManagedPersistence(t, target, tc.format)
			if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{external, connection}) || loaded.CurrentContext != connection.Name {
				t.Errorf("symlink target did not receive saved management settings %+v and URL %q; got %+v", *connection.Management, connection.ConnectionURL, loaded)
			}
			assertNoManagedReplacement(t, filename)
			assertNoManagedReplacement(t, target)
		})
	}
}

func TestManagementPersistenceSymlinkChain(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("Unix symlink persistence")
	}
	managedPersistenceState(t)
	root := t.TempDir()
	filename := filepath.Join(root, "selected", "cohctl.yaml")
	hop := filepath.Join(root, "links", "hop")
	target := filepath.Join(root, "targets", "settings.data")
	for _, path := range []string{filename, hop, target} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{filename: "../links/hop", hop: "../targets/settings.data"}
	for path, text := range links {
		if err := os.Symlink(text, path); err != nil {
			t.Fatal(err)
		}
	}
	// An explicit type still overrides the selected extension through a chain.
	persistenceFixture(t, filename, managementTestJSON, managementTestJSON)
	createManagedConfigTemp = func(dir, pattern string) (managedConfigFile, error) {
		if dir != filepath.Dir(target) {
			t.Fatalf("replacement directory: got %q, want %q", dir, filepath.Dir(target))
		}
		return os.CreateTemp(dir, pattern)
	}
	renameManagedConfig = func(from, to string) error {
		if to != target || filepath.Dir(from) != filepath.Dir(target) {
			t.Fatalf("replacement must be beside resolved target: %q -> %q", from, to)
		}
		return os.Rename(from, to)
	}
	connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{RESTAddress: managementTestIP})
	if err := persistManagedConnection(connection, true); err != nil {
		t.Fatal(err)
	}
	_, loaded := readManagedPersistence(t, target, managementTestJSON)
	if !reflect.DeepEqual(loaded, Config) || loaded.CurrentContext != connection.Name || viper.ConfigFileUsed() != filename {
		t.Fatalf("chain save changed selected path or configuration: %+v", loaded)
	}
	for path, text := range links {
		if got, err := os.Readlink(path); err != nil || got != text {
			t.Fatalf("chain link changed: %q -> %q (%v), want %q", path, got, err, text)
		}
		assertNoManagedReplacement(t, path)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("replacement permissions: %v (%v)", info, err)
	}
	assertNoManagedReplacement(t, target)
}

func TestManagementPersistenceSelectedFileRemoved(t *testing.T) {
	managedPersistenceState(t)
	filename := filepath.Join(t.TempDir(), "cohctl.json")
	persistenceFixture(t, filename, managementTestJSON, "")
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{RESTAddress: localHost})
	if err := persistManagedConnection(connection, true); err != nil {
		t.Fatal(err)
	}
	_, loaded := readManagedPersistence(t, filename, managementTestJSON)
	if !reflect.DeepEqual(loaded, Config) || loaded.CurrentContext != connection.Name || viper.ConfigFileUsed() != filename {
		t.Fatalf("removed selected file did not retain ordinary save behavior: %+v", loaded)
	}
	assertNoManagedReplacement(t, filename)
}

func TestManagementPersistenceOtherFormats(t *testing.T) {
	for _, format := range []string{ordinaryTOML, "env", "dotenv"} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), "cohctl."+format)
			data := "OWNER=original\n"
			if format == ordinaryTOML {
				data = "owner = 'original'\n"
			}
			if err := os.WriteFile(filename, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			viper.SetConfigFile(filename)
			if err := viper.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			viper.Set("owner", "updated")
			if err := writeManagedConfigAtomically(); err != nil {
				t.Fatal(err)
			}
			reader, _ := readManagedPersistence(t, filename, "")
			if reader.GetString("owner") != "updated" {
				t.Fatal("selected codec did not round trip")
			}
			assertNoManagedReplacement(t, filename)
		})
	}
}

// Failures below are injected around a real isolated temporary file. A partial
// write reaches disk; sync/close/rename failures are simulated, not claims about
// native device failures or Windows sharing semantics.
type failingManagedConfigFile struct {
	*os.File
	phase string
	err   error
}

func (f *failingManagedConfigFile) Write(data []byte) (int, error) {
	if f.phase == persistenceWritePhase {
		n, err := f.File.Write(data[:len(data)/2])
		if err != nil {
			return n, err
		}
		return n, f.err
	}
	return f.File.Write(data)
}

func (f *failingManagedConfigFile) Sync() error {
	if f.phase == persistenceSyncPhase {
		return f.err
	}
	return f.File.Sync()
}

func (f *failingManagedConfigFile) Close() error {
	err := f.File.Close()
	if f.phase == persistenceClosePhase {
		return f.err
	}
	return err
}

func TestManagementPersistenceFailureRollback(t *testing.T) {
	for _, tc := range []struct {
		phase  string
		create bool
	}{
		{"serialize", false}, {"codec", false}, {managementTestCreate, false}, {persistenceWritePhase, false}, {persistenceSyncPhase, false}, {persistenceClosePhase, false}, {persistenceRenamePhase, false}, {persistenceRenamePhase, true},
	} {
		phase, create := tc.phase, tc.create
		t.Run(fmt.Sprintf("create-%t/%s", create, phase), func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), "cohctl.json")
			persistenceFixture(t, filename, managementTestJSON, "")
			original := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{localHost, 9999, localHost, loopbackHostname})
			if err := persistManagedConnection(original, true); err != nil {
				t.Fatal(err)
			}
			changed := testManagedConnection(t, original, ClusterManagement{RESTAddress: managementTestIP})
			if create {
				changed.Name = managementTestNewConnectionName
			}
			sentinel := errors.New("injected " + phase + " failure")
			var replacement *os.File
			createManagedConfigTemp = func(dir, pattern string) (managedConfigFile, error) {
				var err error
				replacement, err = os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				return &failingManagedConfigFile{replacement, phase, sentinel}, nil
			}
			switch phase {
			case "serialize":
				// Real JSON encoder rejection, before any target replacement.
				viper.Set("unserializable", make(chan int))
			case "codec":
				// Listed by Viper, but no built-in ini codec is registered.
				viper.SetConfigType("ini")
			case managementTestCreate:
				createManagedConfigTemp = func(dir, pattern string) (managedConfigFile, error) {
					return os.CreateTemp(filepath.Join(dir, "missing"), pattern)
				}
			case persistenceRenamePhase:
				renameManagedConfig = func(_, _ string) error { return sentinel }
			}
			before, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			oldConfig, oldSettings := Config, viper.AllSettings()
			err = persistManagedConnection(changed, create)
			if err == nil || !strings.Contains(err.Error(), "previous configuration retained") {
				t.Fatalf("failure ignored: %v", err)
			}
			if (phase == persistenceSyncPhase || phase == persistenceClosePhase || phase == persistenceRenamePhase) && !errors.Is(err, sentinel) {
				t.Fatalf("lost failure: %v", err)
			}
			if phase == persistenceWritePhase && !strings.Contains(err.Error(), sentinel.Error()) {
				t.Fatalf("lost Viper write failure: %v", err)
			}
			after, err := os.ReadFile(filename)
			if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config, oldConfig) || !reflect.DeepEqual(viper.AllSettings(), oldSettings) || viper.ConfigFileUsed() != filename {
				t.Fatalf("failure changed prior state: %v", err)
			}
			if replacement != nil {
				if _, err := replacement.Write([]byte("handle must be closed")); !errors.Is(err, os.ErrClosed) {
					t.Fatalf("replacement handle retained: %v", err)
				}
			}
			assertNoManagedReplacement(t, filename)
			readManagedPersistence(t, filename, managementTestJSON)
		})
	}
}

func TestManagementPersistenceSymlinkFailureRollback(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("Unix symlink persistence")
	}
	for _, phase := range []string{persistenceDanglingPhase, "link-cycle", "non-directory", persistenceDirectoryPhase, persistenceWritePhase, persistenceRenamePhase} {
		t.Run(phase, func(t *testing.T) {
			managedPersistenceState(t)
			root := t.TempDir()
			filename := filepath.Join(root, "selected", "cohctl.json")
			target := filepath.Join(root, "targets", "settings.data")
			for _, path := range []string{filename, target} {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
			}
			linkTarget := "../targets/settings.data"
			if err := os.Symlink(linkTarget, filename); err != nil {
				t.Fatal(err)
			}
			persistenceFixture(t, filename, managementTestJSON, "")
			sentinel := errors.New("injected " + phase + " failure")
			var replacement *os.File
			createCalled := false
			createManagedConfigTemp = func(dir, pattern string) (managedConfigFile, error) {
				createCalled = true
				var err error
				replacement, err = os.CreateTemp(dir, pattern)
				if err != nil {
					return nil, err
				}
				return &failingManagedConfigFile{replacement, phase, sentinel}, nil
			}
			resolutionFailure := false
			switch phase {
			case persistenceDanglingPhase:
				linkTarget = "../targets/missing"
				resolutionFailure = true
			case "link-cycle":
				linkTarget = filepath.Base(filename)
				resolutionFailure = true
			case "non-directory":
				linkTarget += "/child"
				resolutionFailure = true
			case persistenceDirectoryPhase:
				linkTarget = "../targets"
			case persistenceRenamePhase:
				renameManagedConfig = func(_, _ string) error { return sentinel }
			}
			if err := os.Remove(filename); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(linkTarget, filename); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			oldConfig, oldSettings := Config, viper.AllSettings()
			connection := testManagedConnection(t, ClusterConnection{Name: managementTestConnectionName, ManuallyCreated: true, ManagementPort: 30000}, ClusterManagement{RESTAddress: managementTestIP})
			err = persistManagedConnection(connection, true)
			if err == nil || !strings.Contains(err.Error(), "previous configuration retained") {
				t.Fatalf("failure ignored: %v", err)
			}
			if resolutionFailure && (createCalled || !strings.Contains(err.Error(), "unable to resolve configuration symlink")) {
				t.Fatalf("unresolved link reached replacement creation: %v", err)
			}
			if phase == persistenceDanglingPhase && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("lost resolution failure: %v", err)
			}
			if phase == persistenceRenamePhase && !errors.Is(err, sentinel) {
				t.Fatalf("lost failure: %v", err)
			}
			if phase == persistenceWritePhase && !strings.Contains(err.Error(), sentinel.Error()) {
				t.Fatalf("lost Viper write failure: %v", err)
			}
			after, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config, oldConfig) || !reflect.DeepEqual(viper.AllSettings(), oldSettings) || viper.ConfigFileUsed() != filename {
				t.Fatalf("failure changed prior state: %v", err)
			}
			if got, err := os.Readlink(filename); err != nil || got != linkTarget {
				t.Fatalf("failure changed selected link: got %q (%v), want %q", got, err, linkTarget)
			}
			if replacement != nil {
				if _, err := replacement.Write([]byte("handle must be closed")); !errors.Is(err, os.ErrClosed) {
					t.Fatalf("replacement handle retained: %v", err)
				}
			}
			assertNoManagedReplacement(t, filename)
			assertNoManagedReplacement(t, target)
			assertNoManagedReplacement(t, filepath.Dir(target))
		})
	}
}

func TestManagementPersistenceRequiresSelectedFile(t *testing.T) {
	managedPersistenceState(t)
	if err := writeManagedConfigAtomically(); err == nil || !strings.Contains(err.Error(), "no configuration file selected") {
		t.Fatal(err)
	}
}

const (
	ordinaryPortableProfile = "portable"
	ordinarySerializePhase  = "serialize"
	ordinaryOpenPhase       = "open"
	ordinaryYML             = "yml"
	ordinaryYAMLFile        = "cohctl.yaml"
	ordinaryJSONFile        = "cohctl.json"
	ordinaryTheme           = "green"
	ordinaryProfileValue    = "-Dcustom=true"
	ordinaryPanelLayout     = "members,services"
	ordinaryHeap            = "768m"
	ordinaryPinOne          = "one"
	ordinaryPinTwo          = "two"
	ordinaryKeep            = "keep"
	ordinaryRetain          = "retain"
	ordinaryMappingPhase    = "mapping"
	ordinaryCodecPhase      = "codec"
	ordinaryProfileName     = "caller-profile"
	ordinaryPanelName       = "caller-panel"
	ordinaryTOML            = "toml"
)

// Every saved field is deliberately nonzero, including the wide-output flag.
// It has no transient decode exclusion and therefore must round trip too.
func ordinaryPersistenceConnection(name string, management *ClusterManagement) ClusterConnection {
	return ClusterConnection{
		Name: name, DiscoveryType: nameServiceDiscoveryType, ConnectionType: httpType,
		ConnectionURL:        "https://" + name + ".example/management/coherence/cluster",
		NameServiceDiscovery: "discovery.example:7574", ClusterVersion: "14.1.2-0-5",
		ClusterName: "cluster-" + name, ClusterType: connectionTestStandalone, ManuallyCreated: true,
		BaseClasspath: "/base/coherence.jar", AdditionalClasspath: "/extra classes/custom.jar",
		Arguments:      `  -Dlegacy=it's -Djson={"mode":"safe"}  -Dpath=C:\cache\file `,
		ManagementPort: 30000, PersistenceMode: "active", LoggingDestination: "/logs with spaces",
		ManagementAvailable: true, StartupClass: "example.Start", Management: management,
	}
}

func TestOrdinaryConfigCompleteRecords(t *testing.T) {
	for _, tc := range []struct{ name, filename, format, explicitType string }{
		{configType, ordinaryYAMLFile, configType, ""},
		{ordinaryYML, "cohctl.yml", configType, ""},
		{managementTestJSON, ordinaryJSONFile, managementTestJSON, ""},
		{ordinaryTOML, "cohctl.toml", ordinaryTOML, ""},
		{"explicit-extensionless", "cohctl", managementTestJSON, managementTestJSON},
		// Ordinary Viper WriteConfig historically selects the file extension
		// before the explicit type; managed replacement selects the read type.
		{"extension-before-explicit-json", ordinaryYAMLFile, configType, managementTestJSON},
		{"extension-before-explicit-yaml", ordinaryJSONFile, managementTestJSON, configType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), tc.filename)
			viper.SetConfigFile(filename)
			viper.SetConfigType(tc.explicitType)
			want := []ClusterConnection{
				ordinaryPersistenceConnection("legacy", nil),
				ordinaryPersistenceConnection("managed", &ClusterManagement{managementTestIP, 9999, localHost, loopbackHostname}),
				ordinaryPersistenceConnection("disabled", &ClusterManagement{RESTAddress: localHost}),
				ordinaryPersistenceConnection("empty-marker", &ClusterManagement{}),
			}
			for i := 1; i < len(want); i++ {
				want[i].Arguments = `-Dcoherence.cluster=managed '-Dquoted=two words' '-Dapostrophe=it'"'"'s'`
			}
			viper.Set(clusterKey, want)
			viper.Set(currentContextKey, want[1].Name)
			viper.Set(profilesKey, []ProfileValue{{Name: ordinaryPortableProfile, Value: ordinaryProfileValue}})
			viper.Set(panelsKey, []Panel{{Name: "mine", Layout: ordinaryPanelLayout}})
			viper.Set("preferences", map[string]any{"theme": ordinaryTheme, "pins": []string{ordinaryPinOne, ordinaryPinTwo}})
			viper.Set(defaultHeapKey, ordinaryHeap)
			viper.SetDefault(requestTimeoutKey, 41)
			// The live Config is intentionally stale: it cannot own serialization.
			Config.Clusters = []ClusterConnection{{Name: "stale-must-not-return"}}
			for cycle := range 3 {
				before := viper.AllSettings()
				if err := WriteConfig(); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(viper.AllSettings(), before) || viper.ConfigFileUsed() != filename {
					t.Fatal("ordinary write changed effective in-memory settings or target")
				}
				reader, loaded := readManagedPersistence(t, filename, tc.format)
				if !reflect.DeepEqual(loaded.Clusters, want) || loaded.CurrentContext != want[1].Name || loaded.DefaultHeap != ordinaryHeap || loaded.RequestTimeout != 41 || !reflect.DeepEqual(loaded.Profiles, []ProfileValue{{Name: ordinaryPortableProfile, Value: ordinaryProfileValue}}) || !reflect.DeepEqual(loaded.Panels, []Panel{{Name: "mine", Layout: ordinaryPanelLayout}}) {
					t.Fatalf("cycle %d complete configuration changed: got %+v; want connections %+v", cycle, loaded, want)
				}
				for i, connection := range loaded.Clusters {
					assertPersistenceArguments(t, connection, want[i])
				}
				if reader.GetString("preferences.theme") != ordinaryTheme || !reflect.DeepEqual(reader.GetStringSlice("preferences.pins"), []string{ordinaryPinOne, ordinaryPinTwo}) {
					t.Fatal("unrelated effective data lost")
				}
				if cycle == 0 {
					// A write must not pin a default into the override layer.
					viper.SetDefault(requestTimeoutKey, 42)
					if viper.GetInt(requestTimeoutKey) != 42 {
						t.Fatal("writer pinned effective defaults")
					}
				}
				*viper.GetViper() = *reader
			}
		})
	}
}

func TestOrdinaryConfigEffectiveMaps(t *testing.T) {
	for _, format := range []string{configType, ordinaryYML, managementTestJSON, ordinaryTOML} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), "cohctl."+format)
			viper.SetConfigFile(filename)
			legacy := ordinaryPersistenceConnection("map-legacy", nil)
			managed := ordinaryPersistenceConnection("map-managed", &ClusterManagement{localHost, 9999, managementTestIP, loopbackHostname})
			maps, err := configurationClusters([]ClusterConnection{legacy, managed})
			if err != nil {
				t.Fatal(err)
			}
			nestedValues := []any{ordinaryKeep, true, nil, 23}
			wantNested := `{"nested":["keep",true,null,23]}`
			if format == ordinaryTOML {
				// TOML has no null value; retain every representable unknown entry.
				nestedValues = []any{ordinaryKeep, true, 23}
				wantNested = `{"nested":["keep",true,23]}`
			}
			maps[0]["future"] = map[string]any{"nested": nestedValues}
			maps[1]["management"].(map[string]any)["future-endpoint"] = ordinaryRetain
			// Both Viper's decoded []any maps and a mixed caller representation
			// must retain unknown map entries rather than decode through Config.
			viper.Set(clusterKey, []any{maps[0], maps[1], ordinaryPersistenceConnection("typed-third", nil)})
			viper.Set("future-root", map[string]any{"enabled": true, "unknown": ordinaryKeep})
			Config.Clusters = []ClusterConnection{{Name: "deleted-stale"}}
			for range 3 {
				if err := setContext(managementTestCommand(t), managed.Name); err != nil {
					t.Fatal(err)
				}
				reader, loaded := readManagedPersistence(t, filename, format)
				if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{legacy, managed, ordinaryPersistenceConnection("typed-third", nil)}) {
					t.Fatalf("effective maps lost full records: %+v", loaded.Clusters)
				}
				var raw struct{ Clusters []map[string]any }
				if err := reader.Unmarshal(&raw); err != nil {
					t.Fatal(err)
				}
				nested, err := json.Marshal(raw.Clusters[0]["future"])
				if err != nil || string(nested) != wantNested || raw.Clusters[1]["management"].(map[string]any)["future-endpoint"] != ordinaryRetain || !reader.GetBool("future-root.enabled") || reader.GetString("future-root.unknown") != ordinaryKeep {
					t.Fatalf("unknown settings changed: %s %+v (%v)", nested, raw, err)
				}
				*viper.GetViper() = *reader
			}
		})
	}
}

func TestOrdinaryAndManagedConfigWrites(t *testing.T) {
	for _, format := range []string{configType, ordinaryYML, managementTestJSON, ordinaryTOML} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), "cohctl."+format)
			viper.SetConfigFile(filename)
			legacy := ordinaryPersistenceConnection("interop-legacy", nil)
			connection := ordinaryPersistenceConnection("interop-managed", &ClusterManagement{localHost, 9999, localHost, loopbackHostname})
			// Ordinary writes preserve malformed legacy text; managed preparation accepts
			// only already-validated arguments and encodes those tokens explicitly.
			connection.Arguments = "-Dcustom=managed"

			Config.Clusters = []ClusterConnection{legacy}
			viper.Set(clusterKey, Config.Clusters)
			viper.Set("interop-preference", "preserve")
			if err := WriteConfig(); err != nil {
				t.Fatal(err)
			}
			for cycle := range 3 {
				reader, loaded := readManagedPersistence(t, filename, format)
				*viper.GetViper(), Config = *reader, loaded
				if cycle > 0 {
					connection.Management = &ClusterManagement{RESTAddress: managementTestIP}
				}
				connection = testManagedConnection(t, connection, *connection.Management)
				if err := persistManagedConnection(connection, cycle == 0); err != nil {
					t.Fatal(err)
				}
				reader, loaded = readManagedPersistence(t, filename, format)
				if !reflect.DeepEqual(loaded, Config) || !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{legacy, connection}) {
					t.Fatalf("managed-after-ordinary cycle %d: %+v", cycle, loaded)
				}
				*viper.GetViper(), Config = *reader, loaded
				if err := setDefaultHeapCmd.RunE(managementTestCommand(t), []string{ordinaryHeap}); err != nil {
					t.Fatal(err)
				}
				reader, loaded = readManagedPersistence(t, filename, format)
				if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{legacy, connection}) || loaded.CurrentContext != connection.Name || loaded.DefaultHeap != ordinaryHeap || reader.GetString("interop-preference") != "preserve" {
					t.Fatalf("ordinary-after-managed cycle %d: %+v", cycle, loaded)
				}
			}
			assertNoManagedReplacement(t, filename)
		})
	}
}

func TestOrdinaryConfigErrorsAndState(t *testing.T) {
	for _, phase := range []string{ordinaryMappingPhase, ordinarySerializePhase, ordinaryCodecPhase, ordinaryOpenPhase, persistenceWritePhase, persistenceSyncPhase} {
		t.Run(phase, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), ordinaryJSONFile)
			if err := os.WriteFile(filename, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			viper.SetConfigFile(filename)
			viper.SetDefault(clusterKey, []ClusterConnection{ordinaryPersistenceConnection("error-record", nil)})
			sentinel := errors.New("ordinary injected " + phase + " failure")
			switch phase {
			case ordinaryMappingPhase:
				viper.Set(clusterKey, []any{42})
			case ordinarySerializePhase:
				viper.Set("unserializable", make(chan int))
			case ordinaryCodecPhase:
				viper.SetConfigFile(filepath.Join(filepath.Dir(filename), "unsupported.ini"))
			case persistenceWritePhase, persistenceSyncPhase:
				viper.SetFs(ordinaryFailureFS{afero.NewOsFs(), phase, sentinel})
			case ordinaryOpenPhase:
				viper.SetConfigFile(filepath.Join(filepath.Dir(filename), "missing", ordinaryJSONFile))
			}
			before, target := viper.AllSettings(), viper.ConfigFileUsed()
			err := WriteConfig()
			if err == nil || !strings.Contains(err.Error(), "unable to write config:") || !reflect.DeepEqual(viper.AllSettings(), before) || viper.ConfigFileUsed() != target {
				t.Fatalf("failure/state lost: %v", err)
			}
			switch phase {
			case ordinaryMappingPhase:
				var mappingErr mapstructure.Error
				if !errors.As(err, &mappingErr) || !strings.Contains(err.Error(), "encode cluster settings") {
					t.Fatalf("mapping cause lost: %v", err)
				}
			case ordinarySerializePhase, ordinaryCodecPhase, persistenceWritePhase:
				var marshalErr viper.ConfigMarshalError
				if !errors.As(err, &marshalErr) {
					t.Fatalf("Viper marshal error lost: %v", err)
				}
				if phase == persistenceWritePhase && !strings.Contains(err.Error(), sentinel.Error()) {
					t.Fatalf("Viper write diagnostic lost: %v", err)
				}
			case persistenceSyncPhase:
				if !errors.Is(err, sentinel) {
					t.Fatalf("sync cause lost: %v", err)
				}
			case ordinaryOpenPhase:
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("open cause lost: %v", err)
				}
			}
			if phase == ordinaryMappingPhase {
				data, readErr := os.ReadFile(filename)
				if readErr != nil || string(data) != "original" {
					t.Fatal("mapping failure touched target")
				}
			}
			// Failure must not install a normalized override over a default.
			if phase != ordinaryMappingPhase {
				viper.SetDefault(clusterKey, []ClusterConnection{{Name: "replacement-default"}})
				if !reflect.DeepEqual(viper.Get(clusterKey), []ClusterConnection{{Name: "replacement-default"}}) {
					t.Fatal("failure pinned normalized records")
				}
			}
		})
	}
}

func TestConnectionPublicJSONSchema(t *testing.T) {
	connection := ordinaryPersistenceConnection("public", &ClusterManagement{localHost, 9999, localHost, loopbackHostname})
	data, err := json.Marshal(connection)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if output["url"] != connection.ConnectionURL || output["clusterVersionParam"] != connection.ClusterVersion || output["connectionurl"] != nil || output["clusterversion"] != nil {
		t.Fatalf("public JSON output contract changed: %s", data)
	}
}

func TestOrdinaryConfigCallerOperations(t *testing.T) {
	for _, format := range []string{configType, ordinaryYML, managementTestJSON, ordinaryTOML} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			oldType, oldUser, oldProfile, oldLayout := connectionType, Username, profileValue, panelLayout
			oldTimeout, oldDebug := fetcher.RequestTimeout, fetcher.DebugEnabled
			oldURL, oldCluster := httpManagementURL, httpManagementCluster
			t.Cleanup(func() {
				connectionType, Username, profileValue, panelLayout = oldType, oldUser, oldProfile, oldLayout
				fetcher.RequestTimeout, fetcher.DebugEnabled = oldTimeout, oldDebug
				httpManagementURL, httpManagementCluster = oldURL, oldCluster
			})
			connectionType, Username, automaticallyConfirm = httpType, "", true
			fetcher.RequestTimeout, fetcher.DebugEnabled = 2, false
			for _, name := range []string{"COHERENCE_TLS_CERTS_PATH", "COHERENCE_TLS_CLIENT_CERT", "COHERENCE_TLS_CLIENT_KEY"} {
				t.Setenv(name, "")
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := w.Write([]byte(`{"clusterName":"caller-cluster","version":"15.1.1-0-6"}`)); err != nil {
					t.Errorf("cluster response: %v", err)
				}
			}))
			server.Config.ReadHeaderTimeout, server.Config.WriteTimeout = time.Second, time.Second
			server.Start()
			closed := make(chan struct{})
			timer := time.AfterFunc(10*time.Second, func() {
				server.CloseClientConnections()
				server.Close()
				close(closed)
			})
			t.Cleanup(func() {
				if timer.Stop() {
					server.CloseClientConnections()
					server.Close()
				} else {
					<-closed
				}
			})
			filename := filepath.Join(t.TempDir(), "cohctl."+format)
			viper.SetConfigFile(filename)
			want := CoherenceCLIConfig{Clusters: []ClusterConnection{
				ordinaryPersistenceConnection("caller-legacy", nil),
				ordinaryPersistenceConnection("caller-managed", &ClusterManagement{RESTAddress: localHost}),
			}}
			want.Clusters[1].Arguments = `'-Dquoted=two words' '-Dapostrophe=it'"'"'s'`
			viper.Set(clusterKey, want.Clusters)
			viper.Set("caller-extra", ordinaryRetain)
			check := func() {
				t.Helper()
				reader, loaded := readManagedPersistence(t, filename, format)
				if !reflect.DeepEqual(loaded, want) || reader.GetString("caller-extra") != ordinaryRetain {
					t.Fatalf("caller changed complete configuration: got %+v; want %+v", loaded, want)
				}
				for i, connection := range loaded.Clusters {
					assertPersistenceArguments(t, connection, want.Clusters[i])
				}
				*viper.GetViper(), Config = *reader, loaded
			}
			writeInitial := WriteConfig
			if format == ordinaryTOML {
				// Seed the pre-existing representation through the built-in writer,
				// independently of the adapter exercised by actual add/remove below.
				writeInitial = viper.WriteConfig
			}
			if err := writeInitial(); err != nil {
				t.Fatal(err)
			}
			check()
			cmd := managementTestCommand(t)
			url := server.URL + "/management/coherence/cluster"
			if err := addCluster(cmd, "caller-added", url, nameServiceDiscoveryType, "ns.example:7574", false); err != nil {
				t.Fatal(err)
			}
			added := ClusterConnection{Name: "caller-added", ConnectionType: httpType, ConnectionURL: url, DiscoveryType: nameServiceDiscoveryType, ClusterVersion: "15.1.1-0-6", ClusterName: "caller-cluster", ClusterType: connectionTestStandalone, NameServiceDiscovery: "ns.example:7574"}
			want.Clusters = append(want.Clusters, added)
			check()
			if err := setContext(cmd, added.Name); err != nil {
				t.Fatal(err)
			}
			want.CurrentContext = added.Name
			check()
			profileValue, panelLayout = "-Dcaller=true", ordinaryPanelLayout
			for _, op := range []struct {
				command *cobra.Command
				args    []string
				update  func()
			}{
				{setProfileCmd, []string{ordinaryProfileName}, func() { want.Profiles = []ProfileValue{{Name: ordinaryProfileName, Value: profileValue}} }},
				{addPanelCmd, []string{ordinaryPanelName}, func() { want.Panels = []Panel{{Name: ordinaryPanelName, Layout: panelLayout}} }},
				{setTimeoutCmd, []string{"43"}, func() { want.RequestTimeout = 43 }},
				{setDebugCmd, []string{on}, func() { want.Debug = true }},
				{setBytesFormatCmd, []string{bytesFormatG}, func() { want.DefaultBytesFormat = bytesFormatG }},
				{removeProfileCmd, []string{ordinaryProfileName}, func() { want.Profiles = []ProfileValue{} }},
				{removePanelCmd, []string{ordinaryPanelName}, func() { want.Panels = []Panel{} }},
			} {
				if err := op.command.RunE(cmd, op.args); err != nil {
					t.Fatal(err)
				}
				op.update()
				check()
			}
			if err := removeClusterCmd.RunE(cmd, []string{added.Name}); err != nil {
				t.Fatal(err)
			}
			want.Clusters, want.CurrentContext = want.Clusters[:2], ""
			check()
		})
	}
}

// Exercise the ordinary Viper writer on a real private file. Its historical
// truncation/partial-write behavior is deliberately not an atomic transaction.
type ordinaryFailureFS struct {
	afero.Fs
	phase string
	err   error
}

func (f ordinaryFailureFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := f.Fs.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return ordinaryFailureFile{file, f.phase, f.err}, nil
}

type ordinaryFailureFile struct {
	afero.File
	phase string
	err   error
}

func (f ordinaryFailureFile) Write(data []byte) (int, error) {
	if f.phase == persistenceWritePhase {
		n, err := f.File.Write(data[:len(data)/2])
		if err != nil {
			return n, err
		}
		return n, f.err
	}
	return f.File.Write(data)
}

func (f ordinaryFailureFile) Sync() error {
	if f.phase == persistenceSyncPhase {
		return f.err
	}
	return f.File.Sync()
}

// Compare the client and server argument boundaries against the original record,
// including empty legacy space separators, literal quotes and backslashes.
func assertPersistenceArguments(t *testing.T, got, want ClusterConnection) {
	t.Helper()
	client, clientErr := getCommonArguments(got)
	wantClient, wantClientErr := getCommonArguments(want)
	server, serverErr := serverBaseOptions(got)
	wantServer, wantServerErr := serverBaseOptions(want)
	if clientErr != nil || wantClientErr != nil || serverErr != nil || wantServerErr != nil || !reflect.DeepEqual(client, wantClient) || !reflect.DeepEqual(server, wantServer) {
		t.Fatalf("%s saved argv changed: client %q/%q (%v/%v); server %q/%q (%v/%v)", got.Name, client, wantClient, clientErr, wantClientErr, server, wantServer, serverErr, wantServerErr)
	}
}

func TestOrdinaryTOMLPreexistingLegacy(t *testing.T) {
	managedPersistenceState(t)
	filename := filepath.Join(t.TempDir(), "legacy.toml")
	// This literal file predates configuration-name mapping. Its legacy record
	// has no Management, while the second record intentionally has an empty marker.
	const original = `color = 'on'
currentcontext = 'legacy'
[future-root]
enabled = true
[[clusters]]
name = 'legacy'
connectionurl = 'http://127.0.0.1:1/management/coherence/cluster'
clusterversion = '22.06.15'
manuallycreated = true
arguments = """  -Dcoherence.wka=127.0.0.1 -Dpath=/o'brien.xml -Djson={\"mode\":\"safe\"}  -Dpath=C:\\cache\\file -Djava.rmi.server.hostname=127.0.0.1 """
[clusters.future]
unknown = 'keep'
[[clusters]]
name = 'quoted'
arguments = "'-Dquoted=two words'"
[clusters.management]
`
	if err := os.WriteFile(filename, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	reader, loaded := readManagedPersistence(t, filename, "")
	legacy, quoted := loaded.Clusters[0], loaded.Clusters[1]
	if legacy.Management != nil || quoted.Management == nil || quoted.Arguments != "'-Dquoted=two words'" {
		t.Fatalf("pre-existing marker distinctions lost: %+v", loaded.Clusters)
	}
	*viper.GetViper(), Config = *reader, loaded
	for cycle := range 3 {
		// A settings-only write must retain decoded map extensions and absence.
		if err := setTimeoutCmd.RunE(managementTestCommand(t), []string{"43"}); err != nil {
			t.Fatal(err)
		}
		reader, loaded = readManagedPersistence(t, filename, "")
		var raw struct{ Clusters []map[string]any }
		if err := reader.Unmarshal(&raw); err != nil {
			t.Fatal(err)
		}
		if cycle == 0 && raw.Clusters[0]["future"].(map[string]any)["unknown"] != ordinaryKeep {
			t.Fatal("writer discarded decoded-map extension")
		}
		// add/remove own declared typed records; unlike the writer they cannot
		// preserve undeclared per-record fields. Do not assert a new schema here.
		viper.Set(clusterKey, loaded.Clusters)
		if err := WriteConfig(); err != nil {
			t.Fatal(err)
		}
		reader, loaded = readManagedPersistence(t, filename, "")
		if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{legacy, quoted}) || !reader.GetBool("future-root.enabled") {
			t.Fatalf("cycle %d changed pre-existing records: %+v", cycle, loaded.Clusters)
		}
		for i, want := range []ClusterConnection{legacy, quoted} {
			assertPersistenceArguments(t, loaded.Clusters[i], want)
		}
		*viper.GetViper(), Config = *reader, loaded
	}
}
