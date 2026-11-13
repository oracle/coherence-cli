/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// This uses checkout fixtures and the real generated bootstrap/Compose lifecycle.
// It exercises init's rejection, then stages/installs locally without downloads
// or image pulls; it is not a full download-based CLI initialization test.
func TestGrafanaStoppedContainerPasswordRotation(t *testing.T) {
	f := newGrafanaDockerFixture(t)
	f.mustDocker("image", "inspect", f.image)
	originalDirectory, originalVersion := cfgDirectory, Version
	originalConfirm, originalStandalone := automaticallyConfirm, useDockerComposeParam
	t.Cleanup(func() {
		cfgDirectory, Version = originalDirectory, originalVersion
		automaticallyConfirm, useDockerComposeParam = originalConfirm, originalStandalone
	})
	cfgDirectory, Version = t.TempDir(), "1.10.11"
	automaticallyConfirm, useDockerComposeParam = true, false
	t.Setenv("COMPOSE_PROJECT_NAME", f.volume+"-rotation")
	t.Setenv(monitoringBindAddressEnv, defaultMonitoringAddress)
	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	command.SetErr(&output)
	mon := newMonitoring(command)
	composeFile := filepath.Join(mon.monitoringDir, generatedDockerComposeYAML)
	compose := func(args ...string) []byte {
		t.Helper()
		options, _ := mon.composeArgs(composeFile, defaultMonitoringAddress, args)
		return f.mustDocker(append([]string{"compose"}, options...)...)
	}
	t.Cleanup(func() {
		// Restore hidden models even when a fallback assertion fails, so cleanup
		// can always remove this fixture's exact Compose project.
		for _, file := range []string{composeFile, filepath.Join(mon.monitoringDir, dockerComposeYAML)} {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				if _, savedErr := os.Stat(file + ".saved"); savedErr == nil {
					if err = os.Rename(file+".saved", file); err != nil {
						t.Error(err)
					}
				}
			}
		}
		compose("down", "--volumes")
	})

	stage := func() (string, string) {
		t.Helper()
		directory, err := os.MkdirTemp(cfgDirectory, ".rotation-staging-")
		if err != nil {
			t.Fatal(err)
		}
		staged := newMonitoringAt(command, directory)
		if err = staged.ensureDirectories(); err != nil {
			t.Fatal(err)
		}
		for _, name := range dockerComposeFiles {
			contents, err := os.ReadFile(filepath.Join("..", "..", "monitoring", name))
			if err != nil {
				t.Fatal(err)
			}
			if err = writeFileContents(directory, name, contents); err != nil {
				t.Fatal(err)
			}
		}
		password, err := staged.createGeneratedBundle(defaultMonitoringAddress)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, generatedDockerComposeYAML)
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err = yaml.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		services, _ := stringMap(document["services"])
		grafana, _ := stringMap(services["grafana"])
		environment, _ := stringMap(grafana["environment"])
		// Avoid optional network downloads and shared ports only in this fixture.
		environment["GF_PLUGINS_PREINSTALL_DISABLED"] = "true"
		grafana["ports"] = []any{map[string]any{"target": grafanaPort, "published": "0", "host_ip": defaultMonitoringAddress}}
		contents, err = yaml.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err = writeFileContents(directory, generatedDockerComposeYAML, contents); err != nil {
			t.Fatal(err)
		}
		return directory, password
	}
	directory, oldPassword := stage()
	if err := mon.installStagedDirectory(directory); err != nil {
		t.Fatal(err)
	}
	candidatePassword, err := generateGrafanaPassword()
	if err != nil {
		t.Fatal(err)
	}
	secrets := []string{oldPassword, candidatePassword}
	client := &http.Client{Timeout: 2 * time.Second}
	endpoint := ""
	start := func() string {
		t.Helper()
		compose("up", "-d", "--no-deps", "--pull", "never", "grafana")
		id := strings.TrimSpace(string(compose("ps", "--all", "-q", "grafana")))
		port := f.mustDocker("port", id, "3000/tcp")
		endpoint = "http://" + strings.TrimSpace(string(port))
		deadline := time.Now().Add(60 * time.Second)
		for {
			response, err := client.Get(endpoint + "/api/health")
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return id
				}
			}
			if time.Now().After(deadline) {
				logs := string(f.mustDocker("logs", "--tail", "80", id))
				for _, password := range secrets {
					logs = strings.ReplaceAll(logs, password, "<redacted>")
				}
				t.Fatalf("Grafana did not become healthy: %s", logs)
			}
			time.Sleep(time.Second)
		}
	}
	authenticate := func(password string, want int) {
		t.Helper()
		request, err := http.NewRequest(http.MethodGet, endpoint+"/api/org", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.SetBasicAuth(grafanaAdminUser, password)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != want {
			t.Fatalf("authentication returned HTTP %d, want %d", response.StatusCode, want)
		}
	}
	rejectInit := func() {
		t.Helper()
		before := monitoringDirectorySnapshot(t, cfgDirectory)
		output.Reset()
		assertMonitoringRotationRejection(t, initMonitoringCmd.RunE(command, nil), "existing")
		if after := monitoringDirectorySnapshot(t, cfgDirectory); !reflect.DeepEqual(before, after) {
			t.Fatal("rejected initialization changed the active password/directory")
		}
		if strings.Contains(output.String(), "administrator password:") || strings.Contains(output.String(), OperationCompleted) {
			t.Fatal("rejected initialization reported success or a password")
		}
	}

	originalID := start()
	authenticate(oldPassword, http.StatusOK)
	rejectInit()
	compose("stop", "grafana")
	if strings.TrimSpace(string(compose("ps", "-q"))) != "" {
		t.Fatal("running-only query unexpectedly found the stopped Grafana")
	}
	rejectInit()
	t.Log("running and stopped Grafana both reject CLI initialization without changing the active directory/password")

	// Demonstrate why merely replacing the secret cannot rotate a stopped DB.
	if err = os.WriteFile(filepath.Join(mon.monitoringDir, grafanaPasswordFile), []byte(candidatePassword), 0600); err != nil {
		t.Fatal(err)
	}
	if start() != originalID {
		t.Fatal("fixture did not reproduce stopped-container reuse")
	}
	authenticate(oldPassword, http.StatusOK)
	authenticate(candidatePassword, http.StatusUnauthorized)
	t.Log("same container reused after secret replacement: old password HTTP 200; replacement HTTP 401")
	compose("stop", "grafana")
	if err = os.WriteFile(filepath.Join(mon.monitoringDir, grafanaPasswordFile), []byte(oldPassword), 0600); err != nil {
		t.Fatal(err)
	}
	rejectInit()

	// Missing models must still discover exited containers in this exact project.
	hidden := composeFile + ".saved"
	if err = os.Rename(composeFile, hidden); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(mon.monitoringDir, dockerComposeYAML)
	if err = os.Rename(legacy, legacy+".saved"); err != nil {
		t.Fatal(err)
	}
	rejectInit()
	if err = os.Rename(hidden, composeFile); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(legacy+".saved", legacy); err != nil {
		t.Fatal(err)
	}

	if err = stopMonitoringCmd.RunE(command, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(compose("ps", "--all", "-q"))) != "" {
		t.Fatal("supported stop left monitoring containers")
	}
	if err = mon.ensureMonitoringIsStopped(); err != nil {
		t.Fatal(err)
	}
	directory, newPassword := stage()
	secrets = append(secrets, newPassword)
	if newPassword == oldPassword {
		t.Fatal("reinitialization did not generate a different password")
	}
	if err = mon.installStagedDirectory(directory); err != nil {
		t.Fatal(err)
	}
	if start() == originalID {
		t.Fatal("supported removal/reinitialization reused the old container")
	}
	authenticate(newPassword, http.StatusOK)
	authenticate(oldPassword, http.StatusUnauthorized)
	t.Log("after cohctl stop monitoring and local staged reinitialization: new password HTTP 200; old password HTTP 401")
}
