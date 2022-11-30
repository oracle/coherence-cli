/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package common

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/cmd"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/oracle/coherence-cli/test/test_utils"
	"os"
	"regexp"
	"strings"
	"testing"
)

const configArg = "--config"
const addedCluster = "Added cluster"
const version1221 = "12.2.1"
const version1411 = "14.1.1"
const configYaml = "config.yaml"
const nodeID = "NODE ID"
const jsonPathServices = "jsonpath=$.services"
const doesNotExist = "does not exist"

//
// These tests run each of the CLI commands and inspects the output to ensure that the
// there is expected output
//

// RunTestClusterCommands tests add/remove/get/describe cluster commands
func RunTestClusterCommands(t *testing.T) {
	context := test_utils.GetTestContext()

	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// set the debug to true
	test_utils.EnsureCommandContains(g, t, cliCmd, "on", configArg, file, "set", "debug", "on")

	// set default format to "m"
	test_utils.EnsureCommandContains(g, t, cliCmd, "Bytes format is now set to m", configArg, file, "set", "bytes-format", "m")

	// get bytes format
	test_utils.EnsureCommandContains(g, t, cliCmd, "Current bytes format: m", configArg, file, "get", "bytes-format")

	// clear default format
	test_utils.EnsureCommandContains(g, t, cliCmd, "Default bytes format has been cleared", configArg, file, "clear", "bytes-format")

	// set default heap to 512m
	test_utils.EnsureCommandContains(g, t, cliCmd, "Default heap is now set to 512m", configArg, file, "set", "default-heap", "512m")

	// get default heap
	test_utils.EnsureCommandContains(g, t, cliCmd, "Current default heap: 512m", configArg, file, "get", "default-heap")

	// clear default heap
	test_utils.EnsureCommandContains(g, t, cliCmd, "Default heap has been cleared", configArg, file, "clear", "default-heap")

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")

	// try adding a cluster with invalid URL
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "Unable to query cluster connection", configArg, file, "add", "cluster",
		context.ClusterName, "-u", "http://badurl:123123/")

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// get clusters should return 1 cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "get", "clusters")

	// use wide option to check cluster connection
	test_utils.EnsureCommandContains(g, t, cliCmd, "RUNNING", configArg, file, "get", "clusters", "-o", "wide")

	// should NOT be able to add new cluster with the same name
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "A connection for cluster named", configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// generate a host port to try the automatic generation
	shortUrl := strings.ReplaceAll(strings.ReplaceAll(context.Url,
		"http://", ""), "/management/coherence/cluster", "")

	// should be able to add a second cluster using the shorthand URL
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		"cluster2", "-u", shortUrl)

	// get clusters should return 2 clusters
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "get", "clusters")
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster2", configArg, file, "get", "clusters")

	// try to delete a cluster that doesn't exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find cluster with", configArg, file, "remove", "cluster",
		"not-here")

	// test describe cluster command with verbose
	test_utils.EnsureCommandContainsAll(g, t, cliCmd,
		"CLUSTER,MEMBERS,ROLE,Running:,License Mode,SERVICES,REPORTERS,PROXY,HTTP,FLIGHT RECORDINGS",
		configArg, file, "describe", "cluster", "cluster1", "-v")

	// test describe cluster command with verbose and wide
	test_utils.EnsureCommandContainsAll(g, t, cliCmd,
		"CLUSTER,MEMBERS,ROLE,Running:,License Mode,SERVICES,REPORTERS,PROXY,HTTP,UNBALANCED",
		configArg, file, "describe", "cluster", "cluster1", "-v", "-o", "wide")

	// test describe cluster without verbose
	test_utils.EnsureCommandContainsAll(g, t, cliCmd,
		"CLUSTER,MEMBERS,ROLE,Running:,License Mode,SERVICES,PROXY,HTTP,PERSISTENCE",
		configArg, file, "describe", "cluster", "cluster1")

	// test describe cluster without verbose
	test_utils.EnsureCommandContains(g, t, cliCmd, "[2]",
		configArg, file, "describe", "cluster", "cluster1", "-o", "jsonpath=$.cluster.clusterSize")

	// test JsonPATH
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1", configArg, file, "get", "clusters",
		"-o", "jsonpath=$..['name']")

	// test JsonPATH
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1", configArg, file, "get", "clusters",
		"-o", "json")

	// reset output format to default of TABLE
	cmd.OutputFormat = constants.TABLE

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster2", configArg, file, "remove", "cluster", "cluster2")

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")
}

// RunTestNSLookupCommands tests nslookup commands
func RunTestNSLookupCommands(t *testing.T) {
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")

	// should be able to return cluster name
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1", configArg, file, "nslookup",
		"-q", "Cluster/name")

	// should be 2 foreign clusters
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster3,cluster2", configArg, file, "nslookup",
		"-q", "NameService/string/Cluster/foreign")

	// should be able to see the cluster info
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "127.0.0.1,ServiceJoined", configArg, file,
		"nslookup", "-q", "Cluster/info")
}

// RunTestDiscoverClustersCommands tests discover clusters commands
func RunTestDiscoverClustersCommands(t *testing.T) {
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// sleep to ensure the clusters are ready
	test_utils.Sleep(30)

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")

	// should be able to discover 3 new clusters
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster1,cluster2,cluster3", configArg, file,
		"discover", "clusters", "-y")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1", configArg, file, "remove", "cluster", "cluster1")
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster2", configArg, file, "remove", "cluster", "cluster2")
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster3", configArg, file, "remove", "cluster", "cluster3")
}

// RunTestMemberCommands tests various member commands
func RunTestMemberCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// test default output format
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, nodeID, configArg, file, "get", "members",
		"-c", context.ClusterName)

	// test wide output format
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PUBLISHER,RECEIVER", configArg, file, "get", "members",
		"-o", "wide", "-c", context.ClusterName)

	// test get tracing
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "TRACING ENABLED,SAMPLING RATIO", configArg, file, "get", "tracing",
		"-c", context.ClusterName)

	// set the current context and ensure the command still succeeds
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1\n", configArg, file, "set", "context", "cluster1")

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, nodeID, configArg, file, "get", "members")

	// should receive values
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, nodeID, configArg, file, "get", "members",
		"-r", "OracleCoherenceCliTestingRestServer")

	// should not receive values
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "", configArg, file, "get", "members",
		"-r", "UnknownRole")

	// describe a member
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MEMBER DETAILS,Node Id", configArg, file, "describe", "member", "1")

	// describe a member with extended
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MEMBER DETAILS,Node Id,EXTENDED",
		configArg, file, "describe", "member", "1", "-X", "g1OldGen")

	// describe a member with thread dump
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MEMBER DETAILS,Node Id,THREAD DUMP", configArg, file, "describe", "member", "1", "-D")

	// test jsonpath
	test_utils.EnsureCommandContains(g, t, cliCmd, "n/a", configArg, file, "get", "members",
		"-o", "jsonpath=$.items[0].rackName", "-c", "cluster1")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

func getVersion(restUrl string) (string, error) {
	var (
		err     error
		version []byte
	)
	version, err = test_utils.IssueGetRequest(restUrl + "/version")
	if err != nil {
		return "", err
	}
	return string(version), nil
}

// RunTestServicesCommands tests various services commands
func RunTestServicesCommands(t *testing.T) {
	var (
		g             = NewGomegaWithT(t)
		err           error
		versionString string
		context       = test_utils.GetTestContext()
		restUrl       = context.RestUrl
	)

	versionString, err = getVersion(restUrl)
	g.Expect(err).To(BeNil())

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE NAME", configArg, file, "get", "services",
		"-c", context.ClusterName)

	// set the current context and ensure the command still succeeds
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1\n", configArg, file, "set", "context", "cluster1")

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE NAME,DistributedCache", configArg, file, "get", "services")

	// test service list does not contain a proxy when we request DistributedCache service type
	test_utils.EnsureCommandNotContains(g, t, cliCmd, "Proxy", configArg, file, "get", "services",
		"-t", "DistributedCache")

	// test service list does not contain a DistributedCache when we request Proxy service type
	test_utils.EnsureCommandNotContains(g, t, cliCmd, "DistributedCache", configArg, file, "get", "services",
		"-t", "Proxy")

	// test get service-members
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "NODE ID,PartitionedCache", configArg, file, "get", "service-members", "PartitionedCache")

	// test get services-storage
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "NODES,AVG PARTITION", configArg, file, "get", "service-storage")

	if strings.Contains(versionString, version1221) || strings.Contains(versionString, version1411) {
		t.Log("workaround Coh Bug in test as version is " + versionString)
	} else {
		// test we can set threadCountMin on a service
		test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
			"set", "service", "PartitionedCache", "-y", "-a", "threadCountMin", "-v", "10")

		// sleep for jmx refresh timeout to be passed
		test_utils.Sleep(10)

		test_utils.EnsureCommandContains(g, t, cliCmd, "[10,10]", configArg, file,
			"get", "services", "-o", "jsonpath=$.items[?(@.name == 'PartitionedCache')]..['threadCountMin']")

		// test we can set threadCountMin on a member
		test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
			"set", "service", "PartitionedCache", "-y", "-a", "threadCountMin", "-v", "5", "-n", "1")

		// sleep for jmx refresh timeout to be passed
		test_utils.Sleep(5)

		test_utils.EnsureCommandContains(g, t, cliCmd, "5", configArg, file,
			"get", "services", "-o", "jsonpath=$.items[?(@.name == 'PartitionedCache')]..['threadCountMin']")
	}

	// test jsonpath
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"eventInterceptorInfo\",PartitionedCache",
		configArg, file, "describe", "service", "PartitionedCache", "-c", context.ClusterName,
		"-o", jsonPathServices)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

// RunTestServiceOperations tests various services operations
func RunTestServiceOperations(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		err     error
		context = test_utils.GetTestContext()
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// test can get services
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE NAME,PartitionedCache", configArg,
		file, "get", "services", "-t", "DistributedCache", "-c", context.ClusterName)

	// test suspend service
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "suspend", "service",
		"PartitionedCache", "-y", "-c", context.ClusterName)

	test_utils.Sleep(5)

	// test service is suspended
	test_utils.EnsureCommandContains(g, t, cliCmd, "Suspended", configArg, file, "describe", "service",
		"PartitionedCache", "-c", context.ClusterName)

	test_utils.Sleep(5)

	// test resume service
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "resume", "service",
		"PartitionedCache", "-y", "-c", context.ClusterName)

	// NOTE: The following is disabled because of an intermittent
	// test to ensure that services are now resumed
	// test_utils.Sleep(15)
	//
	// test service is suspended
	//test_utils.EnsureCommandNotContains(g, t, cliCmd, "Suspended", configArg, file, "describe", "service",
	//	"PartitionedCache", "-c", context.ClusterName)

	// test stop service on invalid member
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "no node with node id", configArg, file, "stop", "service",
		"PartitionedCache", "-n", "100", "-y", "-c", context.ClusterName)
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "invalid value for node id", configArg, file, "stop", "service",
		"PartitionedCache", "-n", "100x", "-y", "-c", context.ClusterName)

	// test stopping/ starting and shutdown of service
	// we can't actually check any logs, etc so just confirm that the commands execute
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "stop", "service",
		"PartitionedCache", "-y", "-n", "2", "-c", context.ClusterName)

	test_utils.Sleep(10)

	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "start", "service",
		"PartitionedCache", "-y", "-n", "2", "-c", context.ClusterName)

	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "shutdown", "service",
		"PartitionedCache", "-y", "-n", "2", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

// RunTestProxyCommands tests various services commands
func RunTestProxyCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// set the current context and ensure the commands still succeeds
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1\n", configArg, file, "set", "context", "cluster1")

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE NAME,CONNECTIONS", configArg, file, "get", "proxies",
		"-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PROXY SERVICE DETAILS,PROXY MEMBER DETAILS,PROXY CONNECTIONS", configArg, file,
		"describe", "proxy", "Proxy", "-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PROXY SERVICE DETAILS,PROXY MEMBER DETAILS,PROXY CONNECTIONS", configArg, file,
		"describe", "proxy", "Proxy")

	// test json
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"ManagementHttpProxy\",\"name\"", configArg, file,
		"describe", "proxy", "Proxy", "-o", "json")

	// test jsonpath
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"eventInterceptorInfo\"", configArg, file,
		"describe", "proxy", "Proxy", "-o", jsonPathServices)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

// RunTestManagementCommands tests the get/set management command
func RunTestManagementCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// test we can get management
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Refresh Policy,refresh-ahead,Expiry Delay,1000",
		configArg, file, "get", "management", "-c", context.ClusterName)

	// test we cannot set invalid attributes or values
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "attribute name rubbish is invalid",
		configArg, file, "set", "management", "-a", "rubbish", "-v", "value", "-c", context.ClusterName)

	// test we can set the expiry delay to 10000
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted,
		configArg, file, "set", "management", "-a", "expiryDelay", "-v", "10000", "-y", "-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Expiry Delay,10000",
		configArg, file, "get", "management", "-c", context.ClusterName)

	// test we cannot set refreshPolicy to an invalid value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "attribute value for refreshPolicy",
		configArg, file, "set", "management", "-a", "refreshPolicy", "-v", "value", "-c", context.ClusterName)

	// test we can set refreshPolicy
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted,
		configArg, file, "set", "management", "-a", "refreshPolicy", "-v", "refresh-behind",
		"-y", "-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Refresh Policy,refresh-behind,Expiry Delay,10000",
		configArg, file, "get", "management", "-c", context.ClusterName)

	// test json
	test_utils.EnsureCommandContains(g, t, cliCmd, "\"refreshTime\"",
		configArg, file, "get", "management", "-c", context.ClusterName, "-o", "json")

	// test jsonpath
	test_utils.EnsureCommandContains(g, t, cliCmd, "[10000]",
		configArg, file, "get", "management", "-c", context.ClusterName, "-o", "jsonpath=$.expiryDelay")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

// RunTestSetMemberCommands tests various set member commands
func RunTestSetMemberCommands(t *testing.T) {
	var (
		g             = NewGomegaWithT(t)
		context       = test_utils.GetTestContext()
		err           error
		versionString string
		restUrl       = context.RestUrl
	)

	// Skip if the cluster version is 14.1.1.X as there is an issue with set log level which is being investigated
	versionString, err = getVersion(restUrl)

	g.Expect(err).To(BeNil())
	if strings.Contains(versionString, version1411) || strings.Contains(versionString, version1221) ||
		strings.Contains(versionString, "20.06") {
		t.Log("Ignoring test as version is " + versionString)
		return
	}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// set the debug to true
	test_utils.EnsureCommandContains(g, t, cliCmd, "on", configArg, file, "set", "debug", "on")

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// Set management expiry to 100 ms
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "set", "management",
		"-a", "expiryDelay", "-v", "100", "-y", "-c", "cluster1")

	// should be able to set the log level to 1 for all members
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "set", "member",
		"all", "-a", "loggingLevel", "-v", "1", "-y", "-c", "cluster1")

	// sleep tp ensure management refresh is reached
	test_utils.Sleep(30)

	// query the log level
	test_utils.EnsureCommandContains(g, t, cliCmd, "\"loggingLevel\":1", configArg, file, "get", "members",
		"-o", "json", "-c", "cluster1")

	// should be able to set the log level to 6 for member 1
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "set", "member",
		"1", "-a", "loggingLevel", "-v", "6", "-y", "-c", "cluster1")

	test_utils.Sleep(30)

	// query the log level - should have log level 9 and 6
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"loggingLevel\":1,\"loggingLevel\":6", configArg,
		file, "get", "members", "-o", "json", "-c", "cluster1")

	// should be able to set the log level to 3 for all members
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "set", "member",
		"all", "-a", "loggingLevel", "-v", "3", "-y", "-c", "cluster1")

	test_utils.Sleep(30)

	// query the log level - should have log level 3
	test_utils.EnsureCommandContains(g, t, cliCmd, "\"loggingLevel\":3", configArg, file, "get", "members",
		"-o", "json", "-c", "cluster1")

	loggingFormat := "{date}/{uptime} {product} {version} <{level}> (thread={thread}, member={member}):- {text}"
	// set the loggingFormat for all members
	test_utils.EnsureCommandContains(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "set", "member",
		"all", "-a", "loggingFormat", "-v", loggingFormat, "-y", "-c", "cluster1")

	test_utils.Sleep(15)

	// query the logging format
	test_utils.EnsureCommandContains(g, t, cliCmd, loggingFormat, configArg, file, "get", "members",
		"-o", "json", "-c", "cluster1")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}

// RunTestMachinesCommands tests various machines commands
func RunTestMachinesCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to get machines
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MACHINE,PROCESSORS,TOTAL MEMORY", configArg, file, "get", "machines",
		"-c", context.ClusterName)

	// retrieve the output so we can issue describe
	output := test_utils.GetCommandOutput(t, cliCmd, configArg, file, "get", "machines", "-c", context.ClusterName)
	g.Expect(output).To(Not(BeNil()))

	var re1 = regexp.MustCompile("^((.|\\n)*)server1")
	var re2 = regexp.MustCompile(" ((.|\\n)*)")
	output = re1.ReplaceAllString(output, "")
	output = re2.ReplaceAllString(output, "")
	machine := "server1" + output

	// should be able to describe the machine
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MACHINE DETAILS,Machine Name,"+machine, configArg, file,
		"describe", "machine", machine, "-c", context.ClusterName)

	// should be able to get output via json
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "committedVirtualMemorySize", configArg, file, "get", "machines",
		"-c", context.ClusterName, "-o", "json")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestReporterCommands tests various reporter commands
func RunTestReporterCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
	)

	const outputPath = "/tmp/output/path"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// get the reporters
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "CONFIG FILE,OUTPUT PATH,Stopped", configArg, file,
		"get", "reporters", "-c", context.ClusterName)

	// get the reporters
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Node Id,Auto Start", configArg, file,
		"describe", "reporter", "1", "-c", context.ClusterName)

	// can't stop a reporter that is already stopped
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "already stopped", configArg, file,
		"stop", "reporter", "1", "-c", context.ClusterName, "-y")

	// start a reporter
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "started", configArg, file,
		"start", "reporter", "1", "-c", context.ClusterName, "-y")

	// can't start a reporter that is already started
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "already started", configArg, file,
		"start", "reporter", "1", "-c", context.ClusterName, "-y")

	// get the reporters
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Started", configArg, file,
		"get", "reporters", "-c", context.ClusterName)

	// get the reporters using json
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"lastExecuteTime\"", configArg, file,
		"get", "reporters", "-c", context.ClusterName, "-o", "json")

	// set the reporter interval to 122 seconds
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"set", "reporter", "-c", context.ClusterName, "-a", "intervalSeconds", "-v", "122", "-y", "all")

	test_utils.Sleep(10)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "122", configArg, file,
		"get", "reporters", "-c", context.ClusterName)

	// set the output path to /tmp/output/path
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"set", "reporter", "-c", context.ClusterName, "-a", "outputPath", "-v", outputPath, "-y", "1")

	test_utils.Sleep(10)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, outputPath, configArg, file,
		"get", "reporters", "-c", context.ClusterName)

	// try setting an invalid attribute value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "is invalid", configArg, file,
		"set", "reporter", "-c", context.ClusterName, "-a", "outputPathNotValid", "-v", outputPath, "-y", "1")

	// get the reporters using jsonpath
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"lastExecuteTime\"", configArg, file,
		"get", "reporters", "-c", context.ClusterName, "-o", "jsonpath=$.items[0]")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestThreadDumpsCommands tests thread dump commands
func RunTestThreadDumpsCommands(t *testing.T) {
	var (
		g              = NewGomegaWithT(t)
		context        = test_utils.GetTestContext()
		allThreadDumps = "All thread dumps completed"
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	// create a temp directory
	threadDumpDir := test_utils.CreateTempDirectory("temp")
	err = os.Mkdir(threadDumpDir, 0755)
	g.Expect(err).To(BeNil())
	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to create a thread dump
	test_utils.EnsureCommandContains(g, t, cliCmd, allThreadDumps, configArg, file, "retrieve", "thread-dumps",
		"all", "-O", threadDumpDir, "-c", context.ClusterName, "-y", "-D", "5")

	// assert that the thread dumps exist
	for i := 1; i <= 2; i++ {
		for d := 1; d <= 5; d++ {
			g.Expect(test_utils.FileExistsInDirectory(threadDumpDir, cmd.GetFileName(fmt.Sprintf("%d", i), int32(d))))
		}
	}

	// should be able to create a thread dump for a single node
	test_utils.EnsureCommandContains(g, t, cliCmd, allThreadDumps, configArg, file, "retrieve", "thread-dumps",
		"1", "-O", threadDumpDir, "-c", context.ClusterName, "-y", "-D", "5")

	// test invalid role value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "invalid-role", configArg, file, "retrieve", "thread-dumps",
		"-r", "invalid-role", "-O", threadDumpDir, "-c", context.ClusterName, "-y")

	// should be able to create a thread dump for a role
	test_utils.EnsureCommandContains(g, t, cliCmd, allThreadDumps, configArg, file, "retrieve", "thread-dumps",
		"-r", "OracleCoherenceCliTestingRestServer", "-O", threadDumpDir, "-c", context.ClusterName, "-y", "-D", "5")

	// re-initialize to overwrite invalid role
	cliCmd = cmd.Initialize(nil)

	// test invalid node id values
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "invalid value", configArg, file, "retrieve", "thread-dumps",
		"1,X", "-O", threadDumpDir, "-c", context.ClusterName, "-y")

	// test node 3 which does not exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "no node with node", configArg, file, "retrieve", "thread-dumps",
		"3", "-O", threadDumpDir, "-c", context.ClusterName, "-y")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestExecutorCommands runs basic executor tests
func RunTestExecutorCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
		result  []byte
		restUrl = context.RestUrl
	)

	result, err = test_utils.IssueGetRequest(restUrl + "/executorPresent")
	g.Expect(err).To(BeNil())

	if string(result) != "true" {
		t.Log("Ignoring Executor test as it is not running in docker image")
		return
	}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to get executors
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "NAME,REJECTED,IN PROGRESS", configArg, file,
		"get", "executors", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestJFRCommands tests various jfr commands
func RunTestJFRCommands(t *testing.T) {
	var (
		g             = NewGomegaWithT(t)
		context       = test_utils.GetTestContext()
		err           error
		versionString string
		restUrl       = context.RestUrl
	)

	const jfrName = "test-1"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	// Skip if the cluster version is 14.1.1.X or 12.2.1.4.X as we are running the test with JDK 11
	// as JFR was only in open JDK since 11
	versionString, err = getVersion(restUrl)
	g.Expect(err).To(BeNil())

	if strings.Contains(versionString, version1411) || strings.Contains(versionString, version1221) ||
		strings.Contains(versionString, "20.06") {
		t.Log("Ignoring test as version is " + versionString)
		return
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to get JFR's
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "No available recordings", configArg, file,
		"get", "jfrs", "-c", context.ClusterName)

	// should be able to create a JFR
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Started recording 1", configArg, file,
		"start", "jfr", jfrName, "-O", "/tmp", "-D", "25", "-y", "-c", context.ClusterName)

	// should be able to describe the JFR
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Recording,duration", configArg, file,
		"describe", "jfr", jfrName, "-c", context.ClusterName)

	// should be able to dump the JFR
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Dumped recording,tim.jfr", configArg, file,
		"dump", "jfr", jfrName, "-f", "/tmp/tim.jfr", "-c", context.ClusterName)

	// should be able to see the JFRS running
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "name=test-1 duration=25s", configArg, file,
		"get", "jfrs", "-c", context.ClusterName)

	// sleep long enough for the JFR to finish
	test_utils.Sleep(30)

	// should be able to get JFR's
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "No available recordings", configArg, file,
		"get", "jfrs", "-c", context.ClusterName)

	// try to start a JFR on an invalid node
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, doesNotExist, configArg, file,
		"start", "jfr", jfrName, "-O", "/tmp", "-n", "100", "-y", "-c", context.ClusterName)

	// reset node id
	cmd.NodeID = ""

	// should be able to create a JFR for the role OracleCoherenceCliTestingRestServer
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Started recording 2", configArg, file,
		"start", "jfr", jfrName, "-O", "/tmp", "-D", "20", "-r", "OracleCoherenceCliTestingRestServer", "-y",
		"-c", context.ClusterName)

	// should be able to dump the recording
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Dumped recording \"test-1\",1-tim.jfr,2-tim.jfr",
		configArg, file, "dump", "jfr", jfrName, "-y", "-c", context.ClusterName)

	// should be able to describe the recording
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "running",
		configArg, file, "describe", "jfr", jfrName, "-c", context.ClusterName)

	// should be able to stop the JFR
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Stopped",
		configArg, file, "stop", "jfr", jfrName, "-y", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestDumpClusterHeapCommands tests the dump cluster-heap command
func RunTestDumpClusterHeapCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
	)

	const clusterHeap = "cluster-heap"
	const dump = "dump"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to dump cluster heap for all members
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Operation completed", configArg, file,
		dump, clusterHeap, "-y", "-c", context.ClusterName)

	// should be able to dump cluster heap for a specific role
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "Operation completed", configArg, file,
		dump, clusterHeap, "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServer")

	// an invalid role should cause an error
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find members with role name", configArg, file,
		dump, clusterHeap, "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServerWrong")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestConfigureTracingCommands tests the configure tracing command
// note these commands will not actual starting tracing as the required libraries
// are not included. Messages should be in the logs indicating these deps are missing
func RunTestConfigureTracingCommands(t *testing.T) {
	var (
		g             = NewGomegaWithT(t)
		context       = test_utils.GetTestContext()
		err           error
		versionString string
		restUrl       = context.RestUrl
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	versionString, err = getVersion(restUrl)
	g.Expect(err).To(BeNil())

	if strings.Contains(versionString, "12.2.1.") {
		t.Log("Ignoring as tracing not supported in version " + versionString)
		return
	}

	cliCmd := cmd.Initialize(nil)

	cmd.Config.Debug = true

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to configure tracing for all members
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "completed", configArg, file,
		"configure", "tracing", "-y", "-c", context.ClusterName, "-t", "1.0")

	// should be able to configure tracing heap for a specific role
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "completed", configArg, file,
		"configure", "tracing", "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServer", "-t", "-1.0")

	// an invalid tracing ratio should raise an error
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "tracing ratio must be either", configArg, file,
		"configure", "tracing", "-y", "-c", context.ClusterName, "-t", "-2.0")

	// an invalid role should cause an error
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find members with role name", configArg, file,
		"configure", "tracing", "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServerWrong", "-t", "-1.0")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestLogClusterStateCommands tests the log cluster-state command
func RunTestLogClusterStateCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
	)

	const clusterState = "cluster-state"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to dump cluster heap for all members
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "completed", configArg, file,
		"log", clusterState, "-y", "-c", context.ClusterName)

	// should be able to dump cluster heap for a specific role
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "completed", configArg, file,
		"log", clusterState, "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServer")

	// an invalid role should cause an error
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find members with role name", configArg, file,
		"log", clusterState, "-y", "-c", context.ClusterName, "-r", "OracleCoherenceCliTestingRestServerWrong")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestHttpSessionCommands tests various elastic data commands
func RunTestHttpSessionCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
		edition []byte
		restUrl = context.RestUrl
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// only continue if the cluster is Grid Edition
	edition, err = test_utils.IssueGetRequest(restUrl + "/edition")
	g.Expect(err).To(BeNil())
	editionString := string(edition)
	if editionString != "GE" {
		t.Log("Ignoring test as edition is " + editionString)
		return
	}

	// Register mock MBeans
	_, err = test_utils.IssueGetRequest(restUrl + "/registerMBeans")
	g.Expect(err).To(BeNil())

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to get http session details
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "HttpSessionManager,application1,testcache", configArg, file,
		"get", "http-sessions", "-c", context.ClusterName)

	// should be able to describe
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "HttpSessionManager,application1,testcache", configArg, file,
		"describe", "http-session", "application1", "-c", context.ClusterName)

	// trying describing an application that does not exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find", configArg, file,
		"describe", "http-session", "application1123", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestElasticDataCommands tests various elastic data commands
func RunTestElasticDataCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
		edition []byte
		restUrl = context.RestUrl
	)

	const elasticData = "elastic-data"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// only continue if the cluster is Grid Edition
	edition, err = test_utils.IssueGetRequest(restUrl + "/edition")
	g.Expect(err).To(BeNil())
	editionString := string(edition)
	if editionString != "GE" {
		t.Log("Ignoring test as edition is " + editionString)
		return
	}

	// populate flash and Ram
	_, err = test_utils.IssueGetRequest(restUrl + "/populateFlash")
	g.Expect(err).To(BeNil())
	_, err = test_utils.IssueGetRequest(restUrl + "/populateRam")
	g.Expect(err).To(BeNil())

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// get the elastic data
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "FlashJournalRM,RamJournalRM", configArg, file, "get", elasticData,
		"-c", context.ClusterName)

	// describe Ram Journal
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "RAM JOURNAL DETAILS", configArg, file, "describe", elasticData,
		"RamJournalRM", "-c", context.ClusterName)

	// describe Flash Journal
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "FLASH JOURNAL DETAILS", configArg, file, "describe", elasticData,
		"FlashJournalRM", "-c", context.ClusterName)

	// describe invalid type
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, cmd.ElasticDataMessage, configArg, file, "describe", elasticData,
		"not-valid", "-c", context.ClusterName)

	// compact Flash Journal
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "compact", elasticData,
		"FlashJournalRM", "-y", "-c", context.ClusterName)

	// compact Ram Journal
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file, "compact", elasticData,
		"RamJournalRM", "-y", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestCachesCommands tests various caches commands
func RunTestCachesCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	const cacheName = "cache-1"

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// add some data
	restUrl := context.RestUrl

	_, err = test_utils.IssueGetRequest(restUrl + "/populate")
	g.Expect(err).To(BeNil())

	// populate the cache store
	_, err = test_utils.IssueGetRequest(restUrl + "/populateCacheStore")
	g.Expect(err).To(BeNil())

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE,COUNT,SIZE,cache-1,cache-2", configArg, file,
		"get", "caches", "-c", context.ClusterName)

	// test write-behind
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedCacheWriteBehind,QUEUE SIZE,cache-store-1", configArg, file,
		"get", "cache-stores", "cache-store-1", "-s", "PartitionedCacheWriteBehind", "-c", context.ClusterName)

	// test wide output
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "TOTAL PUTS", configArg, file,
		"get", "caches", "-c", context.ClusterName, "-o", "wide")

	// test describe cache without service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "\"service\" not set", configArg, file, "describe", "cache",
		cacheName, "-c", context.ClusterName)

	// test describe cache with invalid service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find service", configArg, file, "describe", "cache",
		cacheName, "-s", "invalid-service", "-c", context.ClusterName)

	// test describe cache without correct service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "no cache named cache-1", configArg, file, "describe", "cache",
		cacheName, "-s", "PartitionedCache2", "-c", context.ClusterName)

	// test describe cache-1
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "NODE ID,TIER,SIZE,LOCKS GRANTED,INDEX DETAILS", configArg, file, "describe", "cache",
		cacheName, "-s", "PartitionedCache", "-c", context.ClusterName)

	// test describe cache-2
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "NODE ID,TIER,SIZE,LOCKS GRANTED,INDEX DETAILS", configArg, file, "describe", "cache",
		"cache-2", "-s", "PartitionedCache", "-c", context.ClusterName)

	// test set cache errors - invalid tier
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, cmd.InvalidTierMsg, configArg, file,
		"set", "cache", cacheName, "-a", "expiryDelay", "-v", "30", "-s", "PartitionedCache", "-y",
		"-c", context.ClusterName, "-t", "invalid")

	// test set cache errors - invalid float value
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "invalid", configArg, file,
		"set", "cache", cacheName, "-a", "expiryDelay", "-v", "30.fhfhfh", "-s", "PartitionedCache", "-y",
		"-c", context.ClusterName, "-t", "back")

	// test set expiry to 30 seconds
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"set", "cache", cacheName, "-a", "expiryDelay", "-v", "30", "-s", "PartitionedCache", "-y",
		"-c", context.ClusterName)

	test_utils.Sleep(15)

	// test get caches to return 30
	test_utils.EnsureCommandContains(g, t, cliCmd, "30", configArg, file,
		"get", "caches", "-o", "jsonpath=$.items[?(@.name == 'cache-1')]..['name','expiryDelay','nodeId']",
		"-c", context.ClusterName)

	// test set expiry to 60 seconds
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"set", "cache", cacheName, "-a", "expiryDelay", "-v", "60", "-y", "-s", "PartitionedCache",
		"-c", context.ClusterName)

	test_utils.Sleep(10)

	// test get caches to return 60
	test_utils.EnsureCommandContains(g, t, cliCmd, "60", configArg, file,
		"get", "caches", "-o", "jsonpath=$.items[?(@.name == 'cache-1')]..['name','expiryDelay','nodeId']",
		"-c", context.ClusterName)

	// test set invalid attribute
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "attribute name expiryDelayd is invalid", configArg, file,
		"set", "cache", cacheName, "-a", "expiryDelayd", "-v", "60", "-y", "-s", "PartitionedCache",
		"-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestPersistenceCommands tests various caches commands
func RunTestPersistenceCommands(t *testing.T) {
	var (
		g            = NewGomegaWithT(t)
		serviceName  = "PartitionedCache"
		snapshotName = "snapshot-1"
		context      = test_utils.GetTestContext()
		services     []string
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// add some data
	restUrl := context.RestUrl

	_, err = test_utils.IssueGetRequest(restUrl + "/populate")
	g.Expect(err).To(BeNil())

	// check persistent services
	services, err = cmd.GetPersistenceServices(dataFetcher)
	g.Expect(err).To(BeNil())
	g.Expect(utils.SliceContains(services, "PartitionedCache")).To(Equal(true))
	g.Expect(utils.SliceContains(services, "PartitionedCache2")).To(Equal(true))
	g.Expect(utils.SliceContains(services, "PartitionedCache2333")).To(Equal(false))

	// create a snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",create snapshot", configArg,
		file, "create", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	// get the snapshots
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, snapshotName+","+serviceName, configArg,
		file, "get", "snapshots", "-c", context.ClusterName)

	// ensure get persistence works
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PERSISTENCE MODE,active,LATENCY", configArg,
		file, "get", "persistence", "-c", context.ClusterName)

	// archive the snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",archive snapshot", configArg,
		file, "archive", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	// get the archived snapshots
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, snapshotName+","+serviceName+",ARCHIVED SNAPSHOT NAME",
		configArg, file, "get", "snapshots", "-a", "-c", context.ClusterName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	cmd.ArchivedSnapshots = false
	// remove the local snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",remove snapshot", configArg,
		file, "remove", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	// ensure the snapshot is gone
	test_utils.EnsureCommandNotContains(g, t, cliCmd, snapshotName+","+serviceName, configArg,
		file, "get", "snapshots", "-c", context.ClusterName)

	// retrieve the local snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",retrieve snapshot", configArg,
		file, "retrieve", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	cmd.ArchivedSnapshots = false
	// recover the snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",recover snapshot", configArg,
		file, "recover", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	// remove the archived snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",remove archived snapshot", configArg,
		file, "remove", "snapshot", snapshotName, "-a", "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	// ensure the archived snapshot is gone
	test_utils.EnsureCommandNotContains(g, t, cliCmd, snapshotName+","+serviceName, configArg,
		file, "get", "snapshots", "-a", "-c", context.ClusterName)

	// test describe service which will display persistence information
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, snapshotName+",Total Active Space,PERSISTENCE COORDINATOR",
		configArg, file, "describe", "service", serviceName, "-c", context.ClusterName)

	// try to archive a snapshot that does not exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, doesNotExist, configArg,
		file, "archive", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	cmd.ArchivedSnapshots = false
	// remove the local snapshot
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "invoked,"+snapshotName+",remove snapshot", configArg,
		file, "remove", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	err = test_utils.WaitForIdlePersistence(120, dataFetcher, serviceName)
	g.Expect(err).To(BeNil())

	cmd.ArchivedSnapshots = false
	// try to recover a snapshot that doesn't exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, doesNotExist, configArg,
		file, "recover", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	// try to retrieve a snapshot that doesn't exist
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, doesNotExist, configArg,
		file, "retrieve", "snapshot", snapshotName, "-c", context.ClusterName, "-y", "-s", serviceName)

	// test json output
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SafeBerkeleyDBEnvironment", configArg,
		file, "get", "persistence", "-c", context.ClusterName, "-o", "json")

	// test jsonpath output
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SafeBerkeleyDBEnvironment", configArg,
		file, "get", "persistence", "-c", context.ClusterName, "-o", "jsonpath=$.items[*].persistenceEnvironment")

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestHealthCommands tests various health commands
func RunTestHealthCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		context = test_utils.GetTestContext()
		err     error
	)

	// ignore test if health is not enabled in this version
	if !isHealthEnabled(context.RestUrl) {
		return
	}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MEMBER HEALTH,STARTED,LIVE,READY,SAFE", configArg, file, "get", "health",
		"-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "MEMBERS,STARTED,LIVE,READY,SAFE", configArg, file, "get", "health",
		"-S", "-c", context.ClusterName)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestHttpProxyCommands tests various http proxy commands
func RunTestHttpProxyCommands(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVER TYPE,REQUESTS", configArg, file, "get", "http-servers",
		"-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "HTTP SERVER SERVICE DETAILS,HTTP SERVER MEMBER DETAILS", configArg, file,
		"describe", "http-server", "ManagementHttpProxy", "-c", context.ClusterName)

	// set the current context and ensure the commands still succeeds
	test_utils.EnsureCommandContains(g, t, cliCmd, "cluster1\n", configArg, file, "set", "context", "cluster1")

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVER TYPE,REQUESTS", configArg, file, "get", "http-servers",
		"-c", context.ClusterName)

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "HTTP SERVER SERVICE DETAILS,HTTP SERVER MEMBER DETAILS", configArg, file,
		"describe", "http-server", "ManagementHttpProxy")

	// test json output
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"services\":,ManagementHttpProxy", configArg, file,
		"describe", "http-server", "ManagementHttpProxy", "-o", "json")

	// test jsonpath output
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"eventInterceptorInfo\"", configArg, file,
		"describe", "http-server", "ManagementHttpProxy", "-o", jsonPathServices)

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestClusterGetClusterRequest tests get cluster http request
func RunTestClusterGetClusterRequest(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()
	var cluster = config.Cluster{}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// add a new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// get cluster details
	data, err := dataFetcher.GetClusterDetailsJSON()
	g.Expect(err).To(BeNil())

	jsonData := string(data)
	g.Expect(jsonData).To(ContainSubstring("clusterName\":\"" + context.ClusterName + "\""))

	err = json.Unmarshal(data, &cluster)
	g.Expect(err).To(BeNil())
	g.Expect(cluster).To(Not(Equal(nil)))
	g.Expect(cluster.ClusterSize).To(Equal(context.ExpectedServers))
	g.Expect(cluster.ClusterName).To(Equal(context.ClusterName))
	g.Expect(cluster.Running).To(Equal(true))

	// remove the cluster connection
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestClusterGetMembersRequest tests members http request
func RunTestClusterGetMembersRequest(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()
	var members = config.Members{}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// add a new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// get cluster details
	data, err := dataFetcher.GetMemberDetailsJSON(true)
	g.Expect(err).To(BeNil())

	jsonData := string(data)
	g.Expect(jsonData).To(ContainSubstring("nodeId\":\"1\""))

	err = json.Unmarshal(data, &members)
	g.Expect(err).To(BeNil())
	g.Expect(len(members.Members)).To(Equal(context.ExpectedServers))

	for _, value := range members.Members {
		g.Expect(value.NodeID == "1" || value.NodeID == "2").To(Equal(true))
		g.Expect(value.MemberName).Should(ContainSubstring("member"))
		g.Expect(value.SiteName).To(Equal("Site1"))
	}

	// remove the cluster connection
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestClusterServicesRequest tests services http request
func RunTestClusterServicesRequest(t *testing.T) {
	var (
		g        = NewGomegaWithT(t)
		context  = test_utils.GetTestContext()
		services = config.ServicesSummaries{}
		err      error
		found    bool
		data     []byte
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// add a new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// get cluster details
	data, err = dataFetcher.GetServiceDetailsJSON()
	g.Expect(err).To(BeNil())

	jsonData := string(data)
	g.Expect(jsonData).To(ContainSubstring("nodeId\":\"1\""))

	err = json.Unmarshal(data, &services)
	g.Expect(err).To(BeNil())
	g.Expect(len(services.Services) > 0).To(Equal(true))

	for _, value := range services.Services {
		g.Expect(value.ServiceName).To(Not(BeNil()))
		g.Expect(value.ServiceType).To(Not(BeNil()))
		g.Expect(value.StatusHA).To(Not(BeNil()))
	}

	// validate ServiceExists works
	found, err = cmd.ServiceExists(dataFetcher, "PartitionedCache")
	g.Expect(err).To(BeNil())
	g.Expect(found).To(Equal(true))

	found, err = cmd.ServiceExists(dataFetcher, "PartitionedCache222222")
	g.Expect(err).To(BeNil())
	g.Expect(found).To(Equal(false))
	fmt.Println("HELLO")

	// remove the cluster connection
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestCachesRequests tests caches http request
func RunTestCachesRequests(t *testing.T) {
	g := NewGomegaWithT(t)
	context := test_utils.GetTestContext()
	var caches = config.CacheDetails{}

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// add a new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// add some data
	restUrl := context.RestUrl

	_, err = test_utils.IssueGetRequest(restUrl + "/populate")
	g.Expect(err).To(BeNil())

	data, err := dataFetcher.GetCacheMembers("PartitionedCache", "cache-1")
	g.Expect(err).To(BeNil())

	jsonData := string(data)
	g.Expect(jsonData).To(ContainSubstring("nodeId\":\"1\""))

	err = json.Unmarshal(data, &caches)
	g.Expect(err).To(BeNil())
	g.Expect(len(caches.Details) > 0).To(Equal(true))

	for _, value := range caches.Details {
		g.Expect(value).To(Not(BeNil()))
		g.Expect(value.CacheSize).To(Not(BeNil()))
		g.Expect(value.TotalPuts).To(Not(BeNil()))
	}

	// remove the cluster connection
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")
}

// RunTestFederationCommands tests federation commands
func RunTestFederationCommands(t *testing.T) {
	var (
		context = test_utils.GetTestContext()
		restUrl = context.RestUrl
		g       = NewGomegaWithT(t)
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// set the debug to true
	test_utils.EnsureCommandContains(g, t, cliCmd, "on", configArg, file, "set", "debug", "on")

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")

	// should be able to add new cluster cluster1
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// should be able to add new cluster cluster2 on 30001
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		"cluster2", "-u", strings.ReplaceAll(context.Url, ":30000", ":30001"))

	// get clusters should return 1 cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "get", "clusters")

	// get members should only return 1 member being member1 and not member2
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "member1", configArg, file, "get", "members",
		"-c", context.ClusterName)
	test_utils.EnsureCommandNotContains(g, t, cliCmd, "member2", configArg, file, "get", "members",
		"-c", context.ClusterName)

	// ensure federation settles down
	test_utils.Sleep(10)

	// populate data
	_, err = test_utils.IssueGetRequest(restUrl + "/populateFederation")
	g.Expect(err).To(BeNil())

	// Get federation and ensure it is idle or paused
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2", configArg, file,
		"get", "federation", "destinations", "-c", context.ClusterName)

	// Test JSON
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"cluster2\",destinations", configArg, file,
		"get", "federation", "all", "-c", context.ClusterName, "-o", "json")

	// Test JSONPAth
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "\"currentBandwidth\",FederatedService", configArg, file,
		"get", "federation", "all", "-c", context.ClusterName, "-o", "jsonpath=$.destinations")

	// reset output format to default of TABLE
	cmd.OutputFormat = constants.TABLE

	// Start federation
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"start", "federation", "FederatedService", "-y", "-c", context.ClusterName)

	// Start federation
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"start", "federation", "FederatedService", "-y", "-c", "cluster2")

	test_utils.Sleep(20)

	// Get federation and ensure it is IDLE as data should have been sent by now
	// note we have to reset the output format
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2,IDLE", configArg, file,
		"get", "federation", "destinations", "-c", context.ClusterName)

	// ensure there is data in the destination cluster
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "SERVICE,CACHE,SIZE,federated-1,federated-2,federated-3",
		configArg, file, "get", "caches", "-c", "cluster2")

	// Pause federation
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"pause", "federation", "FederatedService", "-y", "-c", context.ClusterName)

	test_utils.Sleep(5)

	// Get federation and ensure it is paused
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2,PAUSED", configArg, file,
		"get", "federation", "destinations", "-c", context.ClusterName)

	// Stop federation
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"stop", "federation", "FederatedService", "-y", "-c", context.ClusterName)

	test_utils.Sleep(5)

	// Get federation and ensure it is stopped
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2,STOPPED", configArg, file,
		"get", "federation", "destinations", "-c", context.ClusterName)

	// Start federation
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"start", "federation", "FederatedService", "-y", "-c", context.ClusterName)

	test_utils.Sleep(10)

	// Get federation and ensure it is IDLE as data should have been sent by now
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2,IDLE", configArg, file,
		"get", "federation", "destinations", "-c", context.ClusterName)

	// validate we cannot replicate all to an unknown participant
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "unable to find participant", configArg, file,
		"replicate", "all", "FederatedService", "-p", "participant", "-y", "-c", context.ClusterName)

	// replicate all
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, cmd.OperationCompleted, configArg, file,
		"replicate", "all", "FederatedService", "-p", "cluster2", "-y", "-c", context.ClusterName)

	test_utils.Sleep(30)

	// get wide output and check for 100.00%
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "cluster2,REPLICATE,100.00%", configArg, file,
		"get", "federation", "destinations", "-o", "wide", "-c", context.ClusterName)

	// test describe cluster
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "AVG BACKLOG DELAY,AVG APPLY,cluster2", configArg, file,
		"describe", "federation", "FederatedService", "-p", "cluster2", "-T", "destinations", "-o", "wide", "-c", context.ClusterName)

	// these commands are only available in the latest coherence versions - 14.1.1.2206.x and above
	if isHealthEnabled(restUrl) {
		// test reset federation-stats
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "federation-stats",
			"FederatedService", "-p", "cluster2", "-T", "outgoing", "-y", "-n", "1", "-c", context.ClusterName)

		// test reset federation-stats
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "federation-stats",
			"FederatedService", "-p", "cluster2", "-T", "outgoing", "-y", "-c", context.ClusterName)
	}

	// remove the cluster entry
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")
}

// RunTestTopicsCommands tests federation commands
func RunTestTopicsCommands(t *testing.T) {
	const noTopics = "there are no topics for service"
	var (
		context = test_utils.GetTestContext()
		restUrl = context.RestUrl
		g       = NewGomegaWithT(t)
	)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// add a new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	dataFetcher := GetDataFetcher(g, context.ClusterName)

	// get cluster details
	_, err = dataFetcher.GetServiceDetailsJSON()
	g.Expect(err).To(BeNil())

	// Start Topics
	_, err = test_utils.IssueGetRequest(restUrl + "/startTopics")
	g.Expect(err).To(BeNil())

	// sleep to ensure the clusters are ready
	test_utils.Sleep(20)

	// validate get topics
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "TOPIC,SUBSCRIBERS,PartitionedTopic,private-messages,public-messages", configArg, file,
		"get", "topics", "-c", context.ClusterName)

	// ensure no topics returned for invalid service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "there are no topics for service invalid-service", configArg, file,
		"get", "topics", "-s", "invalid-service", "-c", context.ClusterName)

	// describe a topic
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "TOPIC DETAILS,MEMBERS,SUBSCRIBERS,SUBSCRIBER GROUPS,private-messages,17", configArg, file,
		"describe", "topic", "private-messages", "-s", "PartitionedTopic", "-c", context.ClusterName)

	// describe a non-existent topic
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, "a topic named", configArg, file,
		"describe", "topic", "private-messagesxxx", "-s", "PartitionedTopic", "-c", context.ClusterName)

	// get topic-members
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedTopic,17,private-messages,PUBLISHED,NODE ID", configArg, file,
		"get", "topic-members", "private-messages", "-s", "PartitionedTopic", "-c", context.ClusterName)

	// test get topic-members with invalid service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, noTopics, configArg, file,
		"get", "topic-members", "private-messages", "-s", "PartitionedTopicx", "-c", context.ClusterName)

	// get member-channels
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedTopic,17,private-messages,PUBLISHED,MEAN,PagedPosition", configArg, file,
		"get", "member-channels", "private-messages", "-s", "PartitionedTopic", "-n", "1", "-c", context.ClusterName)

	// get subscribers
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedTopic,17,public-messages,SUBSCRIBER ID,NODE ID,SUBSCRIBER GROUP", configArg, file,
		"get", "subscribers", "public-messages", "-s", "PartitionedTopic", "-c", context.ClusterName)

	// test get subscribers with invalid service
	test_utils.EnsureCommandErrorContains(g, t, cliCmd, noTopics, configArg, file,
		"get", "subscribers", "public-messages", "-s", "PartitionedTopicxx", "-c", context.ClusterName)

	// get subscriber channels - need to use the datafetcher to find a valid subscriber
	topicsResult, err := dataFetcher.GetTopicsSubscribersJSON("PartitionedTopic", "public-messages")
	g.Expect(err).To(Not(HaveOccurred()))

	topicsSummary := config.TopicsSubscriberDetails{}
	err = json.Unmarshal(topicsResult, &topicsSummary)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(len(topicsSummary.Details)).To(BeNumerically(">", 0))

	// retrieve the subscriber
	ID := topicsSummary.Details[0].NodeID
	subscriber := topicsSummary.Details[0].ID

	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedTopic,17,public-messages,EMPTY,LAST COMMIT,HEAD", configArg, file,
		"get", "subscriber-channels", "public-messages", "-s", "PartitionedTopic", "-n", fmt.Sprintf("%v", ID),
		"-S", fmt.Sprintf("%v", subscriber), "-c", context.ClusterName)

	// test sub-grp-channels
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PartitionedTopic,17,public-messages,MEMBER,MEAN,OWNING SUB", configArg, file,
		"get", "sub-grp-channels", "public-messages", "-s", "PartitionedTopic", "-n", "1", "-G", "1", "-c", context.ClusterName)

	// remove the cluster entry
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1", "-y")

	// get clusters should return nothing
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "clusters")
}

// GetDataFetcher returns a Fetcher instance or throws an assertion if not found
func GetDataFetcher(g *WithT, clusterName string) fetcher.Fetcher {
	found, connection := cmd.GetClusterConnection(clusterName)

	g.Expect(found).To(Equal(true))
	dataFetcher, err := fetcher.GetFetcherOrError(connection.ConnectionType, connection.ConnectionURL,
		"", clusterName)
	g.Expect(err).To(BeNil())
	return dataFetcher
}

// isHealthEnabled returns true if health API is enabled, which is 14.1.1.2206 and 22.06+
func isHealthEnabled(restUrl string) bool {
	result, err := test_utils.IssueGetRequest(restUrl + "/healthPresent")
	return err != nil && string(result) == "true"
}

// RunTestProfileCommands tests profile commands
func RunTestProfileCommands(t *testing.T) {
	g := NewGomegaWithT(t)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	// set the debug to true
	test_utils.EnsureCommandContains(g, t, cliCmd, "on", configArg, file, "set", "debug", "on")

	// get profiles
	test_utils.EnsureCommandContains(g, t, cliCmd, "", configArg, file, "get", "profiles")

	// set a profile
	test_utils.EnsureCommandContains(g, t, cliCmd, "profile profile1", configArg, file, "set", "profile",
		"profile1", "-v", "-Dtim.property=value", "-y")

	// get profiles
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PROFILE,VALUE,profile1,tim.property", configArg, file, "get", "profiles")

	// set a second profile
	test_utils.EnsureCommandContains(g, t, cliCmd, "profile profile2", configArg, file, "set", "profile",
		"profile2", "-v", "-Dnew.property=value", "-y")

	// get profiles
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PROFILE,VALUE,profile1,tim.property,profile2,new.property", configArg, file, "get", "profiles")

	// update the profile
	test_utils.EnsureCommandContains(g, t, cliCmd, "profile profile2", configArg, file, "set", "profile",
		"profile2", "-v", "-Dupdated", "-y")

	// get profiles
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "PROFILE,VALUE,profile1,tim.property,profile2,updated", configArg, file, "get", "profiles")

	// remove the profile
	test_utils.EnsureCommandContainsAll(g, t, cliCmd, "profile profile2 was removed", configArg, file, "remove", "profile",
		"profile2", "-y")
}

// RunTestResetCommands tests various reset commands
func RunTestResetCommands(t *testing.T) {
	var (
		g       = NewGomegaWithT(t)
		err     error
		edition []byte
		context = test_utils.GetTestContext()
		restUrl = context.RestUrl
		result  []byte
	)

	// only continue if the cluster is Grid Edition
	edition, err = test_utils.IssueGetRequest(restUrl + "/edition")
	g.Expect(err).To(BeNil())
	editionString := string(edition)

	file, err := test_utils.CreateNewConfigYaml(configYaml)
	if err != nil {
		t.Fatal(err)
	}

	fullSupport := isHealthEnabled(context.RestUrl)

	test_utils.CleanupConfigFileAfterTest(t, file)

	cliCmd := cmd.Initialize(nil)

	_, err = test_utils.IssueGetRequest(restUrl + "/populate")
	g.Expect(err).To(BeNil())

	// should be able to add new cluster
	test_utils.EnsureCommandContains(g, t, cliCmd, addedCluster, configArg, file, "add", "cluster",
		context.ClusterName, "-u", context.Url)

	// ========= CACHES ================

	// test node specific
	test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "cache-stats",
		"cache-1", "-s", "PartitionedCache", "-y", "-n", "1", "-c", context.ClusterName)

	if fullSupport {
		// we can run a reset statistics against all members
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "cache-stats",
			"cache-1", "-s", "PartitionedCache", "-y", "-c", context.ClusterName)
	}

	// ========= EXECUTOR ============

	result, err = test_utils.IssueGetRequest(restUrl + "/executorPresent")
	g.Expect(err).To(BeNil())

	if string(result) == "true" {
		// executor is present
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "executor-stats",
			"coherence-concurrent-default-executor", "-y", "-c", context.ClusterName)
	}

	// ========= ELASTIC DATA ============

	if editionString == "GE" {
		_, err = test_utils.IssueGetRequest(restUrl + "/populateFlash")
		g.Expect(err).To(BeNil())
		_, err = test_utils.IssueGetRequest(restUrl + "/populateRam")
		g.Expect(err).To(BeNil())

		if fullSupport {
			test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "flashjournal-stats",
				"-c", context.ClusterName, "-y")
			test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "ramjournal-stats",
				"-c", context.ClusterName, "-y")
		}
	}

	// ========= MEMBERS ============

	test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "member-stats",
		"-y", "-n", "1", "-c", context.ClusterName)

	if fullSupport {
		// test for all members
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "member-stats",
			"-y", "-c", context.ClusterName)
	}

	// ========= REPORTERS ============

	test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "reporter-stats",
		"-y", "-n", "1", "-c", context.ClusterName)

	test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "reporter-stats",
		"-y", "-c", context.ClusterName)

	// ========= SERVICES ============

	test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "service-stats",
		"PartitionedCache", "-y", "-n", "1", "-c", context.ClusterName)

	if fullSupport {
		test_utils.EnsureCommandContains(g, t, cliCmd, "completed", configArg, file, "reset", "service-stats",
			"PartitionedCache", "-y", "-c", context.ClusterName)
	}

	// remove the cluster entries
	test_utils.EnsureCommandContains(g, t, cliCmd, context.ClusterName, configArg, file, "remove", "cluster", "cluster1")
}
