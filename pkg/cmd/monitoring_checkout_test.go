/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	checkoutMonitoringMarker  = "cohctl-checkout-fixture.invalid"
	checkoutComposeCommand    = "compose"
	checkoutGrafanaService    = "grafana"
	checkoutPrometheusService = "prometheus"
)

// Command output includes the one-time password. Keep it in memory, including
// on failure, and synchronize subprocess stream writers. Never upload this data.
type monitoringCheckoutOutput struct {
	sync.Mutex
	buffer bytes.Buffer
}

func (o *monitoringCheckoutOutput) Write(p []byte) (int, error) {
	o.Lock()
	defer o.Unlock()
	return o.buffer.Write(p)
}

func (o *monitoringCheckoutOutput) text() string {
	o.Lock()
	defer o.Unlock()
	return o.buffer.String()
}

type monitoringCheckoutFixture struct {
	t          *testing.T
	command    *cobra.Command
	output     *monitoringCheckoutOutput
	mon        *monitoring
	inputs     map[string][]byte
	requests   map[string]int
	dashboards func(string) ([]byte, error)
}

func newMonitoringCheckoutFixture(t *testing.T) *monitoringCheckoutFixture {
	t.Helper()
	oldDirectory, oldVersion := cfgDirectory, Version
	oldConfirm, oldStandalone, oldOpen := automaticallyConfirm, useDockerComposeParam, openHomePageParam
	oldTimeout := fetcher.RequestTimeout
	t.Cleanup(func() {
		cfgDirectory, Version = oldDirectory, oldVersion
		automaticallyConfirm, useDockerComposeParam, openHomePageParam = oldConfirm, oldStandalone, oldOpen
		fetcher.RequestTimeout = oldTimeout
	})
	// This syntactically valid version has no corresponding published fixture tag.
	cfgDirectory, Version = t.TempDir(), "0.0.0-checkout-fixture"
	automaticallyConfirm, useDockerComposeParam, openHomePageParam = true, false, false
	fetcher.RequestTimeout = 15
	t.Setenv("COMPOSE_PROJECT_NAME", fmt.Sprintf("cohctl-checkout-%d", time.Now().UnixNano()))
	t.Setenv(monitoringBindAddressEnv, "")
	if err := os.Unsetenv(monitoringBindAddressEnv); err != nil {
		t.Fatal(err)
	}
	f := &monitoringCheckoutFixture{t: t, output: &monitoringCheckoutOutput{}, command: &cobra.Command{}, inputs: map[string][]byte{}, requests: map[string]int{}}
	f.command.SetOut(f.output)
	f.command.SetErr(f.output)
	f.mon = newMonitoring(f.command)
	for _, name := range dockerComposeFiles {
		contents, err := os.ReadFile(filepath.Join("..", "..", "monitoring", name))
		if err != nil {
			t.Fatal(err)
		}
		f.inputs[name] = contents
	}
	// Mutate inputs BEFORE the shared initializer stages them. Marker propagation
	// proves that neither released templates nor post-init replacement can pass.
	f.inputs["grafana.ini"] = bytes.ReplaceAll(f.inputs["grafana.ini"], []byte("instance_name = Coherence Monitoring"), []byte("instance_name = "+checkoutMonitoringMarker))
	f.inputs["grafana.ini"] = append(f.inputs["grafana.ini"], []byte("\nadmin_password = fixture-password-must-be-removed\n")...)
	var model map[string]any
	if err := yaml.Unmarshal(f.inputs[dockerComposeYAML], &model); err != nil {
		t.Fatal(err)
	}
	services, _ := stringMap(model["services"])
	grafana, _ := stringMap(services[checkoutGrafanaService])
	grafana["environment"] = map[string]any{"GF_SERVER_DOMAIN": checkoutMonitoringMarker, "GF_SECURITY_ADMIN_PASSWORD": "fixture-password-must-be-removed"}
	grafana["ports"] = []any{"0.0.0.0:3000:3000"}
	prometheus, _ := stringMap(services[checkoutPrometheusService])
	prometheus["ports"] = []any{"0.0.0.0:9090:9090"}
	var err error
	f.inputs[dockerComposeYAML], err = yaml.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *monitoringCheckoutFixture) readURL(url string) ([]byte, error) {
	f.requests[url]++
	base, err := monitoringConfigBaseURL(Version)
	if err != nil {
		return nil, err
	}
	for name, contents := range f.inputs {
		if url == base+"/"+name {
			return contents, nil
		}
	}
	for _, name := range dashboardFiles {
		if url == dashboardBaseURL+"/"+name {
			return f.dashboards(url)
		}
	}
	return nil, fmt.Errorf("unexpected monitoring fixture URL %s", url)
}

func (f *monitoringCheckoutFixture) assertInstalled() string {
	f.t.Helper()
	if len(f.requests) != len(dockerComposeFiles)+len(dashboardFiles) {
		f.t.Fatal("initializer did not request every checkout support file and Operator dashboard")
	}
	for _, count := range f.requests {
		if count != 1 {
			f.t.Fatal("unexpected repeated fixture request")
		}
	}
	if err := f.mon.validateEnvironment(); err != nil {
		f.t.Fatal(err)
	}
	if f.mon.bundleMode != monitoringBundleGenerated {
		f.t.Fatal("initializer did not install a generated bundle")
	}
	ini, err := os.ReadFile(filepath.Join(f.mon.monitoringDir, "grafana.ini"))
	if err != nil {
		f.t.Fatal(err)
	}
	if !bytes.Contains(ini, []byte(checkoutMonitoringMarker)) || bytes.Contains(ini, []byte("admin_password")) {
		f.t.Fatal("checkout Grafana marker or INI sanitization missing")
	}
	generated, err := os.ReadFile(f.mon.composeFile)
	if err != nil {
		f.t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte(checkoutMonitoringMarker)) || bytes.Contains(generated, []byte("fixture-password-must-be-removed")) {
		f.t.Fatal("checkout Compose marker or credential sanitization missing")
	}
	for _, name := range []string{"dashboards.yaml", "datasources.yaml", "prometheus.yaml"} {
		contents, err := os.ReadFile(filepath.Join(f.mon.monitoringDir, name))
		if err != nil || !bytes.Equal(contents, f.inputs[name]) {
			f.t.Fatalf("checkout %s was not installed", name)
		}
	}
	matches, err := filepath.Glob(filepath.Join(cfgDirectory, ".monitoring-staging-*"))
	if err != nil || len(matches) != 0 {
		f.t.Fatal("initializer left staging directories")
	}
	password, err := os.ReadFile(filepath.Join(f.mon.monitoringDir, grafanaPasswordFile))
	if err != nil {
		f.t.Fatal(err)
	}
	if strings.Count(f.output.text(), "Grafana administrator password: "+string(password)) != 1 {
		f.t.Fatal("initializer did not display the generated password exactly once")
	}
	if bytes.Contains(generated, password) {
		f.t.Fatal("generated Compose contains the secret value")
	}
	return string(password)
}

func TestMonitoringCheckoutInitialization(t *testing.T) {
	if isWindows() {
		t.Skip("Docker command stub uses a POSIX shell")
	}
	f := newMonitoringCheckoutFixture(t)
	f.dashboards = func(string) ([]byte, error) { return []byte("{}"), nil }
	bin := t.TempDir()
	stub := "#!/bin/sh\ncase \"$*\" in\n*'version --short') echo 2.39.0 ;;\n*'ps --all -q'*|pull\\ *) ;;\n*) exit 1 ;;\nesac\n"
	// #nosec G306 -- executable Docker stub in a private test directory.
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if err := initializeMonitoring(f.command, f.readURL); err != nil {
		t.Fatal(err)
	}
	f.assertInstalled()
	t.Log("checkout inputs staged, sanitized, validated and installed without release-tag HTTP access (Docker and dashboard JSON are unit stubs)")
}

func TestMonitoringCheckoutInitializationRejectsInputs(t *testing.T) {
	if isWindows() {
		t.Skip("Docker command stub uses a POSIX shell")
	}
	for _, failure := range []string{"version", "download", "unsafe model"} {
		t.Run(failure, func(t *testing.T) {
			f := newMonitoringCheckoutFixture(t)
			f.dashboards = func(string) ([]byte, error) { return []byte("{}"), nil }
			bin := t.TempDir()
			stub := "#!/bin/sh\ncase \"$*\" in\n*'version --short') echo 2.39.0 ;;\n*'ps --all -q'*) ;;\n*) exit 1 ;;\nesac\n"
			// #nosec G306 -- executable Docker stub in a private test directory.
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(stub), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			readURL := f.readURL
			want := ""
			switch failure {
			case "version":
				want = "not a valid release tag"
				Version = "main"
			case "download":
				want = "fixture download failed"
				readURL = func(url string) ([]byte, error) {
					if strings.HasSuffix(url, "/grafana.ini") {
						return nil, errors.New("fixture download failed")
					}
					return f.readURL(url)
				}
			case "unsafe model":
				want = "unsupported"
				var document map[string]any
				if err := yaml.Unmarshal(f.inputs[dockerComposeYAML], &document); err != nil {
					t.Fatal(err)
				}
				services, _ := stringMap(document["services"])
				grafana, _ := stringMap(services[checkoutGrafanaService])
				grafana["privileged"] = true
				contents, err := yaml.Marshal(document)
				if err != nil {
					t.Fatal(err)
				}
				f.inputs[dockerComposeYAML] = contents
			}
			if err := initializeMonitoring(f.command, readURL); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("initialization error = %v, want %s", err, want)
			}
			entries, err := os.ReadDir(cfgDirectory)
			if err != nil || len(entries) != 0 {
				t.Fatal("failed initialization installed files or left staging")
			}
			if strings.Contains(f.output.text(), "Grafana administrator password:") {
				t.Fatal("failed initialization displayed a password")
			}
			if Version == "main" && len(f.requests) != 0 {
				t.Fatal("invalid version reached input retrieval")
			}
		})
	}
}

// Full services, not probe containers. Production RunE uses the same initializer
// with GetURLContents. Only coherence-cli support downloads are checkout fixtures;
// Operator dashboard JSON still comes from its separate repository/main URL.
func TestMonitoringCheckoutStack(t *testing.T) {
	if os.Getenv("COHCTL_TEST_MONITORING_DOCKER") != "1" {
		t.Skip("set COHCTL_TEST_MONITORING_DOCKER=1 for the full checkout monitoring stack")
	}
	f := newMonitoringCheckoutFixture(t)
	f.dashboards = GetURLContents
	docker := func(args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, "docker", args...)
		command.WaitDelay = 2 * time.Second
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("Docker %s failed: %v (output withheld to protect credentials)", args[0], err)
		}
		return output
	}
	compose := func(args ...string) []byte {
		t.Helper()
		options, _ := f.mon.composeArgs(filepath.Join(f.mon.monitoringDir, generatedDockerComposeYAML), defaultMonitoringAddress, args)
		return docker(append([]string{checkoutComposeCommand}, options...)...)
	}
	t.Cleanup(func() {
		if _, err := os.Stat(filepath.Join(f.mon.monitoringDir, generatedDockerComposeYAML)); err == nil {
			compose("down", "--volumes", "--remove-orphans")
		}
	})
	if err := initializeMonitoring(f.command, f.readURL); err != nil {
		t.Fatal(err)
	}
	password := f.assertInstalled()
	resolved := compose("config")
	if bytes.Count(resolved, []byte("host_ip: 127.0.0.1")) != 2 || bytes.Contains(resolved, []byte(password)) {
		t.Fatal("resolved model lost loopback listeners or leaked secret")
	}
	if err := startMonitoringCmd.RunE(f.command, nil); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	wait := func(url string) {
		t.Helper()
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			response, err := client.Get(url)
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return
				}
			}
			time.Sleep(time.Second)
		}
		t.Fatalf("service did not become healthy at %s (logs withheld)", url)
	}
	wait("http://127.0.0.1:3000/api/health")
	wait("http://127.0.0.1:9090/-/healthy")
	for _, service := range []struct{ name, port string }{{checkoutGrafanaService, "3000"}, {checkoutPrometheusService, "9090"}} {
		if strings.TrimSpace(string(compose("port", service.name, service.port))) != "127.0.0.1:"+service.port {
			t.Fatalf("%s listener is not loopback", service.name)
		}
	}
	compose("exec", "-T", checkoutPrometheusService, "sh", "-c", "test ! -e /run/secrets/grafana_admin_password")
	for _, service := range []string{checkoutGrafanaService, checkoutPrometheusService} {
		id := strings.TrimSpace(string(compose("ps", "-q", service)))
		config := docker("inspect", "--format", "{{json .Config}}", id)
		if !bytes.Contains(config, []byte(f.mon.composeFile)) {
			t.Fatal("running service did not use the generated Compose model")
		}
		if bytes.Contains(config, []byte(password)) {
			t.Fatal("secret leaked into Docker-configured arguments/environment")
		}
		if service == checkoutGrafanaService && bytes.Contains(config, []byte("GF_PLUGINS_PREINSTALL_DISABLED=")) {
			t.Fatal("full-stack fixture must retain production plugin defaults")
		}
		if service == checkoutGrafanaService && !bytes.Contains(config, []byte("GF_SERVER_DOMAIN="+checkoutMonitoringMarker)) {
			t.Fatal("running Grafana did not consume checkout Compose input")
		}
		mounts := docker("inspect", "--format", "{{json .Mounts}}", id)
		if service == checkoutPrometheusService && bytes.Contains(mounts, []byte(grafanaPasswordFile)) {
			t.Fatal("Prometheus received the Grafana host secret")
		}
	}
	for _, auth := range []struct {
		password string
		status   int
	}{{password, http.StatusOK}, {grafanaAdminUser, http.StatusUnauthorized}} {
		request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:3000/api/org", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.SetBasicAuth(grafanaAdminUser, auth.password)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != auth.status {
			t.Fatalf("authentication returned %d, want %d", response.StatusCode, auth.status)
		}
	}
	beforeStatus := len(f.output.text())
	if err := getMonitoringCmd.RunE(f.command, nil); err != nil {
		t.Fatal(err)
	}
	status := f.output.text()[beforeStatus:]
	for _, want := range []string{f.mon.composeFile, "Status:   ok, version=", "Prometheus Server is Healthy."} {
		if !strings.Contains(status, want) {
			t.Fatalf("get monitoring did not report %q", want)
		}
	}
	if strings.Contains(status, password) {
		t.Fatal("get monitoring displayed the secret")
	}
	if err := validateMonitoringSecretFile(filepath.Join(f.mon.monitoringDir, grafanaPasswordFile)); err != nil {
		t.Fatal(err)
	}
	if !isWindows() {
		info, err := os.Stat(filepath.Join(f.mon.monitoringDir, grafanaPasswordFile))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("host secret is not protected with mode 0600")
		}
	}
	if err := stopMonitoringCmd.RunE(f.command, nil); err != nil {
		t.Errorf("stop monitoring failed (output withheld to protect credentials)")
	}
	if len(bytes.TrimSpace(compose("ps", "--all", "-q"))) != 0 {
		t.Fatal("stop monitoring left containers")
	}
	if !t.Failed() {
		t.Log("full checkout init/start/get/stop passed; generated-password HTTP 200, admin/admin HTTP 401; two loopback listeners, protected host secret and Prometheus isolation verified; stop handler removed all fixture containers")
	}
}
