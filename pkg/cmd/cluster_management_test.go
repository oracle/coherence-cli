/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const managementTestIP = "127.0.0.2"
const managementFixtureName = "managed-fixture"
const managementTestDNS = "server.example"

const (
	managementTestLegacyJavaOptions = "_JAVA_OPTIONS"
	managementTestKill              = "kill"

	managementTestJDKJavaOptions = "JDK_JAVA_OPTIONS"

	managementTestConnectionName    = "managed"
	managementTestNewConnectionName = "new-managed"
	managementTestScale             = "scale"
	managementTestCreate            = "create"
	managementTestRestart           = "restart"
	managementTestJavaToolOptions   = "JAVA_TOOL_OPTIONS"
	managementTestWindows           = "windows"
	managementTestMissingProfile    = "missing-startup-profile"
	managementTestReadiness         = "readiness"
	managementTestJSON              = "json"
)

const (
	managementTestHTTPAll      = "-Dcoherence.management.http=all"
	managementTestJarLaunch    = "-jar app.jar"
	managementTestHiddenFile   = "@hidden"
	managementTestCustomOption = "-Dcustom=value"
)

func managementTestCommand(t *testing.T, flags ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.Flags().Int32(jmxPortArg, 0, "")
	cmd.Flags().String(jmxHostArg, "", "")
	managementFlags(cmd)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("n\n"))
	if err := cmd.ParseFlags(flags); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func managedTestState(t *testing.T) {
	t.Helper()
	oldConfig, oldFile, settings := Config, viper.ConfigFileUsed(), viper.AllSettings()
	oldProfile, oldDelay, oldClass := profileValueParam, startupDelayParam, serverStartClassParam
	oldMetrics, oldHealth, oldReplicas, oldLevel := metricsStartPortParam, healthStartPortParam, replicaCountParam, logLevelParam
	oldDynamic, oldConfirm, oldBackup, oldFirst := dynamicHTTPParam, automaticallyConfirm, backupLogFilesParam, profileFirstParam
	oldAwait := awaitManagedStartup
	oldLogs, oldHeap, oldWrite, oldTimeout := logsDirectory, heapMemoryParam, writeManagedConfig, managedReadinessTimeout
	viper.Reset()
	Config = CoherenceCLIConfig{}
	profileValueParam, serverStartClassParam, startupDelayParam = "", "", "0ms"
	metricsStartPortParam, healthStartPortParam, replicaCountParam, logLevelParam = 0, 0, 1, 5
	dynamicHTTPParam, automaticallyConfirm, backupLogFilesParam, profileFirstParam = false, false, false, false
	heapMemoryParam = "128m"
	logsDirectory = t.TempDir()
	for _, name := range []string{managementTestJavaToolOptions, managementTestLegacyJavaOptions, managementTestJDKJavaOptions} {
		t.Setenv(name, "")
	}
	t.Cleanup(func() {
		Config = oldConfig
		viper.Reset()
		_ = viper.MergeConfigMap(settings)
		viper.SetConfigFile(oldFile)
		profileValueParam, startupDelayParam, serverStartClassParam = oldProfile, oldDelay, oldClass
		metricsStartPortParam, healthStartPortParam, replicaCountParam, logLevelParam = oldMetrics, oldHealth, oldReplicas, oldLevel
		dynamicHTTPParam, automaticallyConfirm, backupLogFilesParam, profileFirstParam = oldDynamic, oldConfirm, oldBackup, oldFirst
		logsDirectory, heapMemoryParam, writeManagedConfig, managedReadinessTimeout = oldLogs, oldHeap, oldWrite, oldTimeout
		awaitManagedStartup = oldAwait
	})
}

func TestManagementResolution(t *testing.T) {
	stored := ClusterConnection{ManuallyCreated: true, Management: &ClusterManagement{RESTAddress: managementTestIP, JMXPort: 9999, JMXAddress: localHost, JMXHost: localHost}}
	for _, tt := range []struct {
		name       string
		connection ClusterConnection
		flags      []string
		want       ClusterManagement
		invalid    bool
	}{
		{"legacy", ClusterConnection{ManuallyCreated: true}, nil, ClusterManagement{RESTAddress: localHost}, false},
		{"reuse", stored, nil, *stored.Management, false},
		{"disable", stored, []string{"--jmx-port=0"}, ClusterManagement{RESTAddress: managementTestIP}, false},
		{loopbackHostname, ClusterConnection{}, []string{"--management-address=localhost", "--jmx-port=9999", "--jmx-address=localhost", "--jmx-host=localhost"}, ClusterManagement{localHost, 9999, localHost, localHost}, false},
		{"loopback range", ClusterConnection{}, []string{"--jmx-port=9999", "--jmx-address=127.20.1.2"}, ClusterManagement{localHost, 9999, "127.20.1.2", "127.20.1.2"}, false},
		{"negative", stored, []string{"--jmx-port=-1"}, ClusterManagement{}, true},
		{"privileged", stored, []string{"--jmx-port=1023"}, ClusterManagement{}, true},
		{"overflow", stored, []string{"--jmx-port=65536"}, ClusterManagement{}, true},
		{"orphan bind", ClusterConnection{}, []string{"--jmx-address=127.0.0.1"}, ClusterManagement{}, true},
		{"orphan host", ClusterConnection{}, []string{"--jmx-host=localhost"}, ClusterManagement{}, true},
		{"wildcard host required", stored, []string{"--jmx-address=0.0.0.0"}, ClusterManagement{}, true},
		{"host alone", stored, []string{"--jmx-host=server.example"}, ClusterManagement{}, true},
		{"DNS bind", stored, []string{"--management-address=server.example"}, ClusterManagement{}, true},
		{"IPv6", stored, []string{"--management-address=::1"}, ClusterManagement{}, true},
		{"multicast", stored, []string{"--management-address=224.0.0.1"}, ClusterManagement{}, true},
		{"broadcast", stored, []string{"--management-address=255.255.255.255"}, ClusterManagement{}, true},
		{"wildcard", stored, []string{"--jmx-address=0.0.0.0", "--jmx-host=server.example"}, ClusterManagement{managementTestIP, 9999, wildcardIPv4, managementTestDNS}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveManagement(managementTestCommand(t, tt.flags...), tt.connection)
			if (err != nil) != tt.invalid {
				t.Fatalf("got %v, error %v", got, err)
			}
			if !tt.invalid && got != tt.want {
				t.Fatalf("got %+v want %+v", got, tt.want)
			}
		})
	}
	for _, host := range []string{"", wildcardIPv4, net.IPv6loopback.String(), "https://host", "host:9999", "-host", "bad host", "bad..host", "255.255.255.255", "224.1.1.1"} {
		if validJMXHost(host) {
			t.Errorf("accepted host %q", host)
		}
	}
}

func TestManagementAcknowledgement(t *testing.T) {
	managedTestState(t)
	m := ClusterManagement{wildcardIPv4, 9999, wildcardIPv4, managementTestDNS}
	for _, allow := range []bool{false, true} {
		cmd := managementTestCommand(t, fmt.Sprintf("--allow-insecure-remote-management=%t", allow))
		automaticallyConfirm = true
		err := authorizeManagement(cmd, m, true, true, 30000)
		if (err == nil) != allow {
			t.Fatalf("ack %t error %v", allow, err)
		}
		if !strings.Contains(cmd.OutOrStdout().(*bytes.Buffer).String(), "without authentication or encryption") {
			t.Fatal("warning missing")
		}
	}
	if err := authorizeManagement(managementTestCommand(t), m, false, false, 30000); err != nil {
		t.Fatal("scale without new listeners should not require consent", err)
	}
	if strings.Contains(managementURL(m, 30000), wildcardIPv4) {
		t.Fatal("wildcard destination")
	}
}

func TestManagementMemberRoles(t *testing.T) {
	managedTestState(t)
	connection := ClusterConnection{ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion}
	m := ClusterManagement{localHost, 9999, localHost, localHost}
	for _, existing := range []int32{0, 3} {
		for _, dynamic := range []bool{false, true} {
			for count := int32(1); count <= 3; count++ {
				jmxCount := 0
				for i := int32(0); i < count; i++ {
					port := memberHTTPPort(connection, i, existing, dynamic)
					want := int32(-1)
					if existing == 0 && i == 0 {
						want = 30000
					} else if dynamic && i > 0 {
						want = 0
					}
					if port != want {
						t.Fatalf("existing=%d dynamic=%t i=%d: got %d want %d", existing, dynamic, i, port, want)
					}
					args := getCacheServerArgs(connection, "storage", port, m, existing == 0 && i == 0)
					if !strings.Contains(strings.Join(args, " "), "-Dcoherence.management.http.address=127.0.0.1") {
						t.Fatal(args)
					}
					seen := map[string]bool{}
					for _, arg := range args {
						if strings.HasPrefix(arg, "-D") {
							key := strings.SplitN(arg, "=", 2)[0]
							if seen[key] {
								t.Fatal("duplicate", key)
							}
							seen[key] = true
						}
						if strings.HasPrefix(arg, "-Dcom.sun.management.jmxremote.port=") {
							jmxCount++
						}
					}
				}
				if existing == 0 && jmxCount != 1 || existing > 0 && jmxCount != 0 {
					t.Fatalf("JMX owners: %d", jmxCount)
				}
			}
		}
	}
	args := getCacheServerArgs(connection, "storage", 30000, ClusterManagement{RESTAddress: localHost}, true)
	if strings.Contains(strings.Join(args, " "), "com.sun.management.jmxremote") {
		t.Fatal("disabled JMX emitted agent properties")
	}
}

func TestManagementOptionOwnership(t *testing.T) {
	managedTestState(t)
	legacy := ClusterConnection{ManuallyCreated: true, Arguments: "-Dcoherence.wka=192.0.2.1 -Djava.rmi.server.hostname=192.0.2.1 -Dcustom=value"}
	if _, err := validateManagedOptions(legacy); err != nil {
		t.Fatal(err)
	}
	base, _ := serverBaseOptions(legacy)
	if strings.Contains(strings.Join(base, " "), "java.rmi.server.hostname") || !strings.Contains(strings.Join(base, " "), managementTestCustomOption) {
		t.Fatal(base)
	}
	duplicate := legacy
	duplicate.Arguments += " -Djava.rmi.server.hostname=192.0.2.1"
	if _, err := validateManagedOptions(duplicate); err == nil {
		t.Fatal("migrated more than the single legacy-generated hostname")
	}
	clientArgs, err := getCommonArguments(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(clientArgs, " "), "java.rmi.server.hostname") {
		t.Fatal("client arguments modified")
	}
	for _, value := range []string{`-Dcustom=ManagementServer`, `-Dcustom="with spaces" -Xmx256m`, `-Dpath=C:\test\file`, `-Dcustom="a'b"`} {
		args, err := splitJVMOptions(value)
		if err != nil {
			t.Fatal(err)
		}
		c := testManagedConnection(t, ClusterConnection{Arguments: value, Management: &ClusterManagement{RESTAddress: localHost}}, ClusterManagement{RESTAddress: localHost})
		got, _ := splitJVMOptions(c.Arguments)
		if !reflect.DeepEqual(args, got) {
			t.Fatalf("round trip %q: %v != %v", value, args, got)
		}
	}
}

const (
	managementTestLegacyRecord   = "legacy"
	managementTestProfileOptions = "startup-profile"
	managementTestMainClass      = "MainClass"
)

func managementModuleOptionCases() []struct{ option, operand string } {
	return []struct{ option, operand string }{
		{jvmAddOpensOption, "java.base/java.lang=ALL-UNNAMED"},
		{jvmAddExportsOption, "java.base/sun.nio.ch=java.sql,ALL-UNNAMED"},
		{jvmAddReadsOption, "java.base=java.sql,ALL-UNNAMED"},
		{jvmAddModulesOption, "java.sql,ALL-SYSTEM"},
		{jvmLimitModulesOption, "java.base,java.sql"},
	}
}

func TestManagementOptionPolicy(t *testing.T) {
	blocked := []string{
		managementTestHTTPAll, "-Dtangosol.coherence.management.http.address=0.0.0.0",
		"-Dcom.sun.management.jmxremote.port=9999", "-Dcom.sun.management.config.file=other",
		"-Dcoherence.management.config=other", "-Djava.rmi.server.hostname=other",
		managementTestArgfile, "-XX:VMOptionsFile=options", "-XX:Flags=options", "-XX:+ManagementServer",
		"-Xmanagement", "-Xoptionsfile=options", managementTestAgentHook,
		managementTestJarLaunch, "-m app/main", "--module=app/main", "--source 17",
	}
	blocked = append(blocked, "-Dcom.sun.management.jmxremote", "-Dcom.sun.management.jmxremote.authenticate=true", "-Dcoherence.management.http=none", "-XX:-ManagementServer", "-XX:ManagementServer=true", "--module app/main", "--source=17", "MainClass")
	for _, value := range blocked {
		if _, err := validateServerOptions("policy", strings.Fields(value), true); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestManagementModuleOptionRejection(t *testing.T) {
	blocked := []string{managementTestHTTPAll, managementTestHiddenFile, managementTestJarLaunch}
	for _, tc := range managementModuleOptionCases() {
		t.Run(tc.option, func(t *testing.T) {
			for _, args := range [][]string{{tc.option}, {tc.option, ""}, {tc.option + "="}, {tc.option, jvmAddOpensOption},
				{tc.option, tc.operand, managementTestMainClass}, {tc.option + "=" + tc.operand, managementTestMainClass}} {
				if _, err := validateServerOptions("test arguments", args, true); err == nil {
					t.Errorf("accepted missing/empty operand or extra main class: %q", args)
				}
			}
			for _, value := range blocked {
				for _, args := range [][]string{
					append([]string{tc.option}, strings.Fields(value)...),
					append([]string{tc.option, tc.operand}, strings.Fields(value)...),
					append([]string{tc.option + "=" + tc.operand}, strings.Fields(value)...),
				} {
					if _, err := validateServerOptions("test arguments", args, true); err == nil {
						t.Errorf("accepted override in %q", args)
					}
				}
				// Attached text stays a single value, even when it names a forbidden option.
				if _, err := validateServerOptions("test arguments", []string{tc.option + "=" + value}, true); err != nil {
					t.Errorf("reinterpreted attached value %q: %v", value, err)
				}
			}
		})
	}
	for _, args := range [][]string{{managementTestMainClass}, {jvmLimitModulesOption, "java.base\x00"},
		{"--unknown-option", "java.base"}, {"--add-opens-extra", "java.base/java.lang=ALL-UNNAMED"}} {
		if _, err := validateServerOptions("test arguments", args, true); err == nil {
			t.Errorf("accepted structural violation: %q", args)
		}
	}
}

func TestManagementModuleOptionSemanticPassThrough(t *testing.T) {
	for _, args := range [][]string{
		{jvmAddOpensOption, managementTestMainClass}, {"--add-exports=MainClass"}, {jvmAddReadsOption, managementTestMainClass},
		{jvmAddOpensOption, "java.base/java.lang"}, {"--add-opens=java.base/=ALL-UNNAMED"},
		{jvmAddExportsOption, "/sun.nio.ch=ALL-UNNAMED"}, {"--add-exports=java.base/sun/nio=ALL-UNNAMED"},
		{jvmAddReadsOption, "java.base="}, {"--add-reads==ALL-UNNAMED"}, {"--add-reads=java.base/java.lang=ALL-UNNAMED"},
		{jvmAddOpensOption, "java.base/java.lang=,ALL-UNNAMED"}, {"--add-reads=java.base=java.sql,"},
		{"--add-modules=java.base,,java.sql"}, {jvmLimitModulesOption, "java.base/java.lang"},
		{jvmAddModulesOption, "java.base -Dcoherence.management.http=all"},
		{"--add-opens=java.base/java.lang=ALL-UNNAMED -jar app.jar"},
	} {
		if _, err := validateServerOptions("test arguments", args, true); err != nil {
			t.Errorf("rejected JVM semantic input %q: %v", args, err)
		}
	}
}

func TestManagementRuntimeChecks(t *testing.T) {
	for _, tt := range []struct {
		output       string
		major, patch int
	}{{`openjdk version "11.0.32.1" 2026-10-20`, 11, 32}, {`java version "1.8.0_504"`, 8, 0}, {`openjdk 21.0.13 2026-10-20`, 21, 13}} {
		major, patch, err := parseJavaVersion(tt.output)
		if err != nil || major != tt.major || patch != tt.patch {
			t.Fatal(major, patch, err)
		}
	}
	for _, tt := range []struct {
		version      string
		major, patch int
		invalid      bool
	}{{managementTestJava11Coherence, 11, 32, false}, {"22.09", 11, 32, true}, {"14.1.1.2206.1", 11, 14, true}, {"14.1.1.2206.1", 11, 15, false}, {defaultCoherenceVersion, 11, 32, true}, {"15.1.1-0-1", 17, 21, false}, {"25.03.2", 17, 21, false}, {managementTestUnknownVersion, 11, 32, false}} {
		if err := validateCoherenceJava(tt.version, tt.major, tt.patch); (err != nil) != tt.invalid {
			t.Fatalf("%+v: %v", tt, err)
		}
	}
	for _, tt := range []struct {
		name     string
		fixture  javaFixture
		contains string
	}{{"nonzero", javaFixture{Version: javaVersion17, Exit: 1}, "unable to complete"}, {"unrecognized", javaFixture{Version: managementTestUnknownVersion, RawOutput: true}, "unrecognized"}, {"success", javaFixture{Version: "11.0.32.1"}, ""}} {
		t.Run(tt.name, func(t *testing.T) {
			fakeJava(t, tt.fixture)
			err := checkManagedRuntime(managementTestJava11Coherence, 0)
			if tt.contains == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatal(err)
			}
		})
	}
	t.Run("missing", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if err := checkManagedRuntime(managementTestJava11Coherence, 0); err == nil || !strings.Contains(err.Error(), "unable to complete") {
			t.Fatal(err)
		}
	})
}

func TestManagementLifecyclePreflightAndCancellation(t *testing.T) {
	managedTestState(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/members") {
			fmt.Fprint(w, `{"items":[{"processName":"99999999"}]}`)
		} else {
			fmt.Fprint(w, `{"clusterName":"managed-fixture","clusterSize":1}`)
		}
	}))
	defer server.Close()
	connection := ClusterConnection{Name: managementFixtureName, ClusterName: managementFixtureName, ManuallyCreated: true, ConnectionType: httpType, ConnectionURL: server.URL, ManagementPort: managementListenerPort(t, server.Listener), ClusterVersion: defaultCoherenceVersion}
	Config.Clusters = []ClusterConnection{connection}
	fakeJava(t, javaFixture{Version: "11.0.32.1"})
	if err := runClusterOperation(managementTestCommand(t), managementFixtureName, "restart cluster"); err == nil {
		t.Fatal("invalid replacement accepted")
	}
	if calls != 0 {
		t.Fatal("restart accessed current cluster before runtime validation")
	}
	profileValueParam = managementTestMissingProfile
	t.Setenv("PATH", t.TempDir())
	t.Setenv(managementTestJavaToolOptions, "@invalid")
	if err := runClusterOperation(managementTestCommand(t), managementFixtureName, stopClusterCommand); err != nil {
		t.Fatal("stop did not bypass startup checks", err)
	}
	if calls == 0 {
		t.Fatal("stop did not inspect existing cluster")
	}
	if Config.Clusters[0].Management != nil || Config.Clusters[0].ConnectionURL != connection.ConnectionURL {
		t.Fatal("cancelled stop rewrote legacy config")
	}
	profileValueParam = ""
	t.Setenv(managementTestJavaToolOptions, "")
	fakeJava(t, javaFixture{Version: javaVersion17})
	if err := runClusterOperation(managementTestCommand(t), managementFixtureName, restartClusterCommand); err != nil {
		t.Fatal("cancelled valid restart changed running cluster", err)
	}
	if Config.Clusters[0].Management != nil || Config.Clusters[0].ConnectionURL != connection.ConnectionURL {
		t.Fatal("cancelled restart rewrote config")
	}
}

func TestManagementPartialLaunchCleanup(t *testing.T) {
	managedTestState(t)
	fakeJava(t, javaFixture{Sleep: true})
	connection := ClusterConnection{Name: "partial", ClusterName: "partial", ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: managementTestJava11Coherence}
	dir := filepath.Join(logsDirectory, connection.Name)
	if err := os.MkdirAll(filepath.Join(dir, "storage-1.log"), 0700); err != nil {
		t.Fatal(err)
	}
	unrelated := stopTestChild(t)
	processes, err := runTestStartup(t, managementTestCommand(t), connection, 2, 0, ClusterManagement{RESTAddress: localHost})
	if err == nil || len(processes) != 1 {
		t.Fatalf("processes=%d err=%v", len(processes), err)
	}
	select {
	case <-processes[0].done:
	default:
		t.Fatal("owned child was not cleaned up")
	}
	assertManagedChildAlive(t, unrelated)
}

func TestManagementReadinessPIDAndFailure(t *testing.T) {
	managedTestState(t)
	fakeJava(t, javaFixture{Sleep: true})
	p, err := launchManagedProcess(managementFixtureName, logTestMember, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cleanupManagedProcesses([]*managedProcess{p}); err != nil {
			t.Errorf("readiness fixture cleanup: %v", err)
		}
	}()
	goodPID := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/members") {
			pid := p.command.Process.Pid
			if !goodPID {
				pid++
			}
			fmt.Fprintf(w, `{"items":[{"processName":"%d"}]}`, pid)
		} else {
			fmt.Fprint(w, `{"clusterName":"managed-fixture","clusterSize":1}`)
		}
	}))
	defer server.Close()
	connection := ClusterConnection{ConnectionURL: server.URL, ClusterName: managementFixtureName}
	if err := waitManagedReady(managementTestCommand(t), connection, []*managedProcess{p}); err != nil {
		t.Fatal(err)
	}
	goodPID = false
	managedReadinessTimeout = 50 * time.Millisecond
	if err := waitManagedReady(managementTestCommand(t), connection, []*managedProcess{p}); err == nil {
		t.Fatal("wrong PID accepted")
	}
	connection.ClusterName = "other"
	if err := waitManagedReady(managementTestCommand(t), connection, []*managedProcess{p}); err == nil {
		t.Fatal("wrong cluster accepted")
	}
}

func TestManagementPreflightPortsAndScale(t *testing.T) {
	managedTestState(t)
	fakeJava(t, javaFixture{Version: javaVersion17})
	connection := ClusterConnection{ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion, Management: &ClusterManagement{RESTAddress: localHost}}
	if _, err := preflightManagedStartup(managementTestCommand(t, "--jmx-port=30000"), connection, 3, 0, false); err == nil {
		t.Fatal("fixed management port collision accepted")
	}
	metricsStartPortParam = 65535
	if _, err := preflightManagedStartup(managementTestCommand(t), connection, 2, 0, false); err == nil {
		t.Fatal("overflowing metrics range accepted")
	}
	metricsStartPortParam = 30000
	if _, err := preflightManagedStartup(managementTestCommand(t), connection, 1, 3, true); err == nil {
		t.Fatal("existing fixed listener collision accepted")
	}
	metricsStartPortParam = 0
	if _, err := preflightManagedStartup(managementTestCommand(t, "--management-address=127.0.0.2"), connection, 2, 3, true); err == nil {
		t.Fatal("endpoint-changing scale accepted")
	}
	if _, err := preflightManagedStartup(managementTestCommand(t, "--management-address=localhost"), connection, 2, 3, true); err != nil {
		t.Fatal("identical scale endpoint rejected", err)
	}
}

func TestManagementScaleIdenticalJMXBindPreservesHost(t *testing.T) {
	managedTestState(t)
	fakeJava(t, javaFixture{Version: javaVersion17})
	connection := ClusterConnection{ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion, Management: &ClusterManagement{RESTAddress: localHost, JMXPort: 9999, JMXAddress: "192.0.2.1", JMXHost: managementTestDNS}}
	if _, err := preflightManagedStartup(managementTestCommand(t, "--jmx-address=192.0.2.1"), connection, 1, 3, true); err != nil {
		t.Fatal("identical JMX bind should preserve stored advertised host", err)
	}
}

const (
	clientFixtureName     = "legacy-client"
	clientClusterOption   = "-Dcoherence.cluster=" + clientFixtureName
	clientDiscoveryOption = "-Dcoherence.wka=127.0.0.1"
	clientLiteralPath     = "-Dcoherence.cacheconfig=/tmp/o'brien-cache.xml"
	clientLiteralJSON     = `-Dcustom={"mode":"safe"}`
	clientQuotedOptions   = clientClusterOption + ` '-Dcoherence.cacheconfig=/tmp/o'"'"'brien cache.xml' '-Dcustom={"mode":"safe"}'`
	clientFixtureClass    = "fixture.ArgumentCapture"
)

func clientArgumentCases() []struct {
	name       string
	saved      string
	management *ClusterManagement
	want       []string
} {
	return []struct {
		name       string
		saved      string
		management *ClusterManagement
		want       []string
	}{
		{"legacy-apostrophe", clientClusterOption + " " + clientDiscoveryOption + " " + clientLiteralPath, nil,
			[]string{clientClusterOption, clientDiscoveryOption, clientLiteralPath}},
		{"legacy-json", clientClusterOption + " " + clientLiteralJSON, nil,
			[]string{clientClusterOption, clientLiteralJSON}},
		{"managed-quoted", clientQuotedOptions, &ClusterManagement{RESTAddress: localHost},
			[]string{clientClusterOption, "-Dcoherence.cacheconfig=/tmp/o'brien cache.xml", clientLiteralJSON}},
	}
}

func TestClientSavedArguments(t *testing.T) {
	for _, tc := range clientArgumentCases() {
		t.Run(tc.name, func(t *testing.T) {
			connection := ClusterConnection{Arguments: tc.saved, Management: tc.management}
			before := connection
			got, err := getCommonArguments(connection)
			if err != nil {
				t.Fatal(err)
			}
			want := append(append([]string{}, tc.want...), commonJVMOptions(connection)...)
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(connection, before) {
				t.Fatalf("saved arguments changed: want %q, got %q; connection %+v", want, got, connection)
			}
		})
	}
	t.Run("legacy-literal-separators", func(t *testing.T) {
		connection := ClusterConnection{Arguments: " -Dpath=C:\\cache\\file  -Dliteral=\"quoted\" "}
		got, err := getCommonArguments(connection)
		want := append(strings.Split(connection.Arguments, " "), commonJVMOptions(connection)...)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("legacy raw tokens: want %q, got %q, error %v", want, got, err)
		}
	})
}

func clientTestState(t *testing.T) {
	t.Helper()
	managedTestState(t)
	oldConnection, oldUsing, oldFormat := clusterConnection, UsingContext, OutputFormat
	oldKB, oldMB, oldGB := kbFormat, mbFormat, gbFormat
	oldExtend, oldGRPC, oldFile, oldStatement := extendClientParam, grpcClientParam, fileNameParam, statementParam
	clusterConnection, OutputFormat = "", "table"
	kbFormat, mbFormat, gbFormat = false, false, false
	extendClientParam, grpcClientParam, fileNameParam, statementParam = false, false, "", ""
	t.Cleanup(func() {
		clusterConnection, UsingContext, OutputFormat = oldConnection, oldUsing, oldFormat
		kbFormat, mbFormat, gbFormat = oldKB, oldMB, oldGB
		extendClientParam, grpcClientParam, fileNameParam, statementParam = oldExtend, oldGRPC, oldFile, oldStatement
	})
}

func clientLaunchCases() []struct {
	name    string
	args    []string
	command *cobra.Command
} {
	return []struct {
		name    string
		args    []string
		command *cobra.Command
	}{
		{"console", nil, startConsoleCmd},
		{"cohql", nil, startCohQLCmd},
		{"class", []string{clientFixtureClass, "query with spaces", `{"arg":"literal"}`}, startClassCmd},
	}
}

func TestClientMalformedArgumentsBeforeLaunch(t *testing.T) {
	for _, launch := range clientLaunchCases() {
		t.Run(launch.name, func(t *testing.T) {
			clientTestState(t)
			connection := ClusterConnection{Name: clientFixtureName, ManuallyCreated: true, ConnectionType: httpType,
				Arguments: clientClusterOption + " '-Dcustom=unterminated", Management: &ClusterManagement{RESTAddress: localHost}}
			Config.CurrentContext, Config.Clusters = connection.Name, []ClusterConnection{connection}
			// No Java executable is available: a launch attempt would return a
			// different error. The command must stop at argument decoding.
			t.Setenv("PATH", t.TempDir())
			var output bytes.Buffer
			cmd := managementTestCommand(t)
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			err := launch.command.RunE(cmd, launch.args)
			if err == nil || !strings.Contains(err.Error(), "saved client arguments: unterminated quote") || strings.Contains(output.String(), "Starting client") {
				t.Fatalf("malformed options reached launch: error %v, output %s", err, output.String())
			}
			if !reflect.DeepEqual(Config.Clusters, []ClusterConnection{connection}) {
				t.Fatal("client failure changed saved configuration")
			}
		})
	}
}

// These classes capture Java's actual input arguments without Coherence jars,
// discovery, cluster attachment or listeners. Every class exits immediately.
func compileClientArgumentFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("JDK compiler unavailable for real Java argv fixture")
	}
	if _, err := exec.LookPath(javaExec); err != nil {
		t.Skip("Java unavailable for real Java argv fixture")
	}
	dir := t.TempDir()
	sources := map[string]string{
		"fixture/ArgumentCapture.java": `package fixture;
import java.lang.management.ManagementFactory;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
public class ArgumentCapture {
  private static void emit(String kind, String value) {
    System.out.println(kind + Base64.getEncoder().encodeToString(value.getBytes(StandardCharsets.UTF_8)));
  }
  public static void main(String[] args) {
    for (String arg : ManagementFactory.getRuntimeMXBean().getInputArguments()) emit("JVM:", arg);
    for (String arg : args) emit("ARG:", arg);
    emit("PROP:", System.getProperty("coherence.cluster", ""));
    emit("PROP:", System.getProperty("coherence.cacheconfig", ""));
    emit("PROP:", System.getProperty("custom", ""));
  }
}`,
		"com/tangosol/net/CacheFactory.java": `package com.tangosol.net;
public class CacheFactory { public static void main(String[] args) { fixture.ArgumentCapture.main(args); } }`,
		"com/tangosol/coherence/dslquery/QueryPlus.java": `package com.tangosol.coherence.dslquery;
public class QueryPlus { public static void main(String[] args) { fixture.ArgumentCapture.main(args); } }`,
	}
	args := []string{"-d", dir}
	for name, source := range sources {
		filename := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, filename)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "javac", args...).CombinedOutput(); err != nil {
		t.Fatalf("compile argv fixture: %v: %s", err, output)
	}
	return dir
}

func decodedClientOutput(t *testing.T, output, prefix string) []string {
	t.Helper()
	var values []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			value, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, prefix))
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, string(value))
		}
	}
	return values
}

func TestClientJavaArgumentPreservation(t *testing.T) {
	clientTestState(t)
	classpath := compileClientArgumentFixture(t)
	for _, tc := range clientArgumentCases() {
		for _, launch := range clientLaunchCases() {
			t.Run(tc.name+"/"+launch.name, func(t *testing.T) {
				connection := ClusterConnection{Name: clientFixtureName, ManuallyCreated: true, ConnectionType: httpType,
					BaseClasspath: classpath, Arguments: tc.saved, Management: tc.management}
				Config.CurrentContext, Config.Clusters = connection.Name, []ClusterConnection{connection}
				var output bytes.Buffer
				cmd := managementTestCommand(t)
				cmd.SetOut(&output)
				cmd.SetErr(&output)
				cmd.SetIn(strings.NewReader(""))
				if err := launch.command.RunE(cmd, launch.args); err != nil {
					t.Fatalf("client launch: %v: %s", err, output.String())
				}
				got := decodedClientOutput(t, output.String(), "JVM:")
				if len(got) < len(tc.want) || !reflect.DeepEqual(got[:len(tc.want)], tc.want) {
					t.Fatalf("actual Java options: want prefix %q, got %q", tc.want, got)
				}
				props := []string{clientFixtureName, "", ""}
				for _, arg := range tc.want {
					for i, key := range []string{"coherence.cluster", "coherence.cacheconfig", "custom"} {
						if value, found := strings.CutPrefix(arg, "-D"+key+"="); found {
							props[i] = value
						}
					}
				}
				if got := decodedClientOutput(t, output.String(), "PROP:"); !reflect.DeepEqual(got, props) {
					t.Fatalf("Java properties: want %q, got %q", props, got)
				}
				var extra []string
				if launch.name == "class" {
					extra = launch.args[1:]
				}
				if got := decodedClientOutput(t, output.String(), "ARG:"); !reflect.DeepEqual(got, extra) {
					t.Fatalf("class arguments: want %q, got %q", extra, got)
				}
				if !reflect.DeepEqual(Config.Clusters, []ClusterConnection{connection}) {
					t.Fatal("client launch changed saved record")
				}
			})
		}
	}
}

func TestClientServerArgumentPersistence(t *testing.T) {
	for _, format := range []string{configType, ordinaryYML, constants.JSON} {
		t.Run(format, func(t *testing.T) {
			managedPersistenceState(t)
			filename := filepath.Join(t.TempDir(), "cohctl."+format)
			viper.SetConfigFile(filename)
			legacy := ClusterConnection{Name: clientFixtureName, ManuallyCreated: true,
				Arguments: clientClusterOption + " " + clientDiscoveryOption + " " + clientLiteralPath + " " + clientLiteralJSON + " -Djava.rmi.server.hostname=" + localHost}
			untouched := legacy
			untouched.Name = "unrelated-legacy"
			Config.Clusters = []ClusterConnection{legacy, untouched}
			viper.Set(clusterKey, Config.Clusters)
			wantServer := []string{clientClusterOption, clientDiscoveryOption, clientLiteralPath, clientLiteralJSON}
			base, err := serverBaseOptions(legacy)
			if err != nil || !reflect.DeepEqual(base, wantServer) {
				t.Fatalf("legacy server migration: want %q, got %q, error %v", wantServer, base, err)
			}
			client, err := getCommonArguments(legacy)
			if err != nil || !strings.Contains(strings.Join(client, " "), "-Djava.rmi.server.hostname="+localHost) {
				t.Fatalf("server migration affected client: %q, error %v", client, err)
			}
			updated := testManagedConnection(t, legacy, ClusterManagement{RESTAddress: localHost})
			if _, err := validateManagedOptions(updated); err != nil {
				t.Fatal(err)
			}
			if err := persistManagedConnection(updated, false); err != nil {
				t.Fatal(err)
			}
			_, loaded := readManagedPersistence(t, filename, "")
			if !reflect.DeepEqual(loaded.Clusters, []ClusterConnection{updated, untouched}) {
				t.Fatalf("persisted records: %+v", loaded.Clusters)
			}
			for label, read := range map[string]func(ClusterConnection) ([]string, error){"client": getCommonArguments, "server": serverBaseOptions} {
				got, err := read(loaded.Clusters[0])
				want := wantServer
				if label == "client" {
					want = append(append([]string{}, wantServer...), commonJVMOptions(updated)...)
				}
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("%s saved round trip: want %q, got %q, error %v", label, want, got, err)
				}
			}
			if again := testManagedConnection(t, loaded.Clusters[0], savedManagement(updated)); !reflect.DeepEqual(again, updated) {
				t.Fatalf("second normalization changed record: %+v", again)
			}
		})
	}
}

const (
	managementTestArgfile           = "@options"
	managementTestAgentHook         = "-Dcom.sun.management.agent.class=Custom"
	managementTestSplitOpens        = "--add-opens java.base/java.lang=ALL-UNNAMED"
	managementTestJava11Coherence   = "22.06.15"
	managementTestUnknownVersion    = "unknown"
	managementTestClasspath         = "-cp"
	managementTestModulePath        = "--module-path"
	managementTestModules           = "modules"
	managementTestUpgradePath       = "--upgrade-module-path"
	managementTestPatchModule       = "--patch-module"
	managementTestDescribeModule    = "--describe-module"
	managementTestNativeAccess      = "--enable-native-access"
	managementTestSplitNativeAccess = "--enable-native-access ALL-UNNAMED"
	managementTestHeapOption        = "-Xmx128m"
	managementTestLongClasspath     = "--class-path"
)

func TestManagementOptionScope(t *testing.T) {
	for _, key := range []string{"com.sun.management.jmxremote", "com.sun.management.jmxremote.host", "com.sun.management.config.file", "com.sun.management.agent.class", "java.rmi.server.hostname", "coherence.management.http", "coherence.management.http.address", "coherence.management.http.port", "coherence.management.config", "tangosol.coherence.management.http.address", "tangosol.management.http.port", "tangosol.management.config"} {
		t.Run(key, func(t *testing.T) {
			if _, err := validateServerOptions("scope", []string{"-D" + key + "=private-value"}, true); err == nil || strings.Contains(err.Error(), "private-value") {
				t.Fatalf("reserved key accepted or value disclosed: %v", err)
			}
		})
	}
	for _, arg := range []string{"-Dcom.sun.management.jmxremoteNote=yes", "-Dcoherence.management.httpNote=yes", "-Dcoherence.management.http.auth=basic", "-Dcoherence.management.http.provider=custom", "-Dcoherence.management=all", "-Dtangosol.management=all", "-Dtangosol.java.rmi.server.hostname=other", "-Dtangosol.com.sun.management.jmxremote=true", "-javaagent:agent.jar=payload", "-agentlib:jdwp", "-agentpath:/tmp/agent", "-Xrunjdwp", "-XX:OnError=echo ManagementServer", "-XX:VMOptionsFileNote=other", "-XX:ManagementServerNote=1", "-XmanagementNote", "-XoptionsfileNote=other", "--module-extra=value", "--source-extra=value"} {
		t.Run(arg, func(t *testing.T) {
			if _, err := validateServerOptions("scope", []string{arg}, true); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagementLauncherStructure(t *testing.T) {
	for _, args := range [][]string{{"--source=17"}, {"--source 17 Probe.java"}, {"--source-extra 17 Probe.java"}, {"--module=app/main"}, {managementTestClasspath, managementTestArgfile}, {"-p", managementTestArgfile}, {"--add-opens", managementTestArgfile}, {"-XX:ManagementServer=true"}, {"-XX:-ManagementServer"}, {"-Xmanagement:port=9999"}, {"-Dcustom=bad\x00value"}, {"--unknown", "Probe"}, {"-XX:+IgnoreUnrecognizedVMOptions", "--unknown", "Probe"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			if _, err := validateServerOptions("boundary", args, true); err == nil {
				t.Fatalf("accepted boundary violation %q", args)
			}
		})
	}
	for _, args := range [][]string{{managementTestClasspath, ""}, {"-classpath", "classes with spaces"}, {"--class-path=classes"}, {managementTestModulePath, `C:\module path`}, {"-p", managementTestModules}, {managementTestUpgradePath, managementTestModules}, {managementTestPatchModule, "bad grammar"}, {managementTestDescribeModule, "not.a.real.module"}, {"-d", "bad grammar"}, {managementTestNativeAccess, "ALL-UNNAMED"}, {"--add-opens", "MainClass"}, {"--add-modules", "bad module -Dcoherence.management.http=all"}, {"--add-opens=bad -jar app.jar"}, {"--class-path=@classes"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			before := append([]string(nil), args...)
			if _, err := validateServerOptions("pass-through", args, true); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(args, before) {
				t.Fatal("changed tokens")
			}
		})
	}
}

func TestManagementEnvironmentOptionSemantics(t *testing.T) {
	for _, name := range []string{managementTestJavaToolOptions, managementTestLegacyJavaOptions, managementTestJDKJavaOptions} {
		for _, value := range []string{"-Dcustom=literal\u00a0-Dcoherence.management.http=all", `-Dpath="C:\some path"`} {
			t.Run(name+"/"+value, func(t *testing.T) {
				managedTestState(t)
				t.Setenv(name, value)
				invalid := strings.HasPrefix(value, "@") || strings.HasPrefix(value, "-Dcom.sun") || name != managementTestJDKJavaOptions && strings.HasPrefix(value, "--add-opens ")
				if _, err := validateManagedOptions(ClusterConnection{}); (err != nil) != invalid {
					t.Fatalf("invalid=%t: %v", invalid, err)
				}
				if os.Getenv(name) != value {
					t.Fatal("changed environment")
				}
			})
		}
	}
}

func TestManagementLauncherCapability(t *testing.T) {
	for _, tc := range []struct {
		name, version, options string
		major                  int
		invalid                bool
	}{
		{"native-old", managementTestUnknownVersion, managementTestSplitNativeAccess, 11, true},
		{"native-current", managementTestJava11Coherence, managementTestSplitNativeAccess, 17, false},
		{"module-old", managementTestUnknownVersion, managementTestSplitOpens, 8, true},
		{"unknown-no-floor", managementTestUnknownVersion, managementTestHeapOption, 8, false},
		{"known-floor", "15.1.1-0-1", managementTestHeapOption, 11, true},
		{"native-attached", managementTestUnknownVersion, "--enable-native-access=ALL-UNNAMED", 11, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			managedTestState(t)
			queries := filepath.Join(t.TempDir(), "queries")
			t.Setenv("COHCTL_TEST_JAVA_VERSION_QUERIES", queries)
			fakeJava(t, javaFixture{Version: fmt.Sprintf("%d.0.21", tc.major), QueryMarker: true})
			connection := ClusterConnection{Name: managementFixtureName, ManuallyCreated: true, ConnectionType: httpType, ManagementPort: 30000, ClusterVersion: tc.version, Arguments: tc.options}
			var err error
			if tc.invalid {
				Config.Clusters = []ClusterConnection{connection}
				err = runClusterOperation(managementTestCommand(t), connection.Name, restartClusterCommand)
				if err == nil || !strings.Contains(err.Error(), "require") || !strings.Contains(err.Error(), "Java") {
					t.Fatalf("restart did not return the runtime requirement: %v", err)
				}
			} else {
				_, err = preflightManagedStartup(managementTestCommand(t), connection, 1, 0, false)
			}
			if (err != nil) != tc.invalid {
				t.Fatalf("invalid=%t: %v", tc.invalid, err)
			}
			data, readErr := os.ReadFile(queries)
			if readErr != nil || string(data) != "query\n" {
				t.Fatalf("want exactly one query, got %q: %v", data, readErr)
			}
		})
	}
	for _, source := range []string{managementTestLegacyRecord, managementTestConnectionName, managementTestProfileOptions, managementTestJavaToolOptions, managementTestLegacyJavaOptions, managementTestJDKJavaOptions} {
		t.Run("aggregate/"+source, func(t *testing.T) {
			managedTestState(t)
			connection := ClusterConnection{Arguments: managementTestSplitOpens}
			value := managementTestSplitNativeAccess
			want := 17
			switch source {
			case managementTestLegacyRecord:
				connection.Arguments += " " + value
			case managementTestConnectionName:
				connection.Management = &ClusterManagement{}
				connection.Arguments += " " + value
			case managementTestProfileOptions:
				profileValueParam = "capability"
				Config.Profiles = []ProfileValue{{Name: profileValueParam, Value: value}}
			default:
				if source != managementTestJDKJavaOptions {
					value = "--enable-native-access=ALL-UNNAMED"
					want = 9 // attached VM values do not need launcher consumption
				}
				t.Setenv(source, value)
			}
			got, err := validateManagedOptions(connection)
			if err != nil || got.requiredMajor != want {
				t.Fatalf("capability=%d, want=%d: %v", got.requiredMajor, want, err)
			}
		})
	}
}

func TestManagementLauncherPreflightBeforeStop(t *testing.T) {
	for _, value := range []string{"-Dcoherence.management.http.address=0.0.0.0", connectionTestStartupClass, "--source=17", managementTestArgfile} {
		t.Run(value, func(t *testing.T) {
			managedTestState(t)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				fmt.Fprint(w, `{"clusterName":"managed-fixture"}`)
			}))
			defer server.Close()
			connection := ClusterConnection{Name: managementFixtureName, ManuallyCreated: true, ClusterName: managementFixtureName, ConnectionType: httpType, ConnectionURL: server.URL, ManagementPort: managementListenerPort(t, server.Listener), Arguments: value, ClusterVersion: managementTestJava11Coherence}
			Config.Clusters = []ClusterConnection{connection}
			// A missing Java executable also proves policy rejection precedes the version query.
			t.Setenv("PATH", t.TempDir())
			err := runClusterOperation(managementTestCommand(t), managementFixtureName, restartClusterCommand)
			if err == nil || strings.Contains(err.Error(), "java -version") || calls != 0 {
				t.Fatalf("policy must fail before query or current-cluster access: calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestManagementEnvironmentOptionLexer(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  []string
	}{
		{" \t\n\r\v\f", nil},
		{`-Dcustom="with spaces" -Dpath='C:\some path'`, []string{"-Dcustom=with spaces", `-Dpath=C:\some path`}},
		{"-Dcustom=a\u00a0b\u2003c", []string{"-Dcustom=a\u00a0b\u2003c"}},
		{`--class-path ""`, []string{managementTestLongClasspath, ""}},
		{`-Dcustom=a"b c"d`, []string{"-Dcustom=ab cd"}},
		{`-Dcustom="a'b"`, []string{"-Dcustom=a'b"}},
	} {
		got, err := splitJVMEnvironmentOptions(tc.value)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q: got %q, want %q: %v", tc.value, got, tc.want, err)
		}
	}
	if _, err := splitJVMEnvironmentOptions(`-Dcustom="bad`); err == nil {
		t.Fatal("accepted unterminated environment quote")
	}
	for _, name := range []string{managementTestJavaToolOptions, managementTestLegacyJavaOptions} {
		t.Run(name, func(t *testing.T) {
			managedTestState(t)
			t.Setenv(name, "--add-opens -Dcoherence.management.http.address=0.0.0.0")
			if _, err := validateManagedOptions(ClusterConnection{}); err == nil || !strings.Contains(err.Error(), "reserved management property") || !strings.Contains(err.Error(), name) {
				t.Fatalf("malformed VM split hid reserved property: %v", err)
			}
		})
	}
}

func TestManagementArgumentAssembly(t *testing.T) {
	for _, firstOnly := range []bool{false, true} {
		t.Run(fmt.Sprintf("profile-first-%t", firstOnly), func(t *testing.T) {
			managedTestState(t)
			profileFirstParam = firstOnly
			profileValueParam = "capture"
			Config.Profiles = []ProfileValue{{Name: profileValueParam, Value: `--class-path "profile classes" --add-opens java.base/java.lang=ALL-UNNAMED -javaagent:benign.jar`}}
			connection := ClusterConnection{Name: "capture", ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: managementTestJava11Coherence, BaseClasspath: "base classes", Arguments: `-cp "saved classes" -Dcustom="saved value"`, StartupClass: "com.tangosol.net.DefaultCacheServer", Management: &ClusterManagement{RESTAddress: localHost}}
			wantMain := connection.StartupClass
			if firstOnly {
				serverStartClassParam = connectionTestStartupClass
				wantMain = serverStartClassParam
			}
			management := ClusterManagement{RESTAddress: localHost, JMXAddress: localHost, JMXHost: localHost, JMXPort: 30001}
			if _, err := validateManagedOptions(connection); err != nil {
				t.Fatal(err)
			}
			fakeJava(t, javaFixture{CaptureArguments: true})
			awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, processes []*managedProcess) error {
				for _, process := range processes {
					select {
					case <-process.done:
					case <-time.After(5 * time.Second):
						return fmt.Errorf("capture child did not exit")
					}
				}
				return nil
			}
			processes, err := runTestStartup(t, managementTestCommand(t), connection, 2, 0, management)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := cleanupManagedProcesses(processes); err != nil {
					t.Error(err)
				}
			}()
			for i, process := range processes {
				data, err := os.ReadFile(process.logFile)
				if err != nil {
					t.Fatal(err)
				}
				got := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
				prefix := []string{managementTestClasspath, "saved classes", "-Dcustom=saved value", managementTestClasspath, "base classes"}
				if len(got) < len(prefix) || !reflect.DeepEqual(got[:len(prefix)], prefix) || got[len(got)-1] != wantMain {
					t.Fatalf("launch boundary/order changed: %q", got)
				}
				profileIndex, restIndex, jmxIndex := -1, -1, -1
				for j, arg := range got {
					switch arg {
					case managementTestLongClasspath:
						profileIndex = j
						if j+1 >= len(got) || got[j+1] != "profile classes" {
							t.Fatal("profile operand changed")
						}
					case "-Dcoherence.management.http.address=" + localHost:
						restIndex = j
					case "-Dcom.sun.management.jmxremote.port=30001":
						jmxIndex = j
					}
				}
				for _, expected := range []string{"-Dcoherence.management.http=none"} {
					want := i != 0
					if slices.Contains(got, expected) != want {
						t.Fatalf("member=%d unexpected REST selection: %q", i, got)
					}
				}
				for _, expected := range []string{managementTestHTTPAll, "-Dcoherence.management.http.port=30000", "-Dcom.sun.management.jmxremote.rmi.port=30001", "-Dcom.sun.management.jmxremote.host=" + localHost, "-Djava.rmi.server.hostname=" + localHost} {
					if slices.Contains(got, expected) != (i == 0) {
						t.Fatalf("member=%d unexpected endpoint property %s: %q", i, expected, got)
					}
				}
				wantProfile := !firstOnly || i == 0
				if (profileIndex >= 0) != wantProfile || restIndex < 0 || restIndex <= profileIndex || (jmxIndex >= 0) != (i == 0) {
					t.Fatalf("member=%d: profile=%d REST=%d JMX=%d argv=%q", i, profileIndex, restIndex, jmxIndex, got)
				}
			}
		})
	}
}

func TestManagementOptionAliasesAndSources(t *testing.T) {
	for _, key := range []string{"management.http", "management.http.address", "management.http.port", "management.config"} {
		for _, prefix := range []string{"coherence.", "tangosol.coherence.", "tangosol."} {
			if !controlledManagementProperty("-D" + prefix + key + "=value") {
				t.Errorf("accepted reserved alias %s%s", prefix, key)
			}
		}
	}
}

func TestManagementEnvironmentOptionArity(t *testing.T) {
	options := managementModuleOptionCases()
	options = append(options, []struct{ option, operand string }{
		{managementTestLongClasspath, "classes"}, {managementTestModulePath, managementTestModules}, {managementTestUpgradePath, managementTestModules},
		{managementTestPatchModule, "app=patch"}, {managementTestNativeAccess, "ALL-UNNAMED"},
	}...)
	for _, tc := range options {
		for _, launcher := range []bool{false, true} {
			for _, attached := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/launcher-%t/attached-%t", tc.option, launcher, attached), func(t *testing.T) {
					args := []string{tc.option, tc.operand, managementTestCustomOption}
					if attached {
						args = []string{tc.option + "=" + tc.operand, managementTestCustomOption}
					}
					before := append([]string(nil), args...)
					_, err := validateServerOptions("arity", args, launcher)
					if (err != nil) != (!attached && !launcher) {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(before, args) {
						t.Fatal("validation changed tokens")
					}
				})
			}
		}
	}
}

var managementOptionSources = []string{managementTestLegacyRecord, managementTestConnectionName, managementTestProfileOptions, managementTestJavaToolOptions, managementTestLegacyJavaOptions, managementTestJDKJavaOptions}

func managedOptionSource(t *testing.T, source, value string) ClusterConnection {
	t.Helper()
	connection := ClusterConnection{ManuallyCreated: true}
	switch source {
	case managementTestLegacyRecord:
		connection.Arguments = "  " + value + "  "
	case managementTestConnectionName:
		connection.Management = &ClusterManagement{RESTAddress: localHost}
		connection.Arguments = value
	case managementTestProfileOptions:
		profileValueParam = "source-options"
		Config.Profiles = []ProfileValue{{Name: profileValueParam, Value: value}}
	default:
		t.Setenv(source, value)
	}
	return connection
}

func TestManagementOptionSources(t *testing.T) {
	for _, source := range managementOptionSources {
		for _, value := range []string{
			managementTestCustomOption, "--add-opens=java.base/java.lang=ALL-UNNAMED", managementTestSplitOpens, "--class-path classes",
			"-Dcoherence.management.http.port=9999", "-XX:Flags=hidden", managementTestHiddenFile, "MainClass", managementTestJarLaunch, "--add-opens -Dcoherence.management.http=all",
		} {
			t.Run(source+"/"+value, func(t *testing.T) {
				managedTestState(t)
				connection := managedOptionSource(t, source, value)
				before := connection
				vm := source == managementTestJavaToolOptions || source == managementTestLegacyJavaOptions
				valid := value == managementTestCustomOption || strings.HasPrefix(value, "--add-opens=") || !vm && (value == managementTestSplitOpens || value == "--class-path classes")
				options, err := validateManagedOptions(connection)
				if (err == nil) != valid {
					t.Fatalf("valid=%t: %v", valid, err)
				}
				if !reflect.DeepEqual(before, connection) {
					t.Fatal("validation mutated source")
				}
				if valid && (source == managementTestLegacyRecord || source == managementTestConnectionName) {
					if !reflect.DeepEqual(options.base, strings.Fields(value)) {
						t.Fatalf("base tokens changed: %q", options.base)
					}
				}
			})
		}
	}
}

func TestManagementLauncherOperandBoundaries(t *testing.T) {
	for _, option := range []string{managementTestClasspath, "-classpath", managementTestLongClasspath, "-p", managementTestModulePath, managementTestUpgradePath, managementTestPatchModule, "-d", managementTestDescribeModule, managementTestNativeAccess} {
		t.Run(option, func(t *testing.T) {
			for _, args := range [][]string{{option}, {option, managementTestHeapOption}, {option, managementTestHiddenFile}, {option, "@@escaped"}, {option, "value", managementTestMainClass}, {option, "--source-extra 17 Probe.java"}} {
				if _, err := validateServerOptions("boundaries", args, true); err == nil {
					t.Errorf("accepted %q", args)
				}
			}
		})
	}
}

const managementPreparedName = "prepared"

func TestManagementPreparedStartupArguments(t *testing.T) {
	managedTestState(t)
	fakeJava(t, javaFixture{Version: javaVersion17, Query: true, CaptureArguments: true})
	profileValueParam = managementPreparedName
	Config.Profiles = []ProfileValue{{Name: profileValueParam, Value: `--class-path "profile classes"`}}
	metricsStartPortParam, healthStartPortParam = 31000, 32000
	oldMachine, oldRack, oldSite, oldRole := machineParam, rackParam, siteParam, roleParam
	t.Cleanup(func() { machineParam, rackParam, siteParam, roleParam = oldMachine, oldRack, oldSite, oldRole })
	machineParam, rackParam, siteParam, roleParam = "machine", "rack", "site", "role"
	connection := ClusterConnection{Name: managementPreparedName, ClusterName: managementPreparedName, ManuallyCreated: true, ManagementPort: 30000, ClusterVersion: defaultCoherenceVersion, Arguments: "-Dcustom=base", StartupClass: connectionTestStartupClass}
	plan, err := preflightManagedStartup(managementTestCommand(t), connection, 2, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	// Creation fills the classpath later. Changing raw inputs after preflight must
	// not change validated launch arguments or provoke another parse/validation.
	plan.connection.BaseClasspath = "resolved classes"
	Config.Profiles = nil
	startupDelayParam = "not-a-delay"
	serverStartClassParam = "invalid class"
	metricsStartPortParam, healthStartPortParam = 0, 0
	machineParam, rackParam, siteParam, roleParam = "changed-machine", "changed-rack", "changed-site", "changed-role"
	awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, children []*managedProcess) error {
		for _, p := range children {
			select {
			case <-p.done:
			case <-time.After(5 * time.Second):
				return fmt.Errorf("capture child timed out")
			}
		}
		return nil
	}
	children, err := runPreparedStartup(managementTestCommand(t), plan, managedNoSave)
	cleanupTestReap(t, children)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range children {
		data, err := os.ReadFile(p.logFile)
		if err != nil {
			t.Fatal(err)
		}
		args := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		if !reflect.DeepEqual(args[:3], []string{"-Dcustom=base", "-cp", "resolved classes"}) {
			t.Fatal(args)
		}
		metrics, profile, health := slices.Index(args, fmt.Sprintf("-Dcoherence.metrics.http.port=%d", 31000+i)), slices.Index(args, "--class-path"), slices.Index(args, fmt.Sprintf("-Dcoherence.health.http.port=%d", 32000+i))
		if metrics < 0 || profile <= metrics || health <= profile || args[profile+1] != "profile classes" {
			t.Fatal("argument order changed", args)
		}
		for _, property := range []string{"machine", "rack", "site", "role"} {
			if !slices.Contains(args, "-Dcoherence."+property+"="+property) {
				t.Fatal("member metadata changed", args)
			}
		}
		if args[len(args)-1] != connectionTestStartupClass {
			t.Fatal("main class changed", args)
		}
	}
}

// A corrupt unnamed saved profile was historically inspected but never selected
// for launch when --profile was absent. Keep validation and selection distinct.
func TestManagementUnnamedProfileNotSelected(t *testing.T) {
	managedTestState(t)
	Config.Profiles = []ProfileValue{{Value: managementTestCustomOption}}
	options, err := validateManagedOptions(ClusterConnection{})
	if err != nil {
		t.Fatal(err)
	}
	plan := prepareManagedMembers(ClusterConnection{}, ClusterManagement{RESTAddress: localHost}, options, 1, 0, 0)
	if slices.Contains(plan.members[0].arguments, managementTestCustomOption) {
		t.Fatal("unnamed profile selected")
	}
}
