/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package standalone

import (
	"github.com/oracle/coherence-cli/test/common"
	"testing"
)

//
// Run the test suite against a standalone Coherence Cluster
//

// TestClusterCommands tests add/remove/get/describe cluster commands
func TestClusterCommands(t *testing.T) {
	common.RunTestClusterCommands(t)
}

// TestMemberCommands tests various member commands
func TestMemberCommands(t *testing.T) {
	common.RunTestMemberCommands(t)
}

// TestManagementCommands tests management commands
func TestManagementCommands(t *testing.T) {
	common.RunTestManagementCommands(t)
}

// TestSetMemberCommands tests set member commands
func TestSetMemberCommands(t *testing.T) {
	common.RunTestSetMemberCommands(t)
}

// TestServicesCommands tests various services commands
func TestServicesCommands(t *testing.T) {
	common.RunTestServicesCommands(t)
}

// TestProxyCommands tests various services commands
func TestProxyCommands(t *testing.T) {
	common.RunTestProxyCommands(t)
}

// TestHttpProxyCommands tests various http proxy commands
func TestHttpProxyCommands(t *testing.T) {
	common.RunTestHttpProxyCommands(t)
}

// TestCachesCommands tests caches commands
func TestCachesCommands(t *testing.T) {
	common.RunTestCachesCommands(t)
}

// TestExecutorCommands tests executor commands
func TestExecutorCommands(t *testing.T) {
	common.RunTestExecutorCommands(t)
}

// TestMachinesCommands tests caches commands
func TestMachinesCommands(t *testing.T) {
	common.RunTestMachinesCommands(t)
}

// TestPersistenceCommands tests persistence commands
func TestPersistenceCommands(t *testing.T) {
	common.RunTestPersistenceCommands(t)
}

// TestReporterCommands tests reporter commands
func TestReporterCommands(t *testing.T) {
	common.RunTestReporterCommands(t)
}

// TestElasticDataCommands tests elastic data commands
func TestElasticDataCommands(t *testing.T) {
	common.RunTestElasticDataCommands(t)
}

// TestHttpSessionCommands tests elastic data commands
func TestHttpSessionCommands(t *testing.T) {
	common.RunTestHttpSessionCommands(t)
}

// TestThreadDumpsCommands tests thread-dump commands
func TestThreadDumpsCommands(t *testing.T) {
	common.RunTestThreadDumpsCommands(t)
}

// TestJFRCommands tests jfr commands
func TestJFRCommands(t *testing.T) {
	common.RunTestJFRCommands(t)
}

// TestDumpClusterHeapCommands tests dump cluster heap commands
func TestDumpClusterHeapCommands(t *testing.T) {
	common.RunTestDumpClusterHeapCommands(t)
}

// TestConfigureTracingCommands tests configure tracing commands
func TestConfigureTracingCommands(t *testing.T) {
	common.RunTestConfigureTracingCommands(t)
}

// TestLogClusterStateCommands tests log cluster heap commands
func TestLogClusterStateCommands(t *testing.T) {
	common.RunTestLogClusterStateCommands(t)
}

// TestClusterGetClusterRequest tests get cluster request
func TestClusterGetClusterRequest(t *testing.T) {
	common.RunTestClusterGetClusterRequest(t)
}

// TestClusterGetClusterHttpRequest tests members request
func TestClusterGetMembersRequest(t *testing.T) {
	common.RunTestClusterGetMembersRequest(t)
}

// TestClusterServicesRequest tests services request
func TestClusterServicesRequest(t *testing.T) {
	common.RunTestClusterServicesRequest(t)
}

// TestCachesRequests tests caches request
func TestCachesRequests(t *testing.T) {
	common.RunTestCachesRequests(t)
}
