/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"errors"
	"go.uber.org/zap"
	"strings"
)

const HTTP = "http"
const CreateSnapshot = "create snapshot"
const RemoveSnapshot = "remove snapshot"
const RemoveArchivedSnapshot = "remove archived snapshot"
const RecoverSnapshot = "recover snapshot"
const ArchiveSnapshot = "archive snapshot"
const RetrieveSnapshot = "retrieve snapshot"

const SuspendService = "suspend service"
const ResumeService = "resume service"
const StopService = "stop"
const StartService = "start"
const ShutdownService = "shutdown"

const JfrTypeRole = "role"
const JfrTypeNode = "node"
const JfrTypeCluster = "cluster"

const StopJFR = "jfrStop"
const DumpJFR = "jfrDump"
const CheckJFR = "jfrCheck"
const GetJFRs = "jfrCheck"

const WithSync = "with-sync"
const NoBacklog = "no-backlog"

var (
	DebugEnabled           bool
	IgnoreInvalidCerts     bool
	RequestTimeout         int32
	Logger                 *zap.Logger
	UnableToFindClusterMsg string
	ReadPassStdin          bool
)

// Fetcher interface contains the methods to get data for the CLI from various implementations
type Fetcher interface {
	GetURL() string
	GetType() string
	IsWebLogicServer() bool
	GetUsername() string

	// GetClusterDetailsJSON returns cluster details in raw json
	GetClusterDetailsJSON() ([]byte, error)

	// GetMemberDetailsJSON returns members details in raw json. verbose indicates to
	// retrieve all fields rather than selected fields
	GetMemberDetailsJSON(verbose bool) ([]byte, error)

	// GetSingleMemberDetailsJSON returns a single members details in raw json
	GetSingleMemberDetailsJSON(nodeID string) ([]byte, error)

	// GetExtendedMemberInfoJSON returns a single members extended info
	GetExtendedMemberInfoJSON(result []byte, nodeID string, tokens []string) ([][]byte, error)

	// GetServiceDetailsJSON returns member details in raw json
	GetServiceDetailsJSON() ([]byte, error)

	// GetStorageDetailsJSON returns member storage details in raw json
	GetStorageDetailsJSON() ([]byte, error)

	// GetExecutorsJSON returns executor details in raw json
	GetExecutorsJSON() ([]byte, error)

	// GetSingleServiceDetailsJSON returns a single service details in raw json
	GetSingleServiceDetailsJSON(serviceName string) ([]byte, error)

	// GetScheduledDistributionsJSON returns scheduled distributions for a service
	GetScheduledDistributionsJSON(serviceName string) ([]byte, error)

	// GetServicePartitionsJSON returns partition information for a service
	GetServicePartitionsJSON(serviceName string) ([]byte, error)

	// GetServiceMembersDetailsJSON returns all the service member details for a service
	GetServiceMembersDetailsJSON(serviceName string) ([]byte, error)

	// GetCachesSummaryJSON returns caches summary json for a service
	GetCachesSummaryJSON(serviceName string) ([]byte, error)

	// GetCachesSummaryJSONAllServices returns summary caches details for all services
	GetCachesSummaryJSONAllServices() ([]byte, error)

	// GetProxySummaryJSON returns proxy server summary
	GetProxySummaryJSON() ([]byte, error)

	// GetProxyConnectionsJSON returns the proxy connections for the specified service and node
	GetProxyConnectionsJSON(serviceName, nodeID string) ([]byte, error)

	// GetThreadDump retrieves a thread dump from a member
	GetThreadDump(memberID string) ([]byte, error)

	// ShutdownMember shuts down a member
	ShutdownMember(memberID string) ([]byte, error)

	// GetEnvironment returns the environment for a member
	GetEnvironment(memberID string) ([]byte, error)

	// SetMemberAttribute sets the given attribute for a member
	SetMemberAttribute(memberID, attribute string, value interface{}) ([]byte, error)

	// SetReporterAttribute sets the given attribute for a reporter member
	SetReporterAttribute(memberID, attribute string, value interface{}) ([]byte, error)

	// SetManagementAttribute sets the given management attribute for a cluster
	SetManagementAttribute(attribute string, value interface{}) ([]byte, error)

	// SetCacheAttribute sets the given attribute for a cache
	SetCacheAttribute(memberID, serviceName, cacheName, tier, attribute string, value interface{}) ([]byte, error)

	// SetServiceAttribute sets the given attribute for a service
	SetServiceAttribute(memberID, serviceName, attribute string, value interface{}) ([]byte, error)

	// DumpClusterHeap instructs the cluster to dump the cluster heap for the role
	DumpClusterHeap(role string) ([]byte, error)

	// ConfigureTracing instructs the cluster to configure tracing for the role or all members
	ConfigureTracing(role string, tracingRatio float32) ([]byte, error)

	// LogClusterState instructs the cluster to log cluster state for the role
	LogClusterState(role string) ([]byte, error)

	// GetCacheMembers retrieves cache member details
	GetCacheMembers(serviceName, cacheName string) ([]byte, error)

	// GetPersistenceCoordinator retrieves persistence coordinator details
	GetPersistenceCoordinator(serviceName string) ([]byte, error)

	// GetMemberOSJson returns the OS information for the member
	GetMemberOSJson(memberID string) ([]byte, error)

	// GetMembersHealth returns the health for the members in the cluster
	GetMembersHealth() ([]byte, error)

	// GetManagementJSON returns the management information
	GetManagementJSON() ([]byte, error)

	// GetReportersJSON returns reporters in raw json
	GetReportersJSON() ([]byte, error)

	// GetReporterJSON returns reporter for a node in raw json
	GetReporterJSON(nodeID string) ([]byte, error)

	// StartReporter starts the reporter on a member
	StartReporter(nodeID string) error

	// StopReporter stops the reporter on a member
	StopReporter(nodeID string) error

	// GetElasticDataDetails retrieves elastic data details for the type of flash or ram
	GetElasticDataDetails(journalType string) ([]byte, error)

	// InvokeSnapshotOperation invokes a snapshot operation against a service
	InvokeSnapshotOperation(serviceName, snapshotName, operation string, archived bool) ([]byte, error)

	// InvokeServiceOperation invokes a service operation such as suspend or resume
	InvokeServiceOperation(serviceName, operation string) ([]byte, error)

	// InvokeServiceMemberOperation invokes a service operation such as start, stop, shutdown against a node
	InvokeServiceMemberOperation(serviceName, nodeID, operation string) ([]byte, error)

	// GetArchivedSnapshots retrieves the list of archives snapshots
	GetArchivedSnapshots(serviceName string) ([]byte, error)

	// GetHTTPSessionDetailsJSON returns Coherence*Web Http session details in raw json
	GetHTTPSessionDetailsJSON() ([]byte, error)

	// StartJFR starts a JFR. type is "role", "cluster" or "node" and target is the role or node
	StartJFR(jfrName, directory, jfrType, target string, duration int32) ([]byte, error)

	// StopJFR stops a JFR. type is "cluster" or "node" and target is the node id if type "node"
	StopJFR(jfrName, jfrType, target string) ([]byte, error)

	// DumpJFR dumps a JFR. type is "cluster" or "node" and target is the node id if type "node"
	DumpJFR(jfrName, jfrType, target, filename string) ([]byte, error)

	// CheckJFR checks a JFR. type is "cluster" or "node" and target is the node id if type "node"
	CheckJFR(jfrName, jfrType, target string) ([]byte, error)

	// GetFederationStatistics returns federation statistics for a service and type
	GetFederationStatistics(serviceName, federationType string) ([]byte, error)

	// GetFederationDetails returns federation statistics for a service and type and participant
	GetFederationDetails(serviceName, federationType, nodeID, participant string) ([]byte, error)

	// InvokeFederationOperation invokes a federation operation against a service and participant
	InvokeFederationOperation(serviceName, command, participant, mode string) ([]byte, error)
}

// GetFetcherOrError returns a fetcher and error
func GetFetcherOrError(connectionType, url, username, clusterName string) (Fetcher, error) {
	if connectionType == HTTP {
		return HTTPFetcher{URL: url, ConnectionType: connectionType, WebLogicServer: IsWebLogicServer(url),
			Username: username, ClusterName: clusterName}, nil
	}

	return nil, errors.New("invalid connection type of " + connectionType)
}

// IsWebLogicServer returns true if the connection is of WebLogic Server format
func IsWebLogicServer(url string) bool {
	if strings.Contains(url, "/management/coherence/") && strings.Contains(url, "clusters") {
		return true
	}
	return false
}
