/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestGenerateGrafanaPassword(t *testing.T) {
	original := monitoringRandomReader
	monitoringRandomReader = bytes.NewReader(bytes.Repeat([]byte{0xa5}, 32))
	t.Cleanup(func() { monitoringRandomReader = original })

	password, err := generateGrafanaPassword()
	if err != nil {
		t.Fatal(err)
	}
	if len(password) != 43 {
		t.Fatalf("expected 43-character unpadded base64url password, got %d", len(password))
	}
	if strings.ContainsAny(password, "+/=") {
		t.Fatalf("password is not unpadded base64url: %q", password)
	}

	directory := t.TempDir()
	if err = writeGrafanaPassword(directory, password); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, grafanaPasswordFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != password {
		t.Fatalf("password file contains unexpected data %q", contents)
	}
	if err = validateMonitoringSecretFile(filepath.Join(directory, grafanaPasswordFile)); err != nil {
		t.Fatalf("new password file did not pass validation: %v", err)
	}
}

func TestGenerateGrafanaPasswordEntropyFailure(t *testing.T) {
	original := monitoringRandomReader
	monitoringRandomReader = &failingReader{}
	t.Cleanup(func() { monitoringRandomReader = original })
	if password, err := generateGrafanaPassword(); err == nil || password != "" {
		t.Fatalf("expected entropy failure without a password, got %q, %v", password, err)
	}
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestMonitoringConfigBaseURL(t *testing.T) {
	tests := []struct {
		version string
		want    string
		valid   bool
	}{
		{"1.10.11", "https://raw.githubusercontent.com/oracle/coherence-cli/refs/tags/1.10.11/monitoring", true},
		{"1.11.0-RC1", "https://raw.githubusercontent.com/oracle/coherence-cli/refs/tags/1.11.0-RC1/monitoring", true},
		{" 1.10.11 ", "https://raw.githubusercontent.com/oracle/coherence-cli/refs/tags/1.10.11/monitoring", true},
		{"", "", false},
		{"v1.10.11", "", false},
		{"../heads/main", "", false},
		{"1.10", "", false},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			actual, err := monitoringConfigBaseURL(test.version)
			if test.valid {
				if err != nil {
					t.Fatal(err)
				}
				if actual != test.want {
					t.Fatalf("expected %q, got %q", test.want, actual)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected version %q to be rejected, got %q", test.version, actual)
			}
		})
	}
}

func TestValidateMonitoringSecretPermissions(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("POSIX permissions are not meaningful on Windows")
	}
	file := filepath.Join(t.TempDir(), grafanaPasswordFile)
	validPassword := bytes.Repeat([]byte("a"), 43)
	if err := os.WriteFile(file, validPassword, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateMonitoringSecretFile(file); err == nil || !strings.Contains(err.Error(), "insecure permissions") {
		t.Fatalf("expected insecure permission failure, got %v", err)
	}
	if err := os.Chmod(file, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateMonitoringSecretFile(file); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "password-link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if err := validateMonitoringSecretFile(link); err == nil || !strings.Contains(err.Error(), "non-symbolic-link") {
		t.Fatalf("expected symlink failure, got %v", err)
	}
	empty := filepath.Join(t.TempDir(), "empty-password")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateMonitoringSecretFile(empty); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty password failure, got %v", err)
	}
}

func TestMonitoringBindAddress(t *testing.T) {
	tests := []struct {
		input       string
		want        string
		wantWarning string
		wantURL     string
	}{
		{"127.0.0.1", "127.0.0.1", "", "http://127.0.0.1:3000/api/health"},
		{"::1", "::1", "", "http://[::1]:3000/api/health"},
		{"192.0.2.15", "192.0.2.15", "non-loopback", "http://192.0.2.15:3000/api/health"},
		{"0.0.0.0", "0.0.0.0", "every network interface", "http://127.0.0.1:3000/api/health"},
		{"::", "::", "every network interface", "http://[::1]:3000/api/health"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			address, err := parseMonitoringBindAddress(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if address != test.want {
				t.Fatalf("got address %q, want %q", address, test.want)
			}
			warning := monitoringExposureWarning(address)
			if !strings.Contains(warning, test.wantWarning) {
				t.Fatalf("got warning %q, want it to contain %q", warning, test.wantWarning)
			}
			if url := monitoringURL(address, 3000, "/api/health"); url != test.wantURL {
				t.Fatalf("got URL %q, want %q", url, test.wantURL)
			}
		})
	}
	if _, err := parseMonitoringBindAddress("localhost"); err == nil {
		t.Fatal("expected hostname to be rejected")
	}
	if _, err := parseMonitoringBindAddress(""); err == nil {
		t.Fatal("expected an explicitly empty address to be rejected")
	}
	t.Setenv(monitoringBindAddressEnv, "")
	if _, err := monitoringBindAddressFromEnvironment(); err == nil {
		t.Fatal("expected an explicitly empty environment value to be rejected")
	}
	if err := os.Unsetenv(monitoringBindAddressEnv); err != nil {
		t.Fatal(err)
	}
	if address, err := monitoringBindAddressFromEnvironment(); err != nil || address != defaultMonitoringAddress {
		t.Fatalf("expected unset environment to default to loopback, got %q, %v", address, err)
	}
}

func TestLegacyPortWarningsUseEffectiveBindAddress(t *testing.T) {
	t.Setenv(monitoringBindAddressEnv, "")
	tracked := trackedCompose(t)
	if warnings := legacyPortWarnings(tracked); len(warnings) != 0 {
		t.Fatalf("tracked loopback defaults unexpectedly warned: %v", warnings)
	}
	t.Setenv(monitoringBindAddressEnv, "0.0.0.0")
	if warnings := legacyPortWarnings(tracked); len(warnings) != 2 {
		t.Fatalf("expected warnings for both inherited wildcard listeners, got %v", warnings)
	}
	shortPorts := []byte(`services:
  grafana:
    ports: ["3000:3000"]
  prometheus:
    ports: ["127.0.0.1:9090:9090"]
`)
	if warnings := legacyPortWarnings(shortPorts); len(warnings) != 1 || !strings.Contains(warnings[0], "grafana") {
		t.Fatalf("expected only Grafana wildcard warning, got %v", warnings)
	}
}

func TestMonitoringBindAddressPersistence(t *testing.T) {
	directory := t.TempDir()
	if err := writeMonitoringEnv(directory, "2001:db8::10"); err != nil {
		t.Fatal(err)
	}
	address, err := readMonitoringEnv(directory)
	if err != nil || address != "2001:db8::10" {
		t.Fatalf("persisted address = %q, %v", address, err)
	}
	if err = os.WriteFile(filepath.Join(directory, monitoringEnvFile), []byte(monitoringBindAddressEnv+"=\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = readMonitoringEnv(directory); err == nil {
		t.Fatal("expected empty persisted address to fail")
	}
	duplicate := monitoringBindAddressEnv + "=127.0.0.1\n" + monitoringBindAddressEnv + "=0.0.0.0\n"
	if err = os.WriteFile(filepath.Join(directory, monitoringEnvFile), []byte(duplicate), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = readMonitoringEnv(directory); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("expected duplicate persisted address to fail, got %v", err)
	}
}

func TestSanitizeGrafanaINI(t *testing.T) {
	input := []byte(`[server]
http_port = 3000
admin_password = misplaced-secret

[security]
admin_user = attacker
admin_password = admin
admin_password = another-secret
cookie_secure = true

[users]
allow_sign_up = false
`)
	result := string(sanitizeGrafanaINI(input))
	if strings.Contains(result, "admin_password") || strings.Contains(result, "attacker") {
		t.Fatalf("credentials were not removed from INI:\n%s", result)
	}
	if strings.Count(result, "admin_user = admin") != 1 {
		t.Fatalf("expected exactly one fixed admin user:\n%s", result)
	}
	if !strings.Contains(result, "cookie_secure = true") || !strings.Contains(result, "allow_sign_up = false") {
		t.Fatalf("unrelated settings were lost:\n%s", result)
	}
}

func TestGenerateSecureComposeSanitizesHostileInputs(t *testing.T) {
	input := []byte(`services:
  grafana:
    image: grafana/grafana:12.0.0
    networks:
      coherence:
    environment:
      - GF_SECURITY_ADMIN_USER=attacker
      - GF_SECURITY_ADMIN_PASSWORD=leaked-direct-password
      - GF_SECURITY_ADMIN_PASSWORD__FILE=/tmp/attacker-password
      - KEEP_ME=yes
    ports:
      - 0.0.0.0:3000:3000
      - 3001:3000
    secrets:
      - attacker_secret
    volumes:
      - ./grafana-admin-password:/tmp/attacker-password
      - ./grafana.ini:/etc/grafana/grafana.ini
      - ./datasources.yaml:/etc/grafana/provisioning/datasources/datasources.yaml
      - ./dashboards.yaml:/etc/grafana/provisioning/dashboards/dashboards.yaml
      - ./dashboards:/etc/grafana/provisioning/dashboards/dashboards
  prometheus:
    image: prom/prometheus:v3.4.1
    networks:
      coherence:
    command:
      - --config.file=/etc/prometheus/prometheus.yaml
    volumes:
      - ./prometheus.yaml:/etc/prometheus/prometheus.yaml
    ports:
      - 9090:9090
      - 9091:9090
    secrets:
      - attacker_secret
secrets:
  grafana_admin_password:
    file: ./attacker-password
  unrelated_secret:
    file: ./unrelated
networks:
  coherence:
`)
	generated, err := generateSecureCompose(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "leaked-direct-password") || strings.Contains(string(generated), "/tmp/attacker-password") {
		t.Fatalf("generated compose retains attacker-controlled credentials:\n%s", generated)
	}

	var document map[string]any
	if err = yaml.Unmarshal(generated, &document); err != nil {
		t.Fatal(err)
	}
	services, _ := stringMap(document["services"])
	grafana, _ := stringMap(services["grafana"])
	prometheus, _ := stringMap(services["prometheus"])
	assertOneSecurePort(t, grafana, grafanaPort)
	assertOneSecurePort(t, prometheus, prometheusPort)

	environment, _ := stringMap(grafana["environment"])
	if environment["GF_SECURITY_ADMIN_USER"] != grafanaAdminUser || environment["GF_SECURITY_ADMIN_PASSWORD__FILE"] != grafanaAdminFileTarget {
		t.Fatalf("Grafana secure environment is incorrect: %#v", environment)
	}
	if _, exists := environment["GF_SECURITY_ADMIN_PASSWORD"]; exists {
		t.Fatalf("direct password remains in environment: %#v", environment)
	}
	if environment["KEEP_ME"] != "yes" {
		t.Fatalf("unrelated environment was lost: %#v", environment)
	}
	if _, exists := prometheus["secrets"]; exists {
		t.Fatalf("Prometheus unexpectedly received secrets: %#v", prometheus["secrets"])
	}
	grafanaSecrets, ok := grafana["secrets"].([]any)
	if !ok || len(grafanaSecrets) != 1 {
		t.Fatalf("Grafana must receive exactly one secret: %#v", grafana["secrets"])
	}
	grant, _ := stringMap(grafanaSecrets[0])
	if grant["source"] != grafanaPasswordSecret || grant["target"] != grafanaPasswordSecret {
		t.Fatalf("unexpected Grafana secret grant: %#v", grant)
	}
	secrets, _ := stringMap(document["secrets"])
	passwordSecret, _ := stringMap(secrets[grafanaPasswordSecret])
	if passwordSecret["file"] != "./"+grafanaPasswordFile {
		t.Fatalf("unexpected password secret source: %#v", passwordSecret)
	}
}

func TestGenerateSecureComposeRejectsUnapprovedServiceFields(t *testing.T) {
	compose := trackedCompose(t)
	compose = bytes.Replace(compose, []byte("    image: grafana/grafana:"), []byte("    privileged: true\n    image: grafana/grafana:"), 1)
	if _, err := generateSecureCompose(compose); err == nil || !strings.Contains(err.Error(), "unsupported field") {
		t.Fatalf("expected unsupported field rejection, got %v", err)
	}
}

func TestGenerateSecureComposeNormalizesMapEnvironment(t *testing.T) {
	compose := trackedCompose(t)
	compose = bytes.Replace(compose, []byte("    image: grafana/grafana:"), []byte(`    environment:
      GF_SECURITY_ADMIN_USER: attacker
      GF_SECURITY_ADMIN_PASSWORD: direct-secret
      GF_SECURITY_ADMIN_PASSWORD__FILE: /tmp/alternate
      KEEP_ME: preserved
    image: grafana/grafana:`), 1)
	generated, err := generateSecureCompose(compose)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = yaml.Unmarshal(generated, &document); err != nil {
		t.Fatal(err)
	}
	services, _ := stringMap(document["services"])
	grafana, _ := stringMap(services["grafana"])
	environment, _ := stringMap(grafana["environment"])
	if len(environment) != 3 || environment["GF_SECURITY_ADMIN_USER"] != grafanaAdminUser ||
		environment["GF_SECURITY_ADMIN_PASSWORD__FILE"] != grafanaAdminFileTarget || environment["KEEP_ME"] != "preserved" {
		t.Fatalf("map environment was not normalized: %#v", environment)
	}
}

func assertOneSecurePort(t *testing.T, service map[string]any, port int) {
	t.Helper()
	ports, ok := service["ports"].([]any)
	if !ok || len(ports) != 1 {
		t.Fatalf("expected one authoritative port, got %#v", service["ports"])
	}
	mapping, _ := stringMap(ports[0])
	if mapping["target"] != port || mapping["published"] != strconv.Itoa(port) {
		t.Fatalf("unexpected port mapping: %#v", mapping)
	}
	if mapping["host_ip"] != "${COHCTL_MONITORING_BIND_ADDRESS:?monitoring bind address is required}" {
		t.Fatalf("unexpected host IP mapping: %#v", mapping)
	}
}

// Both start and get call validateEnvironment before Docker or HTTP operations.
func TestMonitoringBundleRejectsNonRegularArtifacts(t *testing.T) {
	generatedFiles := []string{generatedDockerComposeYAML, monitoringEnvFile, grafanaPasswordFile}
	artifactTypes := []struct {
		name   string
		create func(*testing.T, string)
	}{
		{"directory", func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"valid symlink", func(t *testing.T, path string) {
			target := filepath.Join(t.TempDir(), "target")
			if err := os.WriteFile(target, minimalLegacyCompose(), 0600); err != nil {
				t.Fatal(err)
			}
			monitoringTestSymlink(t, target, path)
		}},
		{"dangling symlink", func(t *testing.T, path string) {
			monitoringTestSymlink(t, path+"-absent", path)
		}},
	}
	for _, artifact := range artifactTypes {
		t.Run(artifact.name, func(t *testing.T) {
			check := func(t *testing.T, invalid []string, regularOthers bool) {
				directory := monitoringValidationFixture(t)
				for _, name := range generatedFiles {
					isInvalid := false
					for _, invalidName := range invalid {
						isInvalid = isInvalid || name == invalidName
					}
					if isInvalid {
						artifact.create(t, filepath.Join(directory, name))
					} else if regularOthers {
						writeMonitoringTestArtifact(t, directory, name)
					}
				}
				mode, err := classifyMonitoringBundle(directory)
				if mode != monitoringBundleInvalid {
					t.Errorf("expected invalid mode, got %v", mode)
				}
				assertInvalidMonitoringArtifact(t, err, invalid[0])
				mon := newMonitoringAt(&cobra.Command{}, directory)
				assertInvalidMonitoringArtifact(t, mon.validateEnvironment(), invalid[0])

				// Stop retains its separate parseable-model rules, including
				// following a valid Compose symlink for recovery.
				wantGenerated := regularOthers && invalid[0] != generatedDockerComposeYAML ||
					artifact.name == "valid symlink" && (invalid[0] == generatedDockerComposeYAML || len(invalid) == len(generatedFiles))
				wantBase := dockerComposeYAML
				if wantGenerated {
					wantBase = generatedDockerComposeYAML
				}
				file, address, generated, err := mon.composeForStop()
				if err != nil || filepath.Base(file) != wantBase || generated != wantGenerated || address != defaultMonitoringAddress {
					t.Fatalf("unexpected stop selection file=%s generated=%v address=%s error=%v", file, generated, address, err)
				}
			}
			for _, name := range generatedFiles {
				t.Run(name+"/alone", func(t *testing.T) { check(t, []string{name}, false) })
				t.Run(name+"/mixed", func(t *testing.T) { check(t, []string{name}, true) })
			}
			t.Run("all", func(t *testing.T) { check(t, generatedFiles, false) })
		})
	}
}

func monitoringTestSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.ENOSYS) {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
}

func assertInvalidMonitoringArtifact(t *testing.T, err error, name string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), name) ||
		!strings.Contains(err.Error(), "regular non-symbolic-link file") || !strings.Contains(err.Error(), "cohctl init monitoring") {
		t.Fatalf("expected actionable non-regular bundle error naming %s, got %v", name, err)
	}
}

func monitoringValidationFixture(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	mon := newMonitoringAt(&cobra.Command{}, directory)
	if err := mon.ensureDirectories(); err != nil {
		t.Fatal(err)
	}
	for _, name := range dashboardFiles {
		if err := os.WriteFile(filepath.Join(mon.dashboardsDir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range dockerComposeFiles {
		if err := os.WriteFile(filepath.Join(directory, name), minimalLegacyCompose(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func writeMonitoringTestArtifact(t *testing.T, directory, name string) {
	t.Helper()
	var contents []byte
	switch name {
	case generatedDockerComposeYAML:
		contents = secureCompose(t)
	case monitoringEnvFile:
		contents = []byte(monitoringBindAddressEnv + "=" + defaultMonitoringAddress + "\n")
	case grafanaPasswordFile:
		contents = bytes.Repeat([]byte("a"), 43)
	default:
		t.Fatalf("unexpected generated artifact: %s", name)
	}
	if err := os.WriteFile(filepath.Join(directory, name), contents, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyMonitoringBundle(t *testing.T) {
	generatedFiles := []string{generatedDockerComposeYAML, monitoringEnvFile, grafanaPasswordFile}
	for present := 0; present < 8; present++ {
		t.Run(strconv.Itoa(present), func(t *testing.T) {
			directory := monitoringValidationFixture(t)
			for index, name := range generatedFiles {
				if present&(1<<index) != 0 {
					writeMonitoringTestArtifact(t, directory, name)
				}
			}
			mode, err := classifyMonitoringBundle(directory)
			mon := newMonitoringAt(&cobra.Command{}, directory)
			if present != 0 && present != 7 {
				if mode != monitoringBundleInvalid || err == nil || !strings.Contains(err.Error(), "incomplete") {
					t.Fatalf("expected partial regular bundle rejection, got mode=%v error=%v", mode, err)
				}
				if err := mon.validateEnvironment(); err == nil || !strings.Contains(err.Error(), "incomplete") {
					t.Fatalf("expected start/get rejection, got %v", err)
				}
				return
			}
			wantMode, wantBase := monitoringBundleLegacy, dockerComposeYAML
			if present == 7 {
				wantMode, wantBase = monitoringBundleGenerated, generatedDockerComposeYAML
			}
			if err != nil || mode != wantMode {
				t.Fatalf("expected mode=%v, got mode=%v error=%v", wantMode, mode, err)
			}
			if err := mon.validateEnvironment(); err != nil {
				t.Fatal(err)
			}
			if mon.bundleMode != wantMode || filepath.Base(mon.composeFile) != wantBase || mon.bindAddress != defaultMonitoringAddress {
				t.Fatalf("unexpected start/get selection: %+v", mon)
			}
			// A complete generated bundle does not depend on retained legacy Compose.
			if err := os.Remove(filepath.Join(directory, dockerComposeYAML)); err != nil {
				t.Fatal(err)
			}
			mode, err = classifyMonitoringBundle(directory)
			if present == 0 {
				if mode != monitoringBundleInvalid || err == nil {
					t.Fatalf("expected absent installation rejection, got mode=%v error=%v", mode, err)
				}
			} else if err != nil || mode != monitoringBundleGenerated {
				t.Fatalf("expected generated mode without legacy, got mode=%v error=%v", mode, err)
			}
		})
	}
}

func TestComposeForStopSelection(t *testing.T) {
	tests := []struct {
		name          string
		generated     []byte
		legacy        []byte
		wantBase      string
		wantGenerated bool
		wantError     bool
	}{
		{"generated only", secureCompose(t), nil, generatedDockerComposeYAML, true, false},
		{"legacy only", nil, minimalLegacyCompose(), dockerComposeYAML, false, false},
		{"mixed prefers generated", secureCompose(t), minimalLegacyCompose(), generatedDockerComposeYAML, true, false},
		{"corrupt generated falls back", []byte("not: [yaml"), minimalLegacyCompose(), dockerComposeYAML, false, false},
		{"no usable model", []byte("not: [yaml"), []byte("services: {}"), "", false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			if test.generated != nil {
				if err := os.WriteFile(filepath.Join(directory, generatedDockerComposeYAML), test.generated, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if test.legacy != nil {
				if err := os.WriteFile(filepath.Join(directory, dockerComposeYAML), test.legacy, 0600); err != nil {
					t.Fatal(err)
				}
			}
			mon := newMonitoringAt(&cobra.Command{}, directory)
			file, address, generated, err := mon.composeForStop()
			if test.wantError {
				if err == nil {
					t.Fatal("expected stop selection to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(file) != test.wantBase || generated != test.wantGenerated || address != defaultMonitoringAddress {
				t.Fatalf("unexpected selection file=%s generated=%v address=%s", file, generated, address)
			}
		})
	}
}

// Recovery follows symlinks to regular files; start/get retain stricter classification.
func TestComposeForStopRecoveryFiles(t *testing.T) {
	for _, test := range []struct {
		name, generated, legacy, env, wantBase, wantAddress string
		wantEnv                                             bool
	}{
		{"generated directory", "directory", "regular", "missing", dockerComposeYAML, defaultMonitoringAddress, false},
		{"legacy directory", "missing", "directory", "missing", "", "", false},
		{"both directories", "directory", "directory", "missing", "", "", false},
		{"generated symlink", "symlink", "regular", "regular", generatedDockerComposeYAML, "::1", true},
		{"legacy symlink", "missing", "symlink", "missing", dockerComposeYAML, defaultMonitoringAddress, false},
		{"dangling generated symlink", "dangling", "regular", "missing", dockerComposeYAML, defaultMonitoringAddress, false},
		{"dangling legacy symlink", "missing", "dangling", "missing", "", "", false},
		{"missing environment", "regular", "regular", "missing", generatedDockerComposeYAML, defaultMonitoringAddress, false},
		{"invalid environment", "regular", "regular", "invalid", generatedDockerComposeYAML, defaultMonitoringAddress, false},
		{"environment directory", "regular", "regular", "directory", generatedDockerComposeYAML, defaultMonitoringAddress, false},
		{"dangling environment symlink", "regular", "regular", "dangling", generatedDockerComposeYAML, defaultMonitoringAddress, false},
		{"environment symlink", "regular", "regular", "symlink", generatedDockerComposeYAML, "::1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(monitoringBindAddressEnv, "0.0.0.0")
			directory := t.TempDir()
			for name, kind := range map[string]string{generatedDockerComposeYAML: test.generated, dockerComposeYAML: test.legacy, monitoringEnvFile: test.env} {
				file := filepath.Join(directory, name)
				contents := minimalLegacyCompose()
				if name == generatedDockerComposeYAML {
					contents = secureCompose(t)
				}
				if name == monitoringEnvFile {
					contents = []byte(monitoringBindAddressEnv + "=::1\n")
				}
				switch kind {
				case "missing":
					continue
				case "directory":
					if err := os.Mkdir(file, 0700); err != nil {
						t.Fatal(err)
					}
					continue
				case "dangling":
					monitoringTestSymlink(t, file+"-absent", file)
					continue
				case "symlink":
					target := file + "-target"
					monitoringTestSymlink(t, target, file)
					file = target
				case "invalid":
					contents = []byte(monitoringBindAddressEnv + "=invalid\n")
				}
				if err := os.WriteFile(file, contents, 0600); err != nil {
					t.Fatal(err)
				}
			}
			assertMonitoringRecovery(t, directory, test.wantBase, test.wantAddress, test.wantEnv)
		})
	}
}

func assertMonitoringRecovery(t *testing.T, directory, wantBase, wantAddress string, wantEnv bool) {
	t.Helper()
	mon := newMonitoringAt(&cobra.Command{}, directory)
	file, address, generated, err := mon.composeForStop()
	if wantBase == "" {
		if err == nil || file != "" || address != "" || generated ||
			!strings.Contains(err.Error(), filepath.Join(directory, generatedDockerComposeYAML)) ||
			!strings.Contains(err.Error(), filepath.Join(directory, dockerComposeYAML)) ||
			!strings.Contains(err.Error(), "restore one of these files") {
			t.Fatalf("expected actionable recovery error for both models, got file=%s address=%s generated=%v error=%v", file, address, generated, err)
		}
		return
	}
	if err != nil || file != filepath.Join(directory, wantBase) || address != wantAddress || generated != (wantBase == generatedDockerComposeYAML) {
		t.Fatalf("unexpected recovery selection: file=%s address=%s generated=%v error=%v", file, address, generated, err)
	}
	assertMonitoringRecoveryArgs(t, mon, file, address, wantEnv)
}

func assertMonitoringRecoveryArgs(t *testing.T, mon *monitoring, file, address string, wantEnv bool) {
	t.Helper()
	args, environment := mon.composeArgs(file, address, []string{"down"})
	wantArgs := []string{"-f", file, "down"}
	if wantEnv {
		wantArgs = append([]string{"--env-file", filepath.Join(filepath.Dir(file), monitoringEnvFile)}, wantArgs...)
	}
	if strings.Join(args, "\n") != strings.Join(wantArgs, "\n") {
		t.Fatalf("unexpected recovery arguments: %q, want %q", args, wantArgs)
	}
	if filepath.Base(file) == generatedDockerComposeYAML {
		if len(environment) != 1 || environment[0] != monitoringBindAddressEnv+"="+address {
			t.Fatalf("expected explicit validated/default bind address, got %q", environment)
		}
	} else if len(environment) != 0 {
		t.Fatalf("unexpected legacy environment override: %q", environment)
	}
}

func TestInstallStagedDirectoryReplacesCompleteDirectory(t *testing.T) {
	parent := t.TempDir()
	active := filepath.Join(parent, "monitoring")
	staged := filepath.Join(parent, "staged")
	if err := os.Mkdir(active, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(active, "old"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(staged, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staged, "new"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	mon := newMonitoringAt(&cobra.Command{}, active)
	if err := mon.installStagedDirectory(staged); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(active, "new")); err != nil {
		t.Fatalf("new directory was not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(active, "old")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old directory was not replaced: %v", err)
	}
}

func secureCompose(t *testing.T) []byte {
	t.Helper()
	generated, err := generateSecureCompose(trackedCompose(t))
	if err != nil {
		t.Fatal(err)
	}
	return generated
}

func trackedCompose(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "monitoring", dockerComposeYAML))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func minimalLegacyCompose() []byte {
	return []byte("services:\n  grafana:\n    image: grafana/grafana:1\n  prometheus:\n    image: prom/prometheus:1\n")
}

func TestParseComposeMajorVersion(t *testing.T) {
	for input, expected := range map[string]int{
		"2.38.2\n":                       2,
		"Docker Compose version v2.24.1": 2,
		"docker-compose version 1.29.2":  1,
	} {
		major, err := parseComposeMajorVersion(input)
		if err != nil || major != expected {
			t.Fatalf("parseComposeMajorVersion(%q) = %d, %v; want %d", input, major, err, expected)
		}
	}
	if _, err := parseComposeMajorVersion("unknown"); err == nil {
		t.Fatal("expected unknown Compose version to fail")
	}
}

func TestWithEnvironmentReplacesInheritedBindAddress(t *testing.T) {
	t.Setenv(monitoringBindAddressEnv, "0.0.0.0")
	environment := withEnvironment([]string{monitoringBindAddressEnv + "=127.0.0.1"})
	count := 0
	for _, entry := range environment {
		if strings.HasPrefix(entry, monitoringBindAddressEnv+"=") {
			count++
			if entry != monitoringBindAddressEnv+"=127.0.0.1" {
				t.Fatalf("inherited bind address survived: %q", entry)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one bind-address environment entry, got %d", count)
	}
}

func TestTrackedMonitoringTemplatesAreSafeByDefault(t *testing.T) {
	ini, err := os.ReadFile(filepath.Join("..", "..", "monitoring", "grafana.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(ini)), "admin_password") {
		t.Fatalf("tracked grafana.ini contains an administrator password:\n%s", ini)
	}
	compose, err := os.ReadFile(filepath.Join("..", "..", "monitoring", dockerComposeYAML))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(compose), "${COHCTL_MONITORING_BIND_ADDRESS:-127.0.0.1}"); count != 2 {
		t.Fatalf("expected both tracked ports to default to loopback, got %d:\n%s", count, compose)
	}
}

func TestGeneratedGrafanaRuntimeCannotBypassSecretDelivery(t *testing.T) {
	for _, field := range []string{"user", "entrypoint", "command", "cap_add", "cap_drop", "security_opt"} {
		t.Run(field, func(t *testing.T) {
			generated, err := generateSecureCompose(trackedCompose(t))
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = yaml.Unmarshal(generated, &document); err != nil {
				t.Fatal(err)
			}
			if err = validateSecureComposeDocument(document); err != nil {
				t.Fatal(err)
			}
			services, _ := stringMap(document["services"])
			grafana, _ := stringMap(services["grafana"])
			if field == "command" {
				grafana[field] = []any{"--config=/tmp/alternate.ini"}
			} else {
				delete(grafana, field)
			}
			if err = validateSecureComposeDocument(document); err == nil {
				t.Fatalf("accepted a generated model without authoritative %s", field)
			}
		})
	}
}

func TestGrafanaCredentialEntrypoint(t *testing.T) {
	if runtime.GOOS == managementTestWindows {
		t.Skip("Linux container shell is exercised by the Docker tests")
	}
	password := strings.Repeat("a", 43)
	for _, test := range []struct {
		name, contents              string
		readFailure, missing, valid bool
	}{
		{name: "valid", contents: password, valid: true},
		{name: "missing", missing: true},
		{name: "empty"},
		{name: "short", contents: "admin"},
		{name: "invalid alphabet", contents: strings.Repeat("!", 43)},
		{name: "failed read with partial output", contents: password, readFailure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			file := filepath.Join(directory, grafanaPasswordFile)
			if !test.missing {
				if err := os.WriteFile(file, []byte(test.contents), 0600); err != nil {
					t.Fatal(err)
				}
			}
			path := os.Getenv("PATH")
			if test.readFailure {
				// Simulate an I/O failure after cat has already emitted valid bytes.
				// #nosec G306 -- owner-only executable read-failure stub in t.TempDir.
				if err := os.WriteFile(filepath.Join(directory, "cat"), []byte("#!/bin/sh\n/bin/cat \"$@\"\nexit 1\n"), 0700); err != nil {
					t.Fatal(err)
				}
				path = directory + string(os.PathListSeparator) + path
			}
			script := grafanaSecretReadScript
			if test.readFailure {
				// Substitute only the executable for fault injection; production
				// uses /bin/cat so a caller-controlled PATH cannot affect root.
				script = strings.ReplaceAll(script, "/bin/cat ", "cat ")
			}
			command := exec.Command("/bin/sh", "-ec", script+`exec "$@"`, "cohctl-grafana", "/bin/sh", "-ec", `test "$GF_SECURITY_ADMIN_PASSWORD" = "$EXPECTED_PASSWORD"; test -z "${GF_SECURITY_ADMIN_PASSWORD__FILE+x}"; echo started`)
			command.Env = append(os.Environ(), "PATH="+path, "GF_SECURITY_ADMIN_PASSWORD__FILE="+file, "EXPECTED_PASSWORD="+password)
			output, err := command.CombinedOutput()
			if test.valid {
				if err != nil || string(output) != "started\n" {
					t.Fatalf("credential handoff failed: %v, %s", err, output)
				}
			} else if err == nil || strings.Contains(string(output), "started") || !strings.Contains(string(output), "administrator secret") {
				t.Fatalf("credential delivery did not fail closed: %v, %s", err, output)
			}
			if strings.Contains(string(output), password) {
				t.Fatal("credential leaked to output")
			}
		})
	}
}

func TestGrafanaBootstrapRejectsStartupEnvironmentHooks(t *testing.T) {
	for _, key := range []string{"LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES", "BASH_FUNC_hook%%", "BASH_ENV", "ENV", "SHELLOPTS", "BASHOPTS", "GCONV_PATH", "LOCPATH", "MUSL_LOCPATH", "NLSPATH", "GLIBC_TUNABLES"} {
		t.Run(key, func(t *testing.T) {
			compose := bytes.Replace(trackedCompose(t), []byte("    image: grafana/grafana:"), []byte("    environment:\n      "+key+": /etc/grafana/grafana.ini\n    image: grafana/grafana:"), 1)
			if _, err := generateSecureCompose(compose); err == nil || !strings.Contains(err.Error(), "does not permit environment") {
				t.Fatalf("accepted downloaded startup hook %s: %v", key, err)
			}
			generated, err := generateSecureCompose(trackedCompose(t))
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err = yaml.Unmarshal(generated, &document); err != nil {
				t.Fatal(err)
			}
			services, _ := stringMap(document["services"])
			grafana, _ := stringMap(services["grafana"])
			environment, _ := stringMap(grafana["environment"])
			environment[key] = "/etc/grafana/grafana.ini"
			if err = validateSecureComposeDocument(document); err == nil {
				t.Fatalf("accepted generated startup hook %s", key)
			}
		})
	}
}
