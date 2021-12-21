/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package config

//
// Various structure definitions for text command output
//

// Cluster is a structure to display cluster details for 'describe cluster'
type Cluster struct {
	ClusterName           string `json:"clusterName"`
	ClusterSize           int    `json:"clusterSize"`
	LicenseMode           string `json:"licenseMode"`
	Version               string `json:"version"`
	Running               bool   `json:"running"`
	MembersDepartureCount int    `json:"membersDepartureCount"`
}

type Members struct {
	Members []Member `json:"items"`
}

// Executor contains individual executor information
type Executor struct {
	Name                 string `json:"name"`
	ID                   string `json:"id"`
	Description          string `json:"description"`
	MemberID             string `json:"memberId"`
	State                string `json:"state"`
	Location             string `json:"location"`
	TasksInProgressCount int64  `json:"tasksInProgressCount"`
	TasksCompletedCount  int64  `json:"tasksCompletedCount"`
	TasksRejectedCount   int64  `json:"tasksRejectedCount"`
	MemberCount          int32  `json:"memberCount"`
	TraceLogging         bool   `json:"traceLogging"`
}

type Executors struct {
	Executors []Executor `json:"items"`
}

// Member describes an individual members output
type Member struct {
	NodeID               string  `json:"nodeId"`
	UnicastAddress       string  `json:"unicastAddress"`
	UnicastPort          int32   `json:"unicastPort"`
	RoleName             string  `json:"roleName"`
	MemberName           string  `json:"memberName"`
	MachineName          string  `json:"machineName"`
	RackName             string  `json:"rackName"`
	SiteName             string  `json:"siteName"`
	ProcessName          string  `json:"processName"`
	MemoryMaxMB          int32   `json:"memoryMaxMB"`
	MemoryAvailableMB    int32   `json:"memoryAvailableMB"`
	ReceiverSuccessRate  float32 `json:"receiverSuccessRate"`
	PublisherSuccessRate float32 `json:"publisherSuccessRate"`
}

// ProxiesSummary provides a summary of individual proxy servers
type ProxiesSummary struct {
	Proxies []ProxySummary `json:"items"`
}

// ProxySummary describes proxy server summary
type ProxySummary struct {
	HostIP                   string `json:"hostIP"`
	NodeID                   string `json:"nodeId"`
	ServiceName              string `json:"name"`
	Type                     string `json:"type"`
	Protocol                 string `json:"protocol"`
	ConnectionCount          int64  `json:"connectionCount"` // proxy service specific
	OutgoingMessageBacklog   int64  `json:"outgoingMessageBacklog"`
	OutgoingByteBacklog      int64  `json:"outgoingByteBacklog"`
	TotalBytesReceived       int64  `json:"totalBytesReceived"`
	TotalBytesSent           int64  `json:"totalBytesSent"`
	TotalMessagesReceived    int64  `json:"totalMessagesReceived"`
	TotalMessagesSent        int64  `json:"totalMessagesSent"`
	UnAuthConnectionAttempts int64  `json:"unauthorizedConnectionAttempts"`

	HTTPServerType    string `json:"httpServerType"` // http service specific
	TotalRequestCount int64  `json:"totalRequestCount"`
	TotalErrorCount   int64  `json:"totalErrorCount"`
	ResponseCount1xx  int64  `json:"responseCount1xx"`
	ResponseCount2xx  int64  `json:"responseCount2xx"`
	ResponseCount3xx  int64  `json:"responseCount3xx"`
	ResponseCount4xx  int64  `json:"responseCount4xx"`
	ResponseCount5xx  int64  `json:"responseCount5xx"`
}

type HTTPSessionSummaries struct {
	HTTPSessions []HTTPSessionSummary `json:"items"`
}

// HTTPSessionSummary provides a summary of Coherence*Web Sessions
type HTTPSessionSummary struct {
	NodeID              string `json:"nodeId"`
	AppID               string `json:"appId"`
	Type                string `json:"type"`
	SessionCacheName    string `json:"sessionCacheName"`
	OverflowCacheName   string `json:"overflowCacheName"`
	SessionTimeout      int32  `json:"sessionTimeout"`
	SessionAverageSize  int32  `json:"sessionAverageSize"`
	ReapedSessionsTotal int64  `json:"reapedSessionsTotal"`
	AverageReapDuration int64  `json:"averageReapDuration"`
	LastReapDuration    int64  `json:"lastReapDuration"`
	SessionUpdates      int64  `json:"sessionUpdates"`

	// calculated
	SessionAverageTotal int64
	TotalReapDuration   int64
	MemberCount         int32
}

type ServicesSummaries struct {
	Services []ServiceSummary `json:"items"`
}

// ServiceSummary provides a summary of individual services
type ServiceSummary struct {
	NodeID               string `json:"nodeId"`
	ServiceName          string `json:"name"`
	ServiceType          string `json:"type"`
	MemberCount          int32  `json:"memberCount"`
	StorageEnabledCount  int32  `json:"storageEnabledCount"`
	StatusHA             string `json:"statusHA"`
	PartitionsAll        int32  `json:"partitionsAll"`
	PartitionsEndangered int32  `json:"partitionsEndangered"`
	PartitionsVulnerable int32  `json:"partitionsVulnerable"`
	PartitionsUnbalanced int32  `json:"partitionsUnbalanced"`
	StorageEnabled       bool   `json:"storageEnabled"`

	// persistence related
	PersistenceMode                   string  `json:"persistenceMode"`
	PersistenceActiveSpaceUsed        int64   `json:"persistenceActiveSpaceUsed"`
	PersistenceLatencyMax             int64   `json:"persistenceLatencyMax"`
	PersistenceLatencyAverage         float64 `json:"persistenceLatencyAverage"`
	PersistenceSnapshotSpaceAvailable int64   `json:"persistenceSnapshotSpaceAvailable"`

	// derived
	PersistenceLatencyAverageTotal float64
	Snapshots                      []string
	OperationStatus                string
	Idle                           bool
}

type StatsSummary struct {
	Count   int64   `json:"count"`
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Sum     float64 `json:"sum"`
}

type FederationSummaries struct {
	Services []FederationSummary `json:"items"`
}

type FederationSummary struct {
	ServiceName                             string       `json:"serviceName"`
	ParticipantName                         string       `json:"participantName"`
	State                                   []string     `json:"state"`
	Member                                  []string     `json:"member"`
	TotalMsgSent                            StatsSummary `json:"totalMsgSent"`
	TotalBytesSent                          StatsSummary `json:"totalBytesSent"`
	TotalRecordsSent                        StatsSummary `json:"totalRecordsSent"`
	MsgNetworkRoundTripTimePercentileMillis StatsSummary `json:"msgNetworkRoundTripTimePercentileMillis"`
	MsgApplyTimePercentileMillis            StatsSummary `json:"msgApplyTimePercentileMillis"`
	RecordBacklogDelayTimePercentileMillis  StatsSummary `json:"recordBacklogDelayTimePercentileMillis"`
	ReplicateAllPercentComplete             StatsSummary `json:"replicateAllPercentComplete"`
	ReplicateAllTotalTime                   StatsSummary `json:"replicateAllTotalTime"`
	CurrentBandwidth                        StatsSummary `json:"currentBandwidth"`

	TotalMsgReceived       StatsSummary `json:"totalMsgReceived"` // incoming
	TotalBytesReceived     StatsSummary `json:"totalBytesReceived"`
	TotalRecordsReceived   StatsSummary `json:"totalRecordsReceived"`
	CurrentConnectionCount StatsSummary `json:"currentConnectionCount"`
}

// ServiceMemberDetails provides service members details
type ServiceMemberDetails struct {
	Services []ServiceMemberDetail `json:"items"`
}

type ServiceMemberDetail struct {
	NodeID                 string  `json:"nodeId"`
	ThreadCount            int32   `json:"threadCount"`
	ThreadCountMin         int32   `json:"threadCountMin"`
	ThreadCountMax         int32   `json:"threadCountMax"`
	ThreadIdleCount        int32   `json:"threadIdleCount"`
	TaskCount              int32   `json:"taskCount"`
	TaskCountBacklog       int32   `json:"taskCountBacklog"`
	OwnedPartitionsPrimary int32   `json:"ownedPartitionsPrimary"`
	OwnedPartitionsBackup  int32   `json:"ownedPartitionsBackup"`
	RequestAverageDuration float32 `json:"requestAverageDuration"`
	TaskAverageDuration    float32 `json:"taskAverageDuration"`
}

// CacheSummaries provides cache summary details
type CacheSummaries struct {
	Caches []CacheSummaryDetail `json:"items"`
}

type CacheSummaryDetail struct {
	ServiceName  string `json:"service"`
	CacheName    string `json:"name"`
	CacheSize    int32  `json:"size"`
	UnitsBytes   int64  `json:"unitsBytes"`
	TotalPuts    int64  `json:"totalPuts"`
	TotalGets    int64  `json:"totalGets"`
	TotalRemoves int64  `json:"removeCount"`
	CacheHits    int64  `json:"cacheHits"`
	CacheMisses  int64  `json:"cacheMisses"`
}

// CacheDetails provides cache details
type CacheDetails struct {
	Details []CacheDetail `json:"items"`
}

type CacheDetail struct {
	NodeID        string `json:"nodeId"`
	Tier          string `json:"tier"`
	UnitsBytes    int64  `json:"unitsBytes"`
	CacheSize     int32  `json:"size"`
	TotalPuts     int64  `json:"totalPuts"`
	TotalGets     int64  `json:"totalGets"`
	TotalRemoves  int64  `json:"removeCount"`
	CacheHits     int64  `json:"cacheHits"`
	CacheMisses   int64  `json:"cacheMisses"`
	StoreReads    int64  `json:"storeReads"`
	StoreWrites   int64  `json:"storeWrites"`
	StoreFailures int64  `json:"storeFailures"`

	LocksGranted                   int64    `json:"locksGranted"`
	LocksPending                   int64    `json:"locksPending"`
	ListenerRegistrations          int64    `json:"listenerRegistrations"`
	MaxQueryDurationMillis         int64    `json:"maxQueryDurationMillis"`
	MaxQueryDescription            string   `json:"maxQueryDescription"`
	NonOptimizedQueryAverageMillis float64  `json:"nonOptimizedQueryAverageMillis"`
	OptimizedQueryAverageMillis    float64  `json:"optimizedQueryAverageMillis"`
	IndexTotalUnits                int64    `json:"indexTotalUnits"`
	IndexingTotalMillis            int64    `json:"indexingTotalMillis"`
	IndexInfo                      []string `json:"indexInfo"`
}

// GenericDetails are a slice of generic Json structures
type GenericDetails struct {
	Details []interface{} `json:"items"`
}

type PersistenceCoordinator struct {
	Idle              bool     `json:"idle"`
	OperationStatus   string   `json:"operationStatus"`
	Snapshots         []string `json:"snapshots"`
	CoordinatorNodeID int32    `json:"coordinatorId"`
}

// Machine provides machine details
type Machine struct {
	MachineName             string  `json:"operationStatus"`
	AvailableProcessors     int32   `json:"availableProcessors"`
	SystemLoadAverage       float32 `json:"systemLoadAverage"` // check first
	SystemCPULoad           float32 `json:"systemCpuLoad"`     // check second
	TotalPhysicalMemorySize int64   `json:"totalPhysicalMemorySize"`
	FreePhysicalMemorySize  int64   `json:"freePhysicalMemorySize"`
	Arch                    string  `json:"arch"`
	Name                    string  `json:"name"`
	Version                 string  `json:"version"`
}

// Reporters provides reporter details
type Reporters struct {
	Reporters []Reporter `json:"items"`
}

type Reporter struct {
	NodeID           string  `json:"nodeId"`
	State            string  `json:"state"`
	OutputPath       string  `json:"outputPath"`
	ConfigFile       string  `json:"configFile"`
	LastReport       string  `json:"lastReport"`
	LastRunMillis    int32   `json:"runLastMillis"`
	CurrentBatch     int32   `json:"currentBatch"`
	RunAverageMillis float64 `json:"runAverageMillis"`
	AutoStart        bool    `json:"autoStart"`
}

// ElasticDataValues provides elastic data details
type ElasticDataValues struct {
	ElasticData []ElasticData `json:"items"`
}

type ElasticData struct {
	NodeID                     string  `json:"nodeId"`
	Name                       string  `json:"name"`
	Type                       string  `json:"type"`
	FileCount                  int32   `json:"fileCount"`
	MaxJournalFilesNumber      int32   `json:"maxJournalFilesNumber"`
	CurrentCollectorLoadFactor float32 `json:"currentCollectorLoadFactor"`
	HighestLoadFactor          float32 `json:"highestLoadFactor"`
	CompactionCount            int64   `json:"compactionCount"`
	ExhaustiveCompactionCount  int64   `json:"exhaustiveCompactionCount"`
	MaxFileSize                int64   `json:"maxFileSize"`
	TotalDataSize              int64   `json:"totalDataSize"`
}

// Links describe any links returned via HTTP
type Links struct {
	Links []Link `json:"links"`
}

type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type ItemLinks struct {
	Links []Links `json:"items"`
}

// Snapshots describe snapshots for services
type Snapshots struct {
	ServiceName string   `json:"serviceName"`
	Snapshots   []string `json:"snapshots"`
}

type Archives struct {
	Snapshots []string `json:"archives"`
}

// StatusValues is a JFR status result
type StatusValues struct {
	Status []string `json:"status"`
}

// SingleStatusValue is a single JFR status result
type SingleStatusValue struct {
	Status string `json:"status"`
}
