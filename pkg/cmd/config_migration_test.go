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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	migrationConnectionName   = "fixture"
	migrationAfterSync        = "directory-after"
	migrationReplace          = "replace"
	migrationConcurrentChange = "concurrent-change"
	migrationShortWrite       = "short"
	migrationStartupVersion   = "migration-startup-fixture"
	migrationVersionKey       = "version"
	migrationHTTPSURL         = "https://example.test/"
	migrationBackupKind       = "backup"
)

const migrationFixture = `# The backup must preserve this comment and the original bytes.
version: fixture-version
debug: false
color: off
ignoreInvalidCerts: true
currentContext: legacy
requestTimeout: 47
extension:
  MixedCase: preserved
  literal.dot: [1, two, null]
  empty: null
clusters:
  - name: legacy
    connectionType: http
    connectionURL: https://example.test:30000/management/coherence/cluster
    futureData:
      CaseSensitive: [one, two]
      literal.dot: null
  - name: verified
    connectionurl: https://example.test:30001/
    ignoreInvalidCerts: false
  - name: exception
    connectionurl: https://example.test:30002/
    ignoreInvalidCerts: true
  - name: plain
    connectionurl: http://example.test:30000/
`

func TestLegacyTLSMigrationPersistedReloadAndPreservation(t *testing.T) {
	file := writeMigrationFixture(t, migrationFixture)
	before := decodeMigrationFixture(t, "yaml", []byte(migrationFixture))
	live, _ := readConnectionConfig(t, file)
	var stderr bytes.Buffer
	config, err := loadExistingConfig(live, &stderr, defaultConfigMigrationOps())
	if err != nil {
		t.Fatal(err)
	}
	_, reloaded := readConnectionConfig(t, file)
	if !reflect.DeepEqual(config, reloaded) || config.IgnoreInvalidCerts || live.GetBool(ignoreCertsContextKey) {
		t.Fatalf("disk/live state differs after migration: %#v, %#v", config, reloaded)
	}
	if config.Clusters[0].IgnoreInvalidCerts == nil || !*config.Clusters[0].IgnoreInvalidCerts ||
		config.Clusters[1].IgnoreInvalidCerts == nil || *config.Clusters[1].IgnoreInvalidCerts ||
		config.Clusters[2].IgnoreInvalidCerts == nil || !*config.Clusters[2].IgnoreInvalidCerts ||
		config.Clusters[3].IgnoreInvalidCerts != nil {
		t.Fatalf("connection choices changed incorrectly: %#v", config.Clusters)
	}
	after := decodeMigrationFixture(t, "yaml", readMigrationBytes(t, file))
	before[ignoreCertsContextKey] = false
	before[clusterKey].([]any)[0].(map[string]any)["ignoreinvalidcerts"] = true
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("unrelated data changed:\nbefore: %#v\nafter: %#v", before, after)
	}
	if !strings.Contains(stderr.String(), `["legacy"]`) || !strings.Contains(stderr.String(), "unnecessary connection exceptions") ||
		strings.Contains(stderr.String(), "example.test") {
		t.Fatalf("unexpected migration report: %s", &stderr)
	}
	backups := migrationBackups(t, file)
	if len(backups) != 1 || !strings.Contains(backups[0], "_UTC-") ||
		!bytes.Equal(readMigrationBytes(t, backups[0]), []byte(migrationFixture)) {
		t.Fatalf("missing exact timestamped sibling backup: %v", backups)
	}
	assertMigrationPermissions(t, backups[0], 0600)
	assertMigrationPermissions(t, file, 0600)
	staging, err := filepath.Glob(filepath.Join(filepath.Dir(file), "."+filepath.Base(file)+".tls-stage-*"))
	if err != nil || len(staging) != 0 {
		t.Fatalf("successful replacement left staging files: %v, %v", staging, err)
	}

	// A fresh startup must be byte-for-byte inert, with no message/extra backup.
	saved := readMigrationBytes(t, file)
	live, _ = readConnectionConfig(t, file)
	stderr.Reset()
	if _, err = loadExistingConfig(live, &stderr, defaultConfigMigrationOps()); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 || !bytes.Equal(saved, readMigrationBytes(t, file)) || len(migrationBackups(t, file)) != 1 {
		t.Fatal("repeated startup migrated again")
	}

	// Match the existing add/save path: a later connection must not inherit true.
	config.Clusters = append(config.Clusters, ClusterConnection{Name: "later", ConnectionType: httpType, ConnectionURL: "https://later.test/"})
	live.Set(clusterKey, config.Clusters)
	if err = live.WriteConfig(); err != nil {
		t.Fatal(err)
	}
	live, _ = readConnectionConfig(t, file)
	config, err = loadExistingConfig(live, &stderr, defaultConfigMigrationOps())
	if err != nil || config.Clusters[len(config.Clusters)-1].IgnoreInvalidCerts != nil || stderr.Len() != 0 || len(migrationBackups(t, file)) != 1 {
		t.Fatalf("new connection inherited the legacy setting: %#v, %v, %s", config, err, &stderr)
	}
}

func TestLegacyTLSMigrationEligibility(t *testing.T) {
	for _, test := range []struct {
		name    string
		address string
		choice  string
		want    *bool
	}{
		{name: "https", address: migrationHTTPSURL, want: boolMigrationPointer(true)},
		{name: "uppercase", address: "HTTPS://EXAMPLE.test/", want: boolMigrationPointer(true)},
		{name: "ipv6", address: "https://[::1]:443/", want: boolMigrationPointer(true)},
		{name: "explicit false", address: migrationHTTPSURL, choice: stringFalse, want: boolMigrationPointer(false)},
		{name: "explicit true", address: migrationHTTPSURL, choice: stringTrue, want: boolMigrationPointer(true)},
		{name: "null optional", address: migrationHTTPSURL, choice: "null", want: boolMigrationPointer(true)},
		{name: httpType, address: "http://example.test/"},
		{name: "http explicit", address: "http://example.test/", choice: stringTrue, want: boolMigrationPointer(true)},
		{name: "other scheme", address: "ftp://example.test/"},
		{name: "relative", address: "example.test/path"},
		{name: "scheme relative", address: "//example.test/path"},
		{name: "opaque", address: "https:example.test"},
		{name: "host absent", address: "https:///path"},
		{name: "hostname absent", address: "https://:443/"},
		{name: "invalid port", address: "https://example.test:no/"},
		{name: "port out of range", address: "https://example.test:65536/"},
		{name: "empty port", address: "https://example.test:/"},
		{name: "zero port", address: "https://example.test:0/"},
		{name: "zero padded zero port", address: "https://example.test:000/"},
		{name: "invalid escape", address: "https://example.test/%zz"},
		{name: "invalid IPv6", address: "https://[::1/"},
		{name: "invalid bracketed hostname", address: "https://[not-an-ip]/"},
		{name: "unmatched closing bracket", address: "https://example.test]/"},
		{name: "unbracketed IPv6", address: "https://2001:db8::1/"},
		{name: "whitespace", address: "https://bad host/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			contents := "ignoreInvalidCerts: true\nclusters:\n  - name: fixture\n    connectiontype: http\n    connectionurl: " + test.address + "\n"
			if test.choice != "" {
				contents += "    ignoreInvalidCerts: " + test.choice + "\n"
			}
			file := writeMigrationFixture(t, contents)
			result, err := migrateLegacyTLSConfig(file, defaultConfigMigrationOps())
			if err != nil || result == nil {
				t.Fatalf("migration failed: %v", err)
			}
			_, config := readConnectionConfig(t, file)
			if config.IgnoreInvalidCerts || !reflect.DeepEqual(config.Clusters[0].IgnoreInvalidCerts, test.want) {
				t.Fatalf("unexpected eligibility result: %#v", config)
			}
			if test.want == nil && len(result.names) != 0 {
				t.Fatalf("ineligible connection reported as migrated: %v", result.names)
			}
		})
	}
}

func TestConfigJSONWritesPreserveUnknownNumbers(t *testing.T) {
	const migrationPhase, startupPhase = "migration", "startup"
	for _, phase := range []string{migrationPhase, startupPhase, "setter"} {
		t.Run(phase, func(t *testing.T) {
			file := certificateCommandConfig(t, "")
			file = filepath.Join(filepath.Dir(file), "selected.json")
			unknown := map[string]any{
				"id": json.Number("9007199254740993"), "ratio": json.Number("0.1234567890123456789"),
				"negativeZero": json.Number("-0"), "exponent": json.Number("1.234567890123456789e+50"),
			}
			original := map[string]any{
				migrationVersionKey: Version, "color": off, "currentContext": migrationConnectionName,
				"ignoreInvalidCerts": phase == migrationPhase, "Extension": unknown,
				"clusters": []any{map[string]any{"name": migrationConnectionName, "connectiontype": httpType,
					"connectionurl": migrationHTTPSURL, "managementport": 30000, "FutureData": unknown}},
			}
			if phase == startupPhase {
				original[migrationVersionKey], original["color"] = "old-fixture-version", ""
			}
			contents, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(file, contents, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{certificateGetAction, certificateCommand}
			if phase == "setter" {
				args = []string{certificateSetAction, certificateCommand, stringTrue}
			}
			if _, err = runCertificateCommand(t, file, args...); err != nil {
				t.Fatal(err)
			}
			// Read expected number tokens independently of Viper's float64 codec;
			// decoding both sides through that codec would hide data loss.
			var saved map[string]any
			decoder := json.NewDecoder(bytes.NewReader(readMigrationBytes(t, file)))
			decoder.UseNumber()
			if err = decoder.Decode(&saved); err != nil {
				t.Fatal(err)
			}
			entry := saved[clusterKey].([]any)[0].(map[string]any)
			if !reflect.DeepEqual(saved["Extension"], unknown) || !reflect.DeepEqual(entry["FutureData"], unknown) {
				t.Fatalf("%s changed unknown numeric values: %s", phase, readMigrationBytes(t, file))
			}
			_, loaded := readConnectionConfig(t, file)
			choice := loaded.Clusters[0].IgnoreInvalidCerts
			if loaded.IgnoreInvalidCerts || loaded.Clusters[0].ManagementPort != 30000 ||
				(phase == startupPhase && (choice != nil || loaded.Color != on || loaded.Version != Version)) ||
				(phase != startupPhase && (choice == nil || !*choice)) {
				t.Fatalf("%s did not persist its intended update: %#v", phase, loaded)
			}
			if phase == migrationPhase {
				backups := migrationBackups(t, file)
				if len(backups) != 1 || !bytes.Equal(readMigrationBytes(t, backups[0]), contents) {
					t.Fatal("migration backup changed original numeric values")
				}
			}
		})
	}
}

func TestConfigJSONUpdateRejectsTrailingData(t *testing.T) {
	for _, suffix := range []string{"{}", "true", "malformed-json"} {
		t.Run(suffix, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "selected.json")
			contents := []byte(`{"color":"off"}` + suffix)
			if err := os.WriteFile(file, contents, 0600); err != nil {
				t.Fatal(err)
			}
			live := viper.New()
			live.SetConfigFile(file)
			if err := writeStartupConfigValue(live, colorContextKey, on); err == nil ||
				!bytes.Equal(readMigrationBytes(t, file), contents) {
				t.Fatalf("invalid JSON was accepted or modified: %v", err)
			}
		})
	}
}

func TestLegacyTLSMigrationFileAndEnvironment(t *testing.T) {
	for _, test := range []struct {
		name    string
		stored  string
		env     string
		migrate bool
	}{
		{name: "file false env true", stored: stringFalse, env: stringTrue},
		{name: "file true env false", stored: stringTrue, env: stringFalse, migrate: true},
		{name: "file absent env true", env: stringTrue},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("IGNOREINVALIDCERTS", test.env)
			t.Setenv("DEBUG", stringTrue)
			contents := "debug: false\nclusters:\n  - name: fixture\n    connectionurl: https://example.test/\n"
			if test.stored != "" {
				contents += "ignoreInvalidCerts: " + test.stored + "\n"
			}
			file := writeMigrationFixture(t, contents)
			live, _ := readConnectionConfig(t, file)
			live.AutomaticEnv()
			// Also prove migration does not consult stale process-local overrides.
			live.Set(ignoreCertsContextKey, test.env == stringTrue)
			live.Set(clusterKey, []ClusterConnection{{Name: "stale"}})
			var stderr bytes.Buffer
			config, err := loadExistingConfig(live, &stderr, defaultConfigMigrationOps())
			if err != nil || config.IgnoreInvalidCerts || live.GetBool(ignoreCertsContextKey) || !config.Debug {
				t.Fatalf("environment isolation failed: %#v, %v", config, err)
			}
			if test.migrate {
				if config.Clusters[0].Name != migrationConnectionName || config.Clusters[0].IgnoreInvalidCerts == nil || !*config.Clusters[0].IgnoreInvalidCerts || stderr.Len() == 0 {
					t.Fatalf("stored true was not migrated: %#v, %s", config, &stderr)
				}
			} else if stderr.Len() != 0 || !bytes.Equal(readMigrationBytes(t, file), []byte(contents)) || len(migrationBackups(t, file)) != 0 {
				t.Fatal("environment caused a persisted migration")
			}
		})
	}
}

func TestLegacyTLSMigrationNoEligibleConnections(t *testing.T) {
	for _, contents := range []string{
		"ignoreInvalidCerts: true\n",
		"ignoreInvalidCerts: true\nclusters: []\n",
		"ignoreInvalidCerts: true\nclusters:\n  - name: plain\n    connectionurl: http://example.test/\n",
	} {
		file := writeMigrationFixture(t, contents)
		live, _ := readConnectionConfig(t, file)
		var stderr bytes.Buffer
		config, err := loadExistingConfig(live, &stderr, defaultConfigMigrationOps())
		if err != nil || config.IgnoreInvalidCerts || !strings.Contains(stderr.String(), "no eligible HTTPS connections") || len(migrationBackups(t, file)) != 1 {
			t.Fatalf("clearing with no eligible connections failed: %#v, %v, %s", config, err, &stderr)
		}
	}
}

func TestLegacyTLSMigrationCustomFormats(t *testing.T) {
	for _, format := range []string{ordinaryYML, "json", ordinaryTOML} {
		t.Run(format, func(t *testing.T) {
			original := decodeMigrationFixture(t, "yaml", []byte(migrationFixture))
			// TOML has no null representation; test its supported values separately.
			if format == ordinaryTOML {
				original = map[string]any{ignoreCertsContextKey: true, "Extension": map[string]any{"Owner": migrationConnectionName},
					clusterKey: []any{map[string]any{"name": migrationConnectionName, "connectionURL": migrationHTTPSURL, "Future": "kept"}}}
			}
			encoder, err := viper.NewCodecRegistry().Encoder(format)
			if err != nil {
				t.Fatal(err)
			}
			contents, err := encoder.Encode(original)
			if err != nil {
				t.Fatal(err)
			}
			want := decodeMigrationFixture(t, format, contents)
			want[ignoreCertsContextKey] = false
			want[clusterKey].([]any)[0].(map[string]any)["ignoreinvalidcerts"] = true
			file := filepath.Join(t.TempDir(), "selected-settings."+format)
			if err = os.WriteFile(file, contents, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := migrateLegacyTLSConfig(file, defaultConfigMigrationOps())
			if err != nil || result == nil || !bytes.Equal(readMigrationBytes(t, result.backup), contents) {
				t.Fatalf("custom selected format failed: %v", err)
			}
			_, config := readConnectionConfig(t, file)
			if config.IgnoreInvalidCerts || config.Clusters[0].IgnoreInvalidCerts == nil || !*config.Clusters[0].IgnoreInvalidCerts {
				t.Fatalf("custom format failed to reload: %#v", config)
			}
			if got := decodeMigrationFixture(t, format, readMigrationBytes(t, file)); !reflect.DeepEqual(got, want) {
				t.Fatalf("custom format lost unrelated data: got %#v, want %#v", got, want)
			}
		})
	}
}

func TestLegacyTLSMigrationSymlink(t *testing.T) {
	file := writeMigrationFixture(t, migrationFixture)
	link := filepath.Join(t.TempDir(), "selected.yaml")
	if err := os.Symlink(file, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result, err := migrateLegacyTLSConfig(link, defaultConfigMigrationOps())
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 || filepath.Dir(result.backup) != filepath.Dir(file) {
		t.Fatalf("selected symlink was replaced or backed up in the wrong directory: %v", err)
	}
	_, config := readConnectionConfig(t, link)
	if config.IgnoreInvalidCerts {
		t.Fatal("symlink target was not migrated")
	}
}

func TestLegacyTLSMigrationInvalidConfiguration(t *testing.T) {
	for _, contents := range []string{
		"ignoreInvalidCerts: [true\n",
		"ignoreInvalidCerts: invalid\n",
		"ignoreInvalidCerts: true\nclusters: invalid\n",
		"ignoreInvalidCerts: true\nclusters: [invalid]\n",
		"ignoreInvalidCerts: true\nclusters:\n  - ignoreInvalidCerts: invalid\n",
		"ignoreInvalidCerts: true\nIGNOREINVALIDCERTS: false\n",
		"ignoreInvalidCerts: true\nclusters:\n  - connectionurl: https://example.test/\n    connectionURL: https://other.test/\n",
	} {
		file := writeMigrationFixture(t, contents)
		result, err := migrateLegacyTLSConfig(file, defaultConfigMigrationOps())
		if err == nil || result != nil || !bytes.Equal(readMigrationBytes(t, file), []byte(contents)) || len(migrationBackups(t, file)) != 0 {
			t.Fatalf("invalid configuration was modified: %q, %v", contents, err)
		}
	}
}

func TestLegacyTLSMigrationFailures(t *testing.T) {
	injected := errors.New("injected persistence failure")
	for _, phase := range []string{"backup-create", "backup-restrict", "backup-write", "backup-short", "backup-sync", "backup-close",
		"stage-create", "stage-restrict", "stage-write", "stage-short", "stage-sync", "stage-close", "directory-before", migrationReplace, migrationAfterSync, migrationConcurrentChange} {
		t.Run(phase, func(t *testing.T) {
			file := writeMigrationFixture(t, migrationFixture)
			live, before := readConnectionConfig(t, file)
			ops := defaultConfigMigrationOps()
			create := ops.createTemp
			ops.createTemp = func(dir, pattern string) (migrationFile, error) {
				kind := "stage"
				if strings.Contains(pattern, migrationBackupKind) {
					kind = migrationBackupKind
				}
				if phase == kind+"-create" {
					return nil, injected
				}
				f, err := create(dir, pattern)
				if err != nil {
					return nil, err
				}
				return &faultMigrationFile{migrationFile: f, phase: strings.TrimPrefix(phase, kind+"-"), injected: injected}, nil
			}
			restrict := ops.restrict
			ops.restrict = func(f migrationFile, mode os.FileMode) error {
				kind := "stage"
				if strings.Contains(f.Name(), migrationBackupKind) {
					kind = migrationBackupKind
				}
				if phase == kind+"-restrict" {
					return injected
				}
				return restrict(f, mode)
			}
			syncDir, calls := ops.syncDir, 0
			ops.syncDir = func(dir string) error {
				calls++
				if (phase == "directory-before" && calls == 1) || (phase == migrationAfterSync && calls == 2) {
					return injected
				}
				if phase == migrationConcurrentChange && calls == 1 {
					return os.WriteFile(file, []byte("ignoreInvalidCerts: false\n# another writer\n"), 0600)
				}
				return syncDir(dir)
			}
			if phase == migrationReplace {
				ops.replace = func(_, _ string) error { return injected }
			}
			var stderr bytes.Buffer
			_, err := loadExistingConfig(live, &stderr, ops)
			if err == nil || stderr.Len() != 0 {
				t.Fatalf("failure reported success: %v, %s", err, &stderr)
			}
			if phase != migrationConcurrentChange && !errors.Is(err, injected) && !errors.Is(err, io.ErrShortWrite) {
				t.Fatalf("failure lost its cause: %v", err)
			}
			var after CoherenceCLIConfig
			if err = live.Unmarshal(&after); err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("failed migration changed live state: %#v, %v", after, err)
			}
			contents := readMigrationBytes(t, file)
			if phase == migrationAfterSync {
				_, installed := readConnectionConfig(t, file)
				if installed.IgnoreInvalidCerts || installed.Clusters[0].IgnoreInvalidCerts == nil {
					t.Fatal("expected replacement before final directory sync failure")
				}
			} else if phase == migrationConcurrentChange {
				if !strings.Contains(string(contents), "another writer") {
					t.Fatal("concurrent edit was overwritten")
				}
			} else if !bytes.Equal(contents, []byte(migrationFixture)) {
				t.Fatal("failed migration damaged the original")
			}
			if !strings.HasPrefix(phase, "backup-") {
				backups := migrationBackups(t, file)
				if len(backups) != 1 || !bytes.Equal(readMigrationBytes(t, backups[0]), []byte(migrationFixture)) {
					t.Fatalf("recovery backup was lost: %v", backups)
				}
				assertMigrationPermissions(t, backups[0], 0600)
			}
		})
	}
}

type faultMigrationFile struct {
	migrationFile
	phase    string
	injected error
}

func (f *faultMigrationFile) Write(data []byte) (int, error) {
	if f.phase == "write" || f.phase == migrationShortWrite {
		n, err := f.migrationFile.Write(data[:len(data)/2])
		if err != nil || f.phase == migrationShortWrite {
			return n, err
		}
		return n, f.injected
	}
	return f.migrationFile.Write(data)
}

func (f *faultMigrationFile) Sync() error {
	if f.phase == "sync" {
		return f.injected
	}
	return f.migrationFile.Sync()
}

func (f *faultMigrationFile) Close() error {
	if err := f.migrationFile.Close(); err != nil {
		return err
	}
	if f.phase == "close" {
		return f.injected
	}
	return nil
}

// The helper runs the real Cobra startup in a separate process: initConfig's
// os.Exit failures must prevent both command execution and its network request.
func TestLegacyTLSStartupHelper(t *testing.T) {
	if os.Getenv("COHCTL_TEST_TLS_MIGRATION_STARTUP_HELPER") == "" {
		return
	}
	Version = migrationStartupVersion
	cli := Initialize(nil)
	cli.AddCommand(&cobra.Command{Use: "migration-probe", RunE: func(cmd *cobra.Command, _ []string) error {
		// #nosec G704 -- The parent test supplies only its task-owned loopback HTTP fixture URL.
		response, err := http.Get(os.Getenv("COHCTL_TEST_TLS_MIGRATION_PROBE_URL"))
		if err != nil {
			return err
		}
		_ = response.Body.Close()
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"global": Config.IgnoreInvalidCerts, "viperGlobal": viper.GetBool(ignoreCertsContextKey),
			"debug":             Config.Debug,
			migrationVersionKey: Config.Version, "color": Config.Color, "clusters": Config.Clusters,
		})
	}})
	cli.SetArgs([]string{configOption, os.Getenv("COHCTL_TEST_TLS_MIGRATION_CONFIG"), "migration-probe"})
	if err := cli.Execute(); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestLegacyTLSStartupOrdering(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	contents := strings.ReplaceAll(migrationFixture, "color: off\n", "")
	file := writeMigrationFixture(t, contents)
	want := decodeMigrationFixture(t, "yaml", []byte(contents))
	want[ignoreCertsContextKey] = false
	want[migrationVersionKey] = migrationStartupVersion
	want[colorContextKey] = on
	want[clusterKey].([]any)[0].(map[string]any)["ignoreinvalidcerts"] = true
	t.Setenv("IGNOREINVALIDCERTS", stringTrue)
	t.Setenv("DEBUG", stringTrue)
	stdout, stderr, err := runMigrationStartup(t, file, server.URL)
	if err != nil || requests.Load() != 1 {
		t.Fatalf("startup failed: %v, stdout %s, stderr %s", err, stdout, stderr)
	}
	var loaded map[string]any
	if err = json.Unmarshal(stdout, &loaded); err != nil {
		t.Fatalf("migration contaminated command stdout: %s, %v", stdout, err)
	}
	if loaded["global"] != false || loaded["viperGlobal"] != false ||
		loaded[migrationVersionKey] != migrationStartupVersion || loaded["color"] != on || loaded["debug"] != true {
		t.Fatalf("startup did not synchronize policy and unrelated environment handling: %#v", loaded)
	}
	backups := migrationBackups(t, file)
	if len(backups) != 1 || !bytes.Equal(readMigrationBytes(t, backups[0]), []byte(contents)) {
		t.Fatal("migration ran after a color/version startup write")
	}
	_, saved := readConnectionConfig(t, file)
	if saved.Color != on || saved.Version != migrationStartupVersion || saved.Clusters[0].IgnoreInvalidCerts == nil ||
		!strings.Contains(string(stderr), "Migrated legacy") {
		t.Fatalf("startup did not persist migration before other updates: %#v, %s", saved, stderr)
	}
	if got := decodeMigrationFixture(t, "yaml", readMigrationBytes(t, file)); !reflect.DeepEqual(got, want) {
		t.Fatalf("startup writes changed unrelated configuration: got %#v, want %#v", got, want)
	}
	stdout, stderr, err = runMigrationStartup(t, file, server.URL)
	if err != nil || requests.Load() != 2 || len(stderr) != 0 || len(migrationBackups(t, file)) != 1 {
		t.Fatalf("repeated startup migrated again: %v, %s, %s", err, stdout, stderr)
	}
	// Reintroducing a persisted legacy true (the old setter still exists) must
	// migrate even though the CLI version matches and color needs no update.
	contents = strings.Replace(string(readMigrationBytes(t, file)), "\nignoreInvalidCerts: false\n", "\nignoreInvalidCerts: true\n", 1)
	if err = os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err = runMigrationStartup(t, file, server.URL)
	if err != nil || requests.Load() != 3 || !strings.Contains(string(stderr), "no eligible HTTPS connections") || len(migrationBackups(t, file)) != 2 {
		t.Fatalf("same-version migration failed or reused a backup: %v, %s, %s", err, stdout, stderr)
	}
}

func TestLegacyTLSStartupFailureStopsWork(t *testing.T) {
	if isWindows() {
		t.Skip("directory Chmod cannot model Windows ACL denial; persistence failures are injected separately")
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	contents := strings.ReplaceAll(migrationFixture, "color: off\n", "")
	file := writeMigrationFixture(t, contents)
	dir := filepath.Dir(file)
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
	probe, err := os.CreateTemp(dir, "permission-probe-")
	if err == nil {
		_ = probe.Close()
		_ = os.Remove(probe.Name())
		t.Skip("current user can write a read-only directory")
	}
	stdout, stderr, err := runMigrationStartup(t, file, server.URL)
	if err == nil || requests.Load() != 0 || len(stdout) != 0 || !strings.Contains(string(stderr), "original not replaced") ||
		strings.Contains(string(stderr), "Migrated legacy") {
		t.Fatalf("failed migration continued startup: %v, requests %d, stdout %s, stderr %s", err, requests.Load(), stdout, stderr)
	}
	if !bytes.Equal(readMigrationBytes(t, file), []byte(contents)) {
		t.Fatal("color/version write preceded a failed migration")
	}
}

func TestStartupConfigValueFailureStopsWork(t *testing.T) {
	if isWindows() {
		t.Skip("directory Chmod cannot model Windows ACL denial; persistence failures are injected separately")
	}
	for _, test := range []struct{ setting, color, version string }{
		{colorContextKey, "", migrationStartupVersion},
		{migrationVersionKey, off, "fixture-version"},
	} {
		t.Run(test.setting, func(t *testing.T) {
			// Persisted false makes migration a no-op. Isolate each later save:
			// color has the current version; version has normalized color.
			t.Setenv("COLOR", "")
			t.Setenv("VERSION", "")
			contents := strings.Replace(migrationFixture, "ignoreInvalidCerts: true", "ignoreInvalidCerts: false", 1)
			contents = strings.Replace(contents, "color: off", fmt.Sprintf("color: %q", test.color), 1)
			contents = strings.Replace(contents, "version: fixture-version", "version: "+test.version, 1)
			file := writeMigrationFixture(t, contents)
			dir := filepath.Dir(file)
			if err := os.Chmod(dir, 0500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
			probe, err := os.CreateTemp(dir, "permission-probe-")
			if err == nil {
				_ = probe.Close()
				_ = os.Remove(probe.Name())
				t.Skip("current user can write a read-only directory")
			}
			if !os.IsPermission(err) {
				t.Fatalf("unexpected directory write probe failure: %v", err)
			}
			// The denied directory must still allow the selected config to be
			// read and decoded, rather than failing before the intended save.
			_, loaded := readConnectionConfig(t, file)
			if loaded.IgnoreInvalidCerts || loaded.Color != test.color || loaded.Version != test.version {
				t.Fatalf("startup fixture does not isolate %s: %#v", test.setting, loaded)
			}
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			stdout, stderr, err := runMigrationStartup(t, file, server.URL)
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("expected startup exit 1: %v, stdout %s, stderr %s", err, stdout, stderr)
			}
			// Existing startup errors may be printed to stdout. Require the
			// intended setting/staging failure, not an empty output stream.
			output := string(stdout) + string(stderr)
			want := fmt.Sprintf("write setting %q in %q; staging file %q may be retained:", test.setting, file, "")
			if !strings.Contains(output, want) || !strings.Contains(output, ".config-stage-") || !strings.Contains(output, "permission denied") {
				t.Fatalf("startup did not reach the intended %s save: stdout %s, stderr %s", test.setting, stdout, stderr)
			}
			if requests.Load() != 0 || strings.Contains(output, `"viperGlobal"`) {
				t.Fatalf("failed startup save ran the probe or emitted its success payload: requests %d, stdout %s, stderr %s", requests.Load(), stdout, stderr)
			}
			if !bytes.Equal(readMigrationBytes(t, file), []byte(contents)) {
				t.Fatal("failed startup save changed the original file")
			}
			if strings.Contains(output, "Migrated legacy") || len(migrationBackups(t, file)) != 0 {
				t.Fatalf("persisted legacy false triggered migration: stdout %s, stderr %s", stdout, stderr)
			}
		})
	}
}

func runMigrationStartup(t *testing.T, file, probeURL string) ([]byte, []byte, error) {
	t.Helper()
	home := t.TempDir()
	// #nosec G204 G702 -- Execute this test binary only; fixed test selector, isolated child configuration and local fixture URL.
	command := exec.Command(os.Args[0], "-test.run=^TestLegacyTLSStartupHelper$")
	command.Env = append(os.Environ(), "COHCTL_TEST_TLS_MIGRATION_STARTUP_HELPER=1", "COHCTL_TEST_TLS_MIGRATION_CONFIG="+file,
		"COHCTL_TEST_TLS_MIGRATION_PROBE_URL="+probeURL, "COHCTL_HOME="+home, "HOME="+home, "USERPROFILE="+home)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func writeMigrationFixture(t *testing.T, contents string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "selected-settings.yaml")
	// Deliberately start with broader permissions: recovery/replacement must tighten them.
	// #nosec G306 -- Deliberately broad task-owned fixture mode verifies that migration restricts backup and replacement access.
	if err := os.WriteFile(file, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return file
}

func readMigrationBytes(t *testing.T, file string) []byte {
	t.Helper()
	// #nosec G703 -- Paths come only from task-owned test fixtures or migration recovery filenames.
	contents, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func decodeMigrationFixture(t *testing.T, format string, contents []byte) map[string]any {
	t.Helper()
	decoder, err := viper.NewCodecRegistry().Decoder(format)
	if err != nil {
		t.Fatal(err)
	}
	data := make(map[string]any)
	if err = decoder.Decode(contents, data); err != nil {
		t.Fatal(err)
	}
	return data
}

func migrationBackups(t *testing.T, file string) []string {
	t.Helper()
	files, err := filepath.Glob(file + ".tls-backup-*.bak")
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func boolMigrationPointer(value bool) *bool { return &value }
