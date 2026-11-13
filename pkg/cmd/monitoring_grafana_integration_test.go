/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// These opt-in tests cover daemon-local Linux files and a real Compose handoff.
// Run with COHCTL_TEST_MONITORING_DOCKER=1; pre-pull busybox:1.37.0 and the tracked
// Grafana image. COHCTL_TEST_SECRET_PROBE_IMAGE can select a cached BusyBox image.
type grafanaDockerFixture struct {
	t            *testing.T
	volume       string
	source       string
	user         string
	securityArgs []string
	entrypoint   []string
	image        string
	probeImage   string
	password     string
	composeFile  string
}

func newGrafanaDockerFixture(t *testing.T) *grafanaDockerFixture {
	t.Helper()
	if os.Getenv("COHCTL_TEST_MONITORING_DOCKER") != "1" {
		t.Skip("set COHCTL_TEST_MONITORING_DOCKER=1 to test Linux secret delivery with Docker")
	}
	f := &grafanaDockerFixture{t: t, volume: "cohctl-monitoring-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)}
	f.probeImage = os.Getenv("COHCTL_TEST_SECRET_PROBE_IMAGE")
	if f.probeImage == "" {
		f.probeImage = "busybox:1.37.0"
	}
	f.mustDocker("image", "inspect", f.probeImage)

	generated, err := generateSecureCompose(trackedCompose(t))
	if err != nil {
		t.Fatal(err)
	}
	composeFile := filepath.Join(t.TempDir(), generatedDockerComposeYAML)
	f.composeFile = composeFile
	if err = os.WriteFile(composeFile, generated, 0600); err != nil {
		t.Fatal(err)
	}
	// Resolve and validate the generated model with Compose itself.
	resolved := f.mustDocker("compose", "-f", composeFile, "config", "--format", "json")
	var model struct {
		Services map[string]struct {
			User        string
			Image       string
			Entrypoint  []string
			Environment map[string]string
			Secrets     []any
			CapAdd      []string `json:"cap_add"`
			CapDrop     []string `json:"cap_drop"`
			SecurityOpt []string `json:"security_opt"`
		} `json:"services"`
	}
	if err = json.Unmarshal(resolved, &model); err != nil {
		t.Fatal(err)
	}
	grafana := model.Services["grafana"]
	if len(grafana.Secrets) != 1 || len(model.Services["prometheus"].Secrets) != 0 {
		t.Fatal("resolved Compose model does not isolate the Grafana secret")
	}
	if _, found := grafana.Environment["GF_SECURITY_ADMIN_PASSWORD"]; found {
		t.Fatal("resolved Compose model contains a direct password environment setting")
	}
	if grafana.Environment["GF_SECURITY_ADMIN_PASSWORD__FILE"] != grafanaAdminFileTarget {
		t.Fatal("resolved Compose model has an unexpected password provider")
	}
	f.user, f.entrypoint, f.image = grafana.User, grafana.Entrypoint, grafana.Image
	for _, cap := range grafana.CapDrop {
		f.securityArgs = append(f.securityArgs, "--cap-drop", cap)
	}
	for _, cap := range grafana.CapAdd {
		f.securityArgs = append(f.securityArgs, "--cap-add", cap)
	}
	for _, opt := range grafana.SecurityOpt {
		f.securityArgs = append(f.securityArgs, "--security-opt", opt)
	}
	if len(f.entrypoint) != 5 || f.entrypoint[4] != "/run.sh" {
		t.Fatal("unexpected resolved Grafana entrypoint")
	}
	// `compose config` re-escapes dollars for round-tripping. Direct `docker
	// run` takes literal arguments; the Compose-run probe below checks the
	// actual Compose-to-container handoff separately.
	f.entrypoint[2] = strings.ReplaceAll(f.entrypoint[2], "$$", "$")
	f.password, err = generateGrafanaPassword()
	if err != nil {
		t.Fatal(err)
	}

	f.mustDocker("volume", "create", f.volume)
	t.Cleanup(func() {
		if output, cleanupErr := f.docker(nil, "volume", "rm", f.volume); cleanupErr != nil {
			t.Errorf("remove test volume: %v, %s", cleanupErr, output)
		}
	})
	f.source = strings.TrimSpace(string(f.mustDocker("volume", "inspect", "-f", "{{.Mountpoint}}", f.volume)))
	uid := "1000"
	// The password travels over stdin, never as a Docker argument or configured
	// environment value. All credential fixtures are 0600 on the daemon's disk.
	script := `set -eu
umask 077
cat > /fixture/password
cp /fixture/password /fixture/unreadable
cp /fixture/password /fixture/expected
chown 472:0 /fixture/expected
cp /etc/passwd /fixture/passwd
printf 'grafana:x:472:0:grafana:/tmp:/bin/sh\n' >> /fixture/passwd
sed 's/grafana:x:472:0:/grafana:x:473:0:/' /fixture/passwd > /fixture/wrong-passwd
chmod 644 /fixture/passwd /fixture/wrong-passwd
: > /fixture/empty
printf invalid > /fixture/invalid
mkdir /fixture/directory
chown "$1:0" /fixture/password /fixture/empty /fixture/invalid
chown 65534:0 /fixture/unreadable
cat > /fixture/run.sh <<'SH'
#!/bin/sh
set -eu
echo invoked
test "$GF_SECURITY_ADMIN_PASSWORD" = "$(cat /expected-password)"
test -z "${GF_SECURITY_ADMIN_PASSWORD__FILE+x}"
grep -q '^CapEff:[[:space:]]*0000000000000000$' /proc/self/status
grep -q '^NoNewPrivs:[[:space:]]*1$' /proc/self/status
printf 'delivered uid=%s gid=%s\n' "$(id -u)" "$(id -g)"
SH
chmod 755 /fixture/run.sh
stat -f -c '%T' /fixture
stat -c '%u:%g:%a' /fixture/password
`
	output, err := f.docker(strings.NewReader(f.password), "run", "--rm", "-i", "--mount", "type=volume,source="+f.volume+",target=/fixture", f.probeImage, "sh", "-ec", script, "fixture", uid)
	if err != nil {
		t.Fatalf("create Linux fixtures: %v, %s", err, output)
	}
	if strings.Contains(string(output), "virtiofs") || strings.Contains(string(output), "9p") || !strings.Contains(string(output), uid+":0:600") {
		t.Fatalf("expected protected daemon-local Linux file: %s", output)
	}
	t.Logf("Linux fixture filesystem and owner:mode: %s", strings.TrimSpace(string(output)))
	return f
}

func (f *grafanaDockerFixture) docker(input *strings.Reader, args ...string) ([]byte, error) {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "docker", args...)
	if input != nil {
		command.Stdin = input
	}
	command.Env = withEnvironment([]string{monitoringBindAddressEnv + "=" + defaultMonitoringAddress})
	return command.CombinedOutput()
}

func (f *grafanaDockerFixture) mustDocker(args ...string) []byte {
	f.t.Helper()
	output, err := f.docker(nil, args...)
	if err != nil {
		f.t.Fatalf("docker %s: %v, %s", strings.Join(args, " "), err, output)
	}
	return output
}

func (f *grafanaDockerFixture) secretMount(name string) string {
	return "type=bind,source=" + f.source + "/" + name + ",target=" + grafanaAdminFileTarget + ",readonly"
}

// Exercise the actual Compose handoff as well as the daemon-local bind tests.
// BusyBox supplies the probe executables; fixture mounts supply its account,
// expected value and /run.sh. The generated bootstrap identity, restrictions,
// password provider, secret grant and entrypoint are used unchanged.
func (f *grafanaDockerFixture) checkComposeEntrypoint() {
	f.t.Helper()
	directory := filepath.Dir(f.composeFile)
	if err := writeGrafanaPassword(directory, f.password); err != nil {
		f.t.Fatal(err)
	}
	for _, name := range []string{"grafana.ini", "datasources.yaml", "dashboards.yaml", "prometheus.yaml"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "monitoring", name))
		if err != nil {
			f.t.Fatal(err)
		}
		// #nosec G306 G703 -- fixed fixture names in a private test directory; non-secret configuration is readable by Grafana.
		if err = os.WriteFile(filepath.Join(directory, name), contents, 0644); err != nil {
			f.t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "dashboards"), 0755); err != nil {
		f.t.Fatal(err)
	}
	override, err := yaml.Marshal(map[string]any{"services": map[string]any{
		"grafana": map[string]any{
			"image":   f.probeImage,
			"volumes": []string{f.source + "/run.sh:/run.sh:ro", f.source + "/passwd:/etc/passwd:ro", f.source + "/expected:/expected-password:ro"},
		},
		"prometheus": map[string]any{"image": f.probeImage},
	}})
	if err != nil {
		f.t.Fatal(err)
	}
	overrideFile := filepath.Join(directory, "probe.yaml")
	if err = os.WriteFile(overrideFile, override, 0600); err != nil {
		f.t.Fatal(err)
	}
	args := []string{"compose", "-p", f.volume, "-f", f.composeFile, "-f", overrideFile}
	f.t.Cleanup(func() {
		if output, cleanupErr := f.docker(nil, append(args, "down", "--volumes")...); cleanupErr != nil {
			f.t.Errorf("remove Compose probe: %v, %s", cleanupErr, output)
		}
	})
	name := f.volume + "-compose-probe"
	f.t.Cleanup(func() { f.mustDocker("rm", "-f", name) })
	output, runErr := f.docker(nil, append(args, "run", "--name", name, "--no-deps", "--pull", "never", "grafana")...)
	if runErr != nil {
		entrypoint := f.mustDocker("inspect", "-f", "{{json .Config.Entrypoint}}", name)
		metadata := f.mustDocker("run", "--rm", "--mount", "type=bind,source="+filepath.Join(directory, grafanaPasswordFile)+",target="+grafanaAdminFileTarget+",readonly", f.probeImage, "stat", "-c", "%u:%g:%a", grafanaAdminFileTarget)
		f.t.Fatalf("Compose probe failed: %v, %s; entrypoint: %s; secret owner:mode: %s", runErr, output, entrypoint, metadata)
	}
	if !strings.Contains(string(output), "delivered uid=472 gid=0") {
		f.t.Fatalf("Compose entrypoint did not deliver the credential: %s", output)
	}
	if bytes.Contains(output, []byte(f.password)) {
		f.t.Fatal("Compose leaked the credential")
	}
	configured := f.mustDocker("inspect", "-f", "{{json .Config}}", name)
	if bytes.Contains(configured, []byte(f.password)) {
		f.t.Fatal("Compose placed the password in Docker configuration")
	}
	f.mustDocker(append(args, "run", "--rm", "--no-deps", "--pull", "never", "--entrypoint", "/bin/sh", "prometheus", "-ec", "test ! -e /run/secrets/grafana_admin_password")...)
}

func TestGrafanaSecretDeliveryLinux(t *testing.T) {
	f := newGrafanaDockerFixture(t)
	f.checkComposeEntrypoint()
	// Reproduce the original native-Linux mismatch before exercising the fix.
	wrongUser := "472:0"
	output, err := f.docker(nil, "run", "--rm", "--user", wrongUser, "--mount", f.secretMount("password"), f.probeImage, "cat", grafanaAdminFileTarget)
	if err == nil || !strings.Contains(string(output), "Permission denied") {
		t.Fatalf("expected UID mismatch to deny access: %v, %s", err, output)
	}
	for _, name := range []string{"password", "unreadable", "missing", "empty", "invalid", "directory"} {
		t.Run(name, func(t *testing.T) {
			args := []string{"run", "--rm", "--user", f.user, "--mount", "type=bind,source=" + f.source + "/run.sh,target=/run.sh,readonly", "--mount", "type=bind,source=" + f.source + "/passwd,target=/etc/passwd,readonly", "--mount", "type=bind,source=" + f.source + "/expected,target=/expected-password,readonly"}
			args = append(args, f.securityArgs...)
			if name != "missing" {
				args = append(args, "--mount", f.secretMount(name))
			}
			if name == "unreadable" && f.user == "0:0" {
				// Model a mapping that denies root access: remove its DAC
				// override capability, so the 0600 foreign-owned file cannot
				// be read. The entrypoint must stop before dropping identity.
				for i, value := range args {
					if value == "DAC_OVERRIDE" {
						args = append(args[:i-1], args[i+1:]...)
						break
					}
				}
			}
			args = append(args, "--env", "GF_SECURITY_ADMIN_PASSWORD__FILE="+grafanaAdminFileTarget, "--entrypoint", f.entrypoint[0], f.probeImage)
			args = append(args, f.entrypoint[1:]...)
			output, err := f.docker(nil, args...)
			if name == "password" {
				expected := "invoked\ndelivered uid=472 gid=0"
				if err != nil || strings.TrimSpace(string(output)) != expected {
					t.Fatalf("protected credential delivery failed: %v, %s", err, output)
				}
			} else if err == nil || strings.Contains(string(output), "invoked") || !strings.Contains(string(output), "administrator secret") {
				t.Fatalf("credential delivery did not fail closed: %v, %s", err, output)
			}
			if bytes.Contains(output, []byte(f.password)) {
				t.Fatal("credential leaked to container output")
			}
		})
	}
	for _, name := range []string{"wrong identity", "identity change denied"} {
		t.Run(name, func(t *testing.T) {
			passwd := "passwd"
			if name == "wrong identity" {
				passwd = "wrong-passwd"
			}
			args := []string{"run", "--rm", "--user", f.user, "--mount", f.secretMount("password"), "--mount", "type=bind,source=" + f.source + "/" + passwd + ",target=/etc/passwd,readonly", "--mount", "type=bind,source=" + f.source + "/run.sh,target=/run.sh,readonly"}
			for i := 0; i < len(f.securityArgs); i += 2 {
				if name == "identity change denied" && f.securityArgs[i+1] == "SETUID" {
					continue
				}
				args = append(args, f.securityArgs[i:i+2]...)
			}
			args = append(args, "--env", "GF_SECURITY_ADMIN_PASSWORD__FILE="+grafanaAdminFileTarget, "--entrypoint", f.entrypoint[0], f.probeImage)
			args = append(args, f.entrypoint[1:]...)
			output, err := f.docker(nil, args...)
			if err == nil || strings.Contains(string(output), "invoked") || bytes.Contains(output, []byte(f.password)) {
				t.Fatalf("identity failure did not stop startup safely: %v, %s", err, output)
			}
		})
	}
	// Read-only delivery must leave ownership and permissions untouched.
	metadata := f.mustDocker("run", "--rm", "--mount", "type=volume,source="+f.volume+",target=/fixture,readonly", f.probeImage, "stat", "-c", "%u:%g:%a", "/fixture/password")
	if strings.TrimSpace(string(metadata)) != "1000:0:600" {
		t.Fatalf("secret protection changed: %s", metadata)
	}
}

func TestGrafanaGeneratedPasswordAuthentication(t *testing.T) {
	f := newGrafanaDockerFixture(t)
	// Do not silently substitute a different Grafana version when pulls fail.
	f.mustDocker("image", "inspect", f.image)
	for _, failure := range []string{"missing", "unreadable"} {
		t.Run(failure, func(t *testing.T) {
			args := []string{"run", "--rm", "--user", f.user}
			for i := 0; i < len(f.securityArgs); i += 2 {
				if failure == "unreadable" && f.securityArgs[i+1] == "DAC_OVERRIDE" {
					continue
				}
				args = append(args, f.securityArgs[i:i+2]...)
			}
			if failure == "unreadable" {
				args = append(args, "--mount", f.secretMount("unreadable"))
			}
			args = append(args, "--env", "GF_SECURITY_ADMIN_PASSWORD__FILE="+grafanaAdminFileTarget, "--entrypoint", f.entrypoint[0], f.image)
			args = append(args, f.entrypoint[1:]...)
			output, err := f.docker(nil, args...)
			if err == nil || !strings.Contains(string(output), "administrator secret") || strings.Contains(string(output), "Starting Grafana") || bytes.Contains(output, []byte(f.password)) {
				t.Fatalf("tracked image did not reject %s delivery safely: %v, %s", failure, err, output)
			}
		})
	}
	name := f.volume + "-grafana"
	t.Cleanup(func() {
		if output, err := f.docker(nil, "rm", "-f", name); err != nil {
			t.Errorf("remove test Grafana: %v, %s", err, output)
		}
	})
	args := append([]string{"run", "-d"}, f.securityArgs...)
	// Authentication and signal checks must not depend on optional plugin
	// downloads. Keep this setting confined to the test container.
	args = append(args, "--env", "GF_PLUGINS_PREINSTALL_DISABLED=true", "--name", name, "--user", f.user, "--publish", "127.0.0.1::3000", "--mount", f.secretMount("password"), "--env", "GF_SECURITY_ADMIN_USER=admin", "--env", "GF_SECURITY_ADMIN_PASSWORD__FILE="+grafanaAdminFileTarget, "--entrypoint", f.entrypoint[0], f.image)
	args = append(args, f.entrypoint[1:]...)
	f.mustDocker(args...)
	configured := f.mustDocker("inspect", "-f", "{{json .Config}}", name)
	if bytes.Contains(configured, []byte(f.password)) {
		t.Fatal("password leaked into Docker-configured environment or arguments")
	}
	port, err := f.docker(nil, "port", name, "3000/tcp")
	if err != nil {
		logs := strings.ReplaceAll(string(f.mustDocker("logs", "--tail", "80", name)), f.password, "<redacted>")
		t.Fatalf("Grafana did not publish its port: %v, %s; logs: %s", err, port, logs)
	}
	endpoint := "http://" + strings.TrimSpace(string(port))
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(60 * time.Second)
	for {
		response, err := client.Get(endpoint + "/api/health")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			logs := strings.ReplaceAll(string(f.mustDocker("logs", "--tail", "80", name)), f.password, "<redacted>")
			t.Fatalf("Grafana did not become healthy: %s", logs)
		}
		time.Sleep(time.Second)
	}
	for _, test := range []struct {
		password string
		status   int
	}{{f.password, http.StatusOK}, {"admin", http.StatusUnauthorized}} {
		request, err := http.NewRequest(http.MethodGet, endpoint+"/api/org", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.SetBasicAuth(grafanaAdminUser, test.password)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != test.status {
			t.Fatalf("authentication returned %d, expected %d", response.StatusCode, test.status)
		}
	}
	t.Log("generated password: HTTP 200; admin/admin: HTTP 401")
	// Docker exec otherwise defaults to the configured bootstrap user (root).
	// Inspect the actual server, PID 1, from an unprivileged exec process.
	status := string(f.mustDocker("exec", "--user", "472:0", name, "cat", "/proc/1/status"))
	for _, expected := range []string{"Uid:\t472\t472\t472\t472", "Gid:\t0\t0\t0\t0", "CapEff:\t0000000000000000", "NoNewPrivs:\t1"} {
		if !strings.Contains(status, expected) {
			t.Fatalf("server identity/security mismatch: missing %s", expected)
		}
	}
	process := strings.TrimSpace(string(f.mustDocker("exec", "--user", "472:0", name, "cat", "/proc/1/comm")))
	if process != "grafana" {
		t.Fatalf("PID 1 is %q, expected the Grafana server", process)
	}
	t.Log("PID 1 is Grafana at 472:0, CapEff=0, NoNewPrivs=1")
	f.mustDocker("stop", "--time", "10", name)
	if code := strings.TrimSpace(string(f.mustDocker("inspect", "-f", "{{.State.ExitCode}}", name))); code != "0" {
		logs := strings.ReplaceAll(string(f.mustDocker("logs", "--tail", "80", name)), f.password, "<redacted>")
		t.Fatalf("Grafana did not stop cleanly: exit %s; logs: %s", code, logs)
	}
	t.Log("SIGTERM shutdown: exit 0 within the 10-second stop timeout")
}
