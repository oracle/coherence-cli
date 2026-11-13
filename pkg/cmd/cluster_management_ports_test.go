/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	managementTestLegacyRESTAddress = "legacy-rest-address"
	managementTestNewWildcardREST   = "new-wildcard-rest"

	managementTestLegacyNewJMXPort      = "legacy-new-jmx-port"
	managementTestWildcardRESTMigration = "wildcard-rest-migration"

	managementTestLegacyJMXAddress = "legacy-jmx-address"
	managementTestRESTLabel        = "Management REST"
	managementTestJMXLabel         = "JMX"
	managementTestOldURL           = "http://127.0.0.1:30000/old"
	managementTestStartStopped     = "start-stopped"
)

func managementPortListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp4", net.JoinHostPort(localHost, "0"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func managementListenerPort(t *testing.T, listener net.Listener) int32 {
	t.Helper()
	port := listener.Addr().(*net.TCPAddr).Port
	if port < 1 || port > 65535 {
		t.Fatalf("unexpected TCP fixture port: %d", port)
		return 0
	}
	return int32(port)
}

func unavailableManagementAddress(t *testing.T) string {
	t.Helper()
	// Do not assume that another loopback alias or a documentation address is
	// unavailable on this host. Check the actual ephemeral bind first.
	for _, address := range []string{"192.0.2.123", "198.51.100.123", "203.0.113.123"} {
		listener, err := net.Listen("tcp4", net.JoinHostPort(address, "0"))
		if err != nil {
			t.Logf("observed unavailable address %s: %v", address, err)
			return address
		}
		_ = listener.Close()
	}
	t.Skip("no unavailable candidate bind on this host")
	return ""
}

func managementPortSeam(t *testing.T) {
	t.Helper()
	old := listenManagementEndpoint
	t.Cleanup(func() { listenManagementEndpoint = old })
}

func saveManagementFixture(t *testing.T, connection ClusterConnection) []byte {
	t.Helper()
	Config.Clusters[0] = connection
	viper.Set(clusterKey, Config.Clusters)
	if err := writeManagedConfigAtomically(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(viper.ConfigFileUsed())
	if err != nil {
		t.Fatal(err)
	}
	return before
}

func TestManagementReplacementEndpointRejection(t *testing.T) {
	for _, name := range []string{managementTestLegacyRESTAddress, managementTestLegacyJMXAddress, "wildcard-rest-address", "wildcard-jmx-address", "new-jmx-port", managementTestLegacyNewJMXPort, managementTestNewWildcardREST} {
		t.Run(name, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			child := stopTestChild(t)
			connection := stopTestConnection(t, []int{child.command.Process.Pid}, nil)
			busyJMX := managementPortListener(t)
			jmxPort := managementListenerPort(t, busyJMX)
			connection.Management = &ClusterManagement{RESTAddress: localHost, JMXPort: jmxPort, JMXAddress: localHost, JMXHost: localHost}
			flags := []string{"--allow-insecure-remote-management"}
			label, address := "", ""
			switch name {
			case managementTestLegacyRESTAddress, "wildcard-rest-address":
				address = unavailableManagementAddress(t)
				label = managementTestRESTLabel
				flags = append(flags, "--management-address="+address)
				if name == managementTestLegacyRESTAddress {
					connection.Management = nil
				} else {
					connection.Management.RESTAddress = wildcardIPv4
				}
			case managementTestLegacyJMXAddress, "wildcard-jmx-address":
				address = unavailableManagementAddress(t)
				label = managementTestJMXLabel
				flags = append(flags, fmt.Sprintf("--jmx-port=%d", jmxPort), "--jmx-address="+address)
				if name == managementTestLegacyJMXAddress {
					connection.Management = nil
				} else {
					connection.Management.JMXAddress = wildcardIPv4
				}
			case "new-jmx-port", managementTestLegacyNewJMXPort:
				connection.Management.JMXPort = 0
				if name == managementTestLegacyNewJMXPort {
					connection.Management = nil
				}
				flags = append(flags, fmt.Sprintf("--jmx-port=%d", jmxPort))
				label, address = managementTestJMXLabel, busyJMX.Addr().String()
			case managementTestNewWildcardREST:
				// Some hosts permit wildcard/specific coexistence with these
				// socket options. Simulate a reported conflict, not its presence.
				managementPortSeam(t)
				listenManagementEndpoint = func(network, address string) (net.Listener, error) {
					if address == net.JoinHostPort(wildcardIPv4, fmt.Sprint(connection.ManagementPort)) {
						return nil, os.NewSyscallError("bind", errManagementAddressInUse)
					}
					return net.Listen(network, address)
				}
				flags = append(flags, "--management-address="+wildcardIPv4)
				label, address = managementTestRESTLabel, net.JoinHostPort(wildcardIPv4, fmt.Sprint(connection.ManagementPort))
			}
			// Wildcard settings above model old metadata over a loopback-only
			// fixture. No insecure remote management service is launched.
			before := saveManagementFixture(t, connection)
			launchMarker := filepath.Join(t.TempDir(), "launched")
			t.Setenv("COHCTL_ENDPOINT_TEST_LAUNCH", launchMarker)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: windowsEndpointLaunchKey, Sleep: true})
			finds, kills, starts, writes := 0, 0, 0, 0
			findProcessForStop = func(pid int) (*os.Process, error) { finds++; return os.FindProcess(pid) }
			killProcessForStop = func(p *os.Process) error { kills++; return p.Kill() }
			awaitManagedStartup = func(_ *cobra.Command, _ ClusterConnection, processes []*managedProcess) error {
				starts++
				return cleanupManagedProcesses(processes)
			}
			writeManagedConfig = func() error { writes++; return nil }
			automaticallyConfirm = true
			err := runClusterOperation(managementTestCommand(t, flags...), connection.Name, restartClusterCommand)
			if err == nil {
				t.Fatal("invalid replacement accepted")
			}
			for _, want := range []string{label, address, "unavailable", "choose an available local"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("missing %q: %v", want, err)
				}
			}
			if finds != 0 || kills != 0 || starts != 0 || writes != 0 {
				t.Fatalf("side effects: find=%d kill=%d start=%d write=%d", finds, kills, starts, writes)
			}
			if _, err := os.Stat(launchMarker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("replacement launched: %v", err)
			}
			select {
			case <-child.done:
				t.Fatal("old owned member exited")
			default:
			}
			after, err := os.ReadFile(viper.ConfigFileUsed())
			if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config.Clusters[0], connection) || !reflect.DeepEqual(viper.Get(clusterKey), Config.Clusters) {
				t.Fatalf("preflight changed saved bytes/settings/URL: %v", err)
			}
		})
	}
}

func TestManagementPortConflictExemptions(t *testing.T) {
	// Socket errors/alternate address availability are simulated here, so the
	// overlap cases do not rely on host interface aliases or remote listeners.
	for _, tt := range []struct {
		name     string
		old      *ClusterManagement
		url      string
		proposed ClusterManagement
		allow    bool
	}{
		{"saved-rest-and-jmx", &ClusterManagement{localHost, 30001, localHost, localHost}, "", ClusterManagement{localHost, 30001, localHost, localHost}, true},
		{"legacy-loopback", nil, managementTestOldURL, ClusterManagement{RESTAddress: localHost}, true},
		{"legacy-localhost", nil, "http://localhost:30000/old", ClusterManagement{RESTAddress: localHost}, true},
		{"legacy-changed-rest", nil, managementTestOldURL, ClusterManagement{RESTAddress: managementTestIP}, false},
		{"legacy-unknown-url", nil, "", ClusterManagement{RESTAddress: localHost}, false},
		{"legacy-url-other-port", nil, "http://127.0.0.1:30002/old", ClusterManagement{RESTAddress: localHost}, false},
		{"legacy-url-dns", nil, "http://server.example:30000/old", ClusterManagement{RESTAddress: localHost}, false},
		{"legacy-url-wildcard", nil, "http://0.0.0.0:30000/old", ClusterManagement{RESTAddress: localHost}, false},
		{"saved-wildcard-to-reachable-rest", &ClusterManagement{RESTAddress: wildcardIPv4}, managementTestOldURL, ClusterManagement{RESTAddress: localHost}, true},
		{"saved-wildcard-to-other-rest", &ClusterManagement{RESTAddress: wildcardIPv4}, managementTestOldURL, ClusterManagement{RESTAddress: managementTestIP}, false},
		{managementTestNewWildcardREST, &ClusterManagement{RESTAddress: localHost}, "", ClusterManagement{RESTAddress: wildcardIPv4}, false},
		{"unchanged-wildcard-rest", &ClusterManagement{RESTAddress: wildcardIPv4}, "", ClusterManagement{RESTAddress: wildcardIPv4}, true},
		{"changed-wildcard-jmx", &ClusterManagement{localHost, 30001, wildcardIPv4, localHost}, "", ClusterManagement{localHost, 30001, localHost, localHost}, false},
		{"new-wildcard-jmx", &ClusterManagement{localHost, 30001, localHost, localHost}, "", ClusterManagement{localHost, 30001, wildcardIPv4, localHost}, false},
		{"unchanged-wildcard-jmx", &ClusterManagement{localHost, 30001, wildcardIPv4, localHost}, "", ClusterManagement{localHost, 30001, wildcardIPv4, localHost}, true},
		{"changed-jmx-port", &ClusterManagement{localHost, 30001, localHost, localHost}, "", ClusterManagement{localHost, 30002, localHost, localHost}, false},
		{"legacy-new-jmx", nil, managementTestOldURL, ClusterManagement{localHost, 30001, localHost, localHost}, false},
		{"jmx-is-not-old-rest", &ClusterManagement{RESTAddress: localHost}, "", ClusterManagement{localHost, 30000, localHost, localHost}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			managementPortSeam(t)
			probes, fixed := 0, 0
			listenManagementEndpoint = func(network, address string) (net.Listener, error) {
				if network != "tcp4" {
					t.Fatal(network)
				}
				if strings.HasSuffix(address, ":0") {
					probes++
					return net.Listen("tcp4", "127.0.0.1:0")
				}
				fixed++
				return nil, &net.OpError{Op: "listen", Net: network, Err: os.NewSyscallError("bind", errManagementAddressInUse)}
			}
			connection := ClusterConnection{ManagementPort: 30000, ConnectionURL: tt.url, Management: tt.old}
			err := checkManagementPorts(connection, tt.proposed, true)
			if (err == nil) != tt.allow {
				t.Fatalf("allow=%t error=%v", tt.allow, err)
			}
			if probes == 0 || probes != fixed {
				t.Fatalf("address/fixed probes %d/%d", probes, fixed)
			}
			if tt.allow && tt.proposed.JMXPort > 0 && probes != 2 {
				t.Fatal("JMX was not probed")
			}
		})
	}
}

func TestManagementPortErrorsAreNotExempt(t *testing.T) {
	for _, failAt := range []int{1, 2, 3, 4} {
		for _, cause := range []error{os.ErrPermission, errors.New("injected socket failure"), errManagementAddressInUse} {
			// Only a fixed endpoint's address-in-use error is exempt. Even
			// address-in-use from the ephemeral address probe must be returned.
			if failAt%2 == 0 && cause == errManagementAddressInUse {
				continue
			}
			t.Run(fmt.Sprintf("probe-%d/%v", failAt, cause), func(t *testing.T) {
				managementPortSeam(t)
				m := ClusterManagement{localHost, 30001, localHost, localHost}
				calls := 0
				listenManagementEndpoint = func(network, _ string) (net.Listener, error) {
					calls++
					if calls == failAt {
						return nil, &net.OpError{Op: "listen", Net: network, Err: os.NewSyscallError("bind", cause)}
					}
					return net.Listen("tcp4", "127.0.0.1:0")
				}
				err := checkManagementPorts(ClusterConnection{ManagementPort: 30000, Management: &m}, m, true)
				if !errors.Is(err, cause) {
					t.Fatalf("other socket error suppressed: %v", err)
				}
				label := managementTestRESTLabel
				if failAt > 2 {
					label = managementTestJMXLabel
				}
				if !strings.Contains(err.Error(), label) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestManagementPortsRealListeners(t *testing.T) {
	rest, jmx := managementPortListener(t), managementPortListener(t)
	m := ClusterManagement{localHost, managementListenerPort(t, jmx), localHost, localHost}
	connection := ClusterConnection{ManagementPort: managementListenerPort(t, rest), Management: &m}
	if err := checkManagementPorts(connection, m, true); err != nil {
		t.Fatal("same busy endpoints rejected", err)
	}
	if err := checkManagementPorts(connection, m, false); err == nil || !strings.Contains(err.Error(), "Management REST endpoint") {
		t.Fatal("busy start/stopped-restart REST accepted", err)
	}
	_ = rest.Close()
	if err := checkManagementPorts(connection, m, false); err == nil || !strings.Contains(err.Error(), "JMX endpoint") {
		t.Fatal("busy start/stopped-restart JMX accepted", err)
	}
	_ = jmx.Close()
	if err := checkManagementPorts(connection, m, false); err != nil {
		t.Fatal("available start/stopped-restart endpoints rejected", err)
	}
}

func TestManagementReplacementEndpointSuccess(t *testing.T) {
	for _, name := range []string{"legacy-migration", "same-rest-and-jmx", managementTestWildcardRESTMigration, managementTestStartStopped} {
		t.Run(name, func(t *testing.T) {
			managedTestState(t)
			stopTestSeams(t)
			running := name != managementTestStartStopped
			var child *managedProcess
			var pids []int
			if running {
				child = stopTestChild(t)
				pids = []int{child.command.Process.Pid}
			}
			var stopped atomic.Bool
			connection, endpoint := stopTestConnectionEndpoint(t, pids, &stopped)
			flags := []string{}
			var jmx net.Listener
			if name == "legacy-migration" || name == managementTestWildcardRESTMigration {
				connection.Management = nil
				flags = append(flags, "--management-address=localhost")
				if name == managementTestWildcardRESTMigration {
					connection.Management = &ClusterManagement{RESTAddress: wildcardIPv4}
				}
			}
			if name == "same-rest-and-jmx" {
				jmx = managementPortListener(t)
				connection.Management = &ClusterManagement{localHost, managementListenerPort(t, jmx), localHost, localHost}
			}
			if !running {
				free := managementPortListener(t)
				connection.ManagementPort = managementListenerPort(t, free)
				_ = free.Close()
			}
			saveManagementFixture(t, connection)
			fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Sleep: true})
			var attempted []int
			killProcessForStop = func(p *os.Process) error {
				attempted = append(attempted, p.Pid)
				err := p.Kill()
				if err == nil {
					stopped.Store(true)
					endpoint.Close()
					if jmx != nil {
						_ = jmx.Close()
					}
				}
				return err
			}
			var replacement []*managedProcess
			t.Cleanup(func() {
				if err := cleanupManagedProcesses(replacement); err != nil {
					t.Errorf("replacement fixture cleanup: %v", err)
				}
			})
			awaitManagedStartup = func(_ *cobra.Command, proposed ClusterConnection, processes []*managedProcess) error {
				replacement = processes
				if proposed.ConnectionURL != managementURL(savedManagement(proposed), proposed.ManagementPort) {
					t.Fatal("readiness did not use proposed URL")
				}
				if proposed.ConnectionURL == connection.ConnectionURL {
					t.Fatal("readiness used old fixture URL")
				}
				return nil // Simulate Coherence readiness; the launched child is a sleeper.
			}
			writes := 0
			writeManagedConfig = func() error { writes++; return writeManagedConfigAtomically() }
			automaticallyConfirm = true
			operation := restartClusterCommand
			if name == managementTestStartStopped {
				operation = startClusterCommand
			}
			if err := runClusterOperation(managementTestCommand(t, flags...), connection.Name, operation); err != nil {
				t.Fatal(err)
			}
			if running {
				waitStopTestChild(t, child)
			}
			if !reflect.DeepEqual(attempted, pids) || len(replacement) != 1 || writes != 1 {
				t.Fatalf("attempts=%v want=%v replacements=%d writes=%d", attempted, pids, len(replacement), writes)
			}
			saved := Config.Clusters[0]
			if saved.Management == nil || saved.Management.RESTAddress != localHost || saved.ConnectionURL != managementURL(*saved.Management, saved.ManagementPort) {
				t.Fatalf("wrong saved endpoint: %+v", saved)
			}
			reader := viper.New()
			reader.SetConfigFile(viper.ConfigFileUsed())
			if err := reader.ReadInConfig(); err != nil {
				t.Fatal(err)
			}
			var loaded CoherenceCLIConfig
			if err := reader.Unmarshal(&loaded); err != nil || !reflect.DeepEqual(loaded.Clusters[0], saved) {
				t.Fatalf("persisted replacement mismatch: %v", err)
			}
		})
	}
}

func TestManagementPortsUnavailableAddress(t *testing.T) {
	address := unavailableManagementAddress(t)
	for _, name := range []string{"legacy-rest", "wildcard-rest", "wildcard-jmx", "unchanged-jmx"} {
		t.Run(name, func(t *testing.T) {
			rest := managementPortListener(t)
			jmx := managementPortListener(t)
			m := ClusterManagement{localHost, 0, "", ""}
			connection := ClusterConnection{ManagementPort: managementListenerPort(t, rest)}
			if strings.HasSuffix(name, "rest") {
				m.RESTAddress = address
				if name == "wildcard-rest" {
					connection.Management = &ClusterManagement{RESTAddress: wildcardIPv4}
				}
			} else {
				_ = rest.Close() // Let the JMX address be the first failing endpoint.
				m.JMXPort, m.JMXAddress, m.JMXHost = managementListenerPort(t, jmx), address, address
				old := m
				if name == "wildcard-jmx" {
					old.JMXAddress = wildcardIPv4
				}
				connection.Management = &old
			}
			for _, replacing := range []bool{false, true} {
				err := checkManagementPorts(connection, m, replacing)
				if err == nil || !strings.Contains(err.Error(), address) {
					t.Fatalf("replacing=%t unavailable address accepted: %v", replacing, err)
				}
			}
		})
	}
}

func TestManagementStoppedEndpointConflicts(t *testing.T) {
	for _, operation := range []string{startClusterCommand} {
		for _, endpoint := range []string{managementTestRESTLabel, managementTestJMXLabel} {
			t.Run(operation+"/"+endpoint, func(t *testing.T) {
				managedTestState(t)
				connection := stopTestConnection(t, nil, nil)
				if endpoint == managementTestJMXLabel {
					free := managementPortListener(t)
					connection.ManagementPort = managementListenerPort(t, free)
					_ = free.Close()
					jmx := managementPortListener(t)
					connection.Management = &ClusterManagement{localHost, managementListenerPort(t, jmx), localHost, localHost}
				}
				before := saveManagementFixture(t, connection)
				fakeJava(t, javaFixture{Version: javaVersion17, Query: true, Marker: windowsEndpointLaunchKey})
				launchMarker := filepath.Join(t.TempDir(), "launched")
				t.Setenv("COHCTL_ENDPOINT_TEST_LAUNCH", launchMarker)
				writes := 0
				writeManagedConfig = func() error { writes++; return nil }
				automaticallyConfirm = true
				err := runClusterOperation(managementTestCommand(t), connection.Name, operation)
				if err == nil || !strings.Contains(err.Error(), endpoint+" endpoint") {
					t.Fatalf("busy stopped endpoint accepted: %v", err)
				}
				if _, err := os.Stat(launchMarker); !errors.Is(err, os.ErrNotExist) || writes != 0 {
					t.Fatalf("launch/write reached: %v writes=%d", err, writes)
				}
				after, err := os.ReadFile(viper.ConfigFileUsed())
				if err != nil || !bytes.Equal(before, after) || !reflect.DeepEqual(Config.Clusters[0], connection) {
					t.Fatalf("stopped conflict persisted endpoints: %v", err)
				}
			})
		}
	}
}
