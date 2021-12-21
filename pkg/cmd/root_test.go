/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/test/test_utils"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	g := NewGomegaWithT(t)
	cliCmd := Initialize(nil)
	test_utils.EnsureCommandContains(g, t, cliCmd, "CLI Version:", "version")
}

func TestSettingConfigDirectoryOnly(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)
	dir := test_utils.CreateTempDirectory("temp")

	test_utils.EnsureCommandContains(g, t, cliCmd, "CLI Version:", "--config-dir", dir, "version")

	// we should see a file in the temp directory with the name of cohctl.yaml
	g.Expect(test_utils.FileExistsInDirectory(dir, configName+"."+configType)).To(Equal(true))
}

func TestSettingConfigFileOnly(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)
	dir := test_utils.CreateTempDirectory("temp")
	err := os.Mkdir(dir, 0755)
	if err != nil {
		t.Fatal("Unable to create directory", err)
	}

	file := filepath.Join(dir, "my-config.yaml")

	test_utils.EnsureCommandContains(g, t, cliCmd, "CLI Version:", "--config", file, "version")

	// we should see a file in the temp directory with the name of cohctl.yaml
	g.Expect(test_utils.FileExistsInDirectory(dir, "my-config.yaml")).To(Equal(true))
}

// TestContextCommands tests the get, set and clear context commands
func TestContextCommands(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml("config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(file)
	})

	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getContextMsg+"\n", "--config", file, "get", "context")

	// try to set a context when there is no cluster
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, UnableToFindClusterMsg, "--config", file, "set", "context", "cluster1")

	// manually add a cluster, so we have a context to set
	newCluster := ClusterConnection{Name: "cluster1", ConnectionType: "http", ConnectionURL: "dummy",
		DiscoveryType: "manual", ClusterVersion: "21.06.1"}
	Config.Clusters = append(Config.Clusters, newCluster)

	// set the context
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setContextMsg+"cluster1\n", "--config", file, "set", "context", "cluster1")

	g.Expect(viper.GetString(currentContextKey)).To(Equal("cluster1"))

	// clear the context
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, clearContextMessage+"\n", "--config", file, "clear", "context")

	g.Expect(viper.GetString(currentContextKey)).To(Equal(""))
}

// TestDebugCommands tests the get and set debug commands
func TestDebugCommands(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml("config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(file)
	})

	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getDebugMsg+"off\n", "--config", file, "get", "debug")

	// set the debug to true
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setDebugMsg+"on\n", "--config", file, "set", "debug", "on")

	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getDebugMsg+"on\n", "--config", file, "get", "debug")

	// set the debug to false
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setDebugMsg+"off\n", "--config", file, "set", "debug", "off")

	// set the debug to invalid value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, setDebugError, "--config", file, "set", "debug", "dont-know")
}

// TestIgnoreCertsCommands tests the get and set ignore-certs commands
func TestIgnoreCertsCommands(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml("config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(file)
	})

	// get the default value
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getIgnoreCertsMsg+"false\n", "--config", file, "get", "ignore-certs")

	// set the ignore-certs to true
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setIgnoreCertsMsg+"true\n", "--config", file, "set", "ignore-certs", "true")

	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getIgnoreCertsMsg+"true\n", "--config", file, "get", "ignore-certs")

	// set the ignore-certs to false
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setIgnoreCertsMsg+"false\n", "--config", file, "set", "ignore-certs", "false")

	// set the ignore-certs to invalid value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, setIgnoreCertsError, "--config", file, "set", "ignore-certs", "dont-know")
}

// TestGetLogsCommands tests the get logs command
func TestGetLogsCommands(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml("config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(file)
	})

	// generate a log message
	test_utils.EnsureCommandContains(g, t, cliCmd, "CLI Version:", "version")

	// ensure we can get logs
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "INFO,CLI Details", "--config", file, "get", "logs")
}

// TestTimeoutCommands tests the get adn set timeout commands
func TestTimeoutCommands(t *testing.T) {
	cliCmd := Initialize(nil)
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml("config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(file)
	})

	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, getTimeoutMessage+"30\n", "--config", file, "get", "timeout")

	// set the timeout to 65
	test_utils.EnsureCommandOutputEquals(g, t, cliCmd, setTimeoutMsg+"65\n", "--config", file, "set", "timeout", "65")

	// set the timeout to an invalid value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "invalid", "--config", file, "set", "timeout", "123c")
}

func TestIsStatusHASaferThan(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(isStatusHASaferThan("NODE-SAFE", "NODE-SAFE")).Should(Equal(true))
	g.Expect(isStatusHASaferThan("NODE-SAFE", "MACHINE-SAFE")).Should(Equal(false))
	g.Expect(isStatusHASaferThan("MACHINE-SAFE", "SITE-SAFE")).Should(Equal(false))
	g.Expect(isStatusHASaferThan("RACK-SAFE", "SITE-SAFE")).Should(Equal(false))
	g.Expect(isStatusHASaferThan("NODE-SAFE", "ENDANGERED")).Should(Equal(true))
	g.Expect(isStatusHASaferThan("MACHINE-SAFE", "NODE-SAFE")).Should(Equal(true))
	g.Expect(isStatusHASaferThan("RACK-SAFE", "MACHINE-SAFE")).Should(Equal(true))
	g.Expect(isStatusHASaferThan("SITE-SAFE", "MACHINE-SAFE")).Should(Equal(true))
	g.Expect(isStatusHASaferThan("SITE-SAFE", "RACK-SAFE")).Should(Equal(true))
}

func TestGetDataFetcher(t *testing.T) {
	var (
		g           = NewGomegaWithT(t)
		err         error
		dataFetcher fetcher.Fetcher
		ok          bool
	)

	setConfig(g)

	dataFetcher, err = GetDataFetcher("one")
	g.Expect(err).To(BeNil())
	g.Expect(dataFetcher).To(Not(BeNil()))
	_, ok = interface{}(dataFetcher).(fetcher.HTTPFetcher)
	g.Expect(ok).To(BeTrue())

	_, err = GetDataFetcher("not-here")
	g.Expect(err).To(Not(BeNil()))
}

func TestGetClusterConnection(t *testing.T) {
	var (
		g          = NewGomegaWithT(t)
		found      bool
		connection ClusterConnection
	)

	setConfig(g)

	found, connection = GetClusterConnection("one")
	g.Expect(found).To(Equal(true))
	g.Expect(connection.Name).To(Equal("one"))

	found, connection = GetClusterConnection("two")
	g.Expect(found).To(Equal(true))
	g.Expect(connection.Name).To(Equal("two"))

	found, connection = GetClusterConnection("three")
	g.Expect(found).To(Equal(false))
}

func TestGetConnectionNameFromContextOrArg(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		err     error
		cluster string
	)

	Config.Clusters = make([]ClusterConnection, 0)

	// test with -c local and no current context
	Config.CurrentContext = ""
	clusterConnection = "local"
	cluster, err = GetConnectionNameFromContextOrArg()
	g.Expect(err).To(BeNil())
	g.Expect(cluster).To(Equal("local"))

	// test with -c local and context set to "context". -c should win
	Config.CurrentContext = "context"
	cluster, err = GetConnectionNameFromContextOrArg()
	g.Expect(err).To(BeNil())
	g.Expect(cluster).To(Equal("local"))

	// test with no -c and the context
	Config.CurrentContext = "local"
	clusterConnection = ""
	cluster, err = GetConnectionNameFromContextOrArg()
	g.Expect(err).To(BeNil())
	g.Expect(cluster).To(Equal("local"))

	// test with neither -c or context
	// test with no -c and the context
	Config.CurrentContext = ""
	clusterConnection = ""
	_, err = GetConnectionNameFromContextOrArg()
	g.Expect(err).To(Not(BeNil()))
}

func TestDeduplicateServices(t *testing.T) {
	var (
		g               = NewGomegaWithT(t)
		servicesSummary = config.ServicesSummaries{}
	)

	services1 := generateServiceSummary("DistributedCache1", "DistributedCache", 3)
	services2 := generateServiceSummary("DistributedCache2", "FederatedCache", 1)

	servicesSummary.Services = append(services1, services2...)

	result := DeduplicateServices(servicesSummary, "all")
	g.Expect(len(result)).To(Equal(2))

	result = DeduplicateServices(servicesSummary, "FederatedCache")
	g.Expect(len(result)).To(Equal(1))
}

func TestDeduplicateSessions(t *testing.T) {
	var (
		g         = NewGomegaWithT(t)
		summaries = config.HTTPSessionSummaries{}
	)

	session1 := generateHTTPSessionSummary("app1", 3)
	session2 := generateHTTPSessionSummary("app2", 1)

	summaries.HTTPSessions = append(session1, session2...)

	result := DeduplicateSessions(summaries)
	g.Expect(len(result)).To(Equal(2))

	for _, value := range result {
		if value.AppID == "app1" {
			g.Expect(value.SessionUpdates).To(Equal(int64(3)))
			g.Expect(value.ReapedSessionsTotal).To(Equal(int64(30)))
			g.Expect(value.SessionAverageSize).To(Equal(int32(100)))
			g.Expect(value.AverageReapDuration).To(Equal(int64(100)))
		} else if value.AppID == "app2" {
			g.Expect(value.SessionUpdates).To(Equal(int64(1)))
			g.Expect(value.ReapedSessionsTotal).To(Equal(int64(10)))
			g.Expect(value.SessionAverageSize).To(Equal(int32(100)))
			g.Expect(value.AverageReapDuration).To(Equal(int64(100)))
		}
	}
}

func setConfig(g *WithT) {
	Config.Clusters = make([]ClusterConnection, 0)
	Config.Clusters = append(Config.Clusters, ClusterConnection{Name: "one", ConnectionType: "http", ConnectionURL: "url-one"})
	Config.Clusters = append(Config.Clusters, ClusterConnection{Name: "two", ConnectionType: "http", ConnectionURL: "url-two"})
	g.Expect(len(Config.Clusters)).To(Equal(2))
}

func TestErrorSink(t *testing.T) {
	var (
		g          = NewGomegaWithT(t)
		errorCount = 10000
		wg         sync.WaitGroup
		errorSink  = createErrorSink()
	)

	wg.Add(errorCount)
	for i := 0; i < errorCount; i++ {
		go func(iteration int) {
			defer wg.Done()
			errorSink.AppendError(fmt.Errorf("%d", iteration))
		}(i)
	}

	wg.Wait()
	errorList := errorSink.GetErrors()
	g.Expect(len(errorList)).To(Equal(errorCount))

	// check to see that we have the data we expect
	valuesMap := make(map[string]string)
	for _, value := range errorList {
		text := value.Error()
		if _, ok := valuesMap[text]; !ok {
			// add
			valuesMap[text] = text
		}
	}

	// ensure we have the exact number of unique values
	g.Expect(len(valuesMap)).To(Equal(len(errorList)))
}

func TestDeduplicatePersistenceServices(t *testing.T) {
	var (
		g               = NewGomegaWithT(t)
		servicesSummary = config.ServicesSummaries{}
	)

	services1 := generateServiceSummary("DistributedCache1", "DistributedCache", 3)
	services2 := generateServiceSummary("DistributedCache2", "FederatedCache", 1)
	services3 := generateServiceSummary("ReplicatedCache", "ReplicatedCache", 1)

	servicesSummary.Services = append(services1, services2...)
	servicesSummary.Services = append(servicesSummary.Services, services3...)
	result := DeduplicatePersistenceServices(servicesSummary)
	g.Expect(len(result)).To(Equal(2))

	for _, value := range result {
		if value.ServiceName == "DistributedCache1" {
			g.Expect(value.PersistenceActiveSpaceUsed).To(Equal(int64(30)))
			g.Expect(value.PersistenceLatencyAverageTotal).To(Equal(1.0))
			g.Expect(value.PersistenceLatencyMax).To(Equal(int64(3)))
		} else {
			// federated
			g.Expect(value.PersistenceActiveSpaceUsed).To(Equal(int64(10)))
			g.Expect(value.PersistenceLatencyAverageTotal).To(Equal(1.0))
			g.Expect(value.PersistenceLatencyMax).To(Equal(int64(1)))
		}
	}

}

// generateServiceSummary generates the required number of service summaries
func generateServiceSummary(serviceName, serviceType string, nodes int) []config.ServiceSummary {
	var (
		services = make([]config.ServiceSummary, 0)
	)

	for i := 1; i <= nodes; i++ {
		services = append(services, config.ServiceSummary{
			NodeID:                         fmt.Sprintf("%d", i),
			ServiceName:                    serviceName,
			StorageEnabledCount:            int32(nodes),
			ServiceType:                    serviceType,
			MemberCount:                    int32(nodes),
			StorageEnabled:                 true,
			PersistenceActiveSpaceUsed:     10,
			PersistenceLatencyAverageTotal: 1.0,
			PersistenceLatencyMax:          int64(i),
		})
	}

	return services
}

// generateHTTPSessionSummary generates the required number of http summaries
func generateHTTPSessionSummary(applicationID string, nodes int) []config.HTTPSessionSummary {
	var (
		sessions = make([]config.HTTPSessionSummary, 0)
	)

	for i := 1; i <= nodes; i++ {
		sessions = append(sessions, config.HTTPSessionSummary{
			AppID:               applicationID,
			NodeID:              fmt.Sprintf("%d", i),
			SessionCacheName:    "test",
			SessionTimeout:      30,
			SessionAverageSize:  100,
			ReapedSessionsTotal: 10,
			AverageReapDuration: 100,
			SessionUpdates:      1,
		})
	}

	return sessions
}
