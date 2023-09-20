/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package config

//
// Various structure definitions for text command output
//

// Cluster is a structure to display cluster details for 'describe cluster'.
type Cluster struct {
	ClusterName           string `json:"clusterName"`
	ClusterSize           int    `json:"clusterSize"`
	LicenseMode           string `json:"licenseMode"`
	Version               string `json:"version"`
	Running               bool   `json:"running"`
	MembersDepartureCount int    `json:"membersDepartureCount"`
}

// Members contains an array of member objects.
type Members struct {
	Members []Member `json:"items"`
}

// NetworkStats is used to decode network stats call for a member.
type NetworkStats struct {
	ViewerStatistics []string `json:"viewerStatistics"`
}

// NetworkStatsDetails contains viewer statistics for a member.
type NetworkStatsDetails struct {
	NodeID               string  `json:"nodeId"`
	ReceiverSuccessRate  float32 `json:"receiverSuccessRate"`
	PublisherSuccessRate float32 `json:"publisherSuccessRate"`
	PauseRate            float32 `json:"pauseRate"`
	Threshold            int64   `json:"threshold"`
	Paused               bool    `json:"paused"`
	Deferring            bool    `json:"deferring"`
	OutstandingPackets   int64   `json:"outstandingPackets"`
	DeferredPackets      int64   `json:"deferredPackets"`
	ReadyPackets         int64   `json:"readyPackets"`
	LastIn               string  `json:"lastIn"`
	LastOut              string  `json:"lastOut"`
	LastSlow             string  `json:"LastSlow"`
	LastHeuristicDeath   string  `json:"lastHeuristicDeath"`
}

// Executor contains individual executor information.
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

// Executors contains multiple Executor objects.
type Executors struct {
	Executors []Executor `json:"items"`
}

// Member contains an individual members output.
type Member struct {
	NodeID                   string  `json:"nodeId"`
	UnicastAddress           string  `json:"unicastAddress"`
	UnicastPort              int32   `json:"unicastPort"`
	RoleName                 string  `json:"roleName"`
	MemberName               string  `json:"memberName"`
	MachineName              string  `json:"machineName"`
	RackName                 string  `json:"rackName"`
	SiteName                 string  `json:"siteName"`
	ProcessName              string  `json:"processName"`
	MemoryMaxMB              int32   `json:"memoryMaxMB"`
	MemoryAvailableMB        int32   `json:"memoryAvailableMB"`
	ReceiverSuccessRate      float32 `json:"receiverSuccessRate"`
	PublisherSuccessRate     float32 `json:"publisherSuccessRate"`
	TracingSamplingRatio     float32 `json:"tracingSamplingRatio"`
	StorageEnabled           bool    `json:"storageEnabled"`
	PacketDeliveryEfficiency float64 `json:"packetDeliveryEfficiency"`
	PacketsResent            int64   `json:"packetsResent"`
	PacketsSent              int64   `json:"packetsSent"`
	PacketsReceived          int64   `json:"packetsReceived"`
	SendQueueSize            int64   `json:"sendQueueSize"`
	TransportReceivedBytes   int64   `json:"transportReceivedBytes"`
	TransportSentBytes       int64   `json:"transportSentBytes"`
	WeakestChannel           int32   `json:"weakestChannel"`
}

// StorageDetails contains a summary of storage member details.
type StorageDetails struct {
	Details []StorageDetail `json:"items"`
}

// StorageDetail contains an individual storage details for a member.
type StorageDetail struct {
	NodeID                 string `json:"nodeId"`
	OwnedPartitionsPrimary int    `json:"ownedPartitionsPrimary"`
}

// ProxiesSummary contains a summary of individual proxy servers.
type ProxiesSummary struct {
	Proxies []ProxySummary `json:"items"`
}

// ProxySummary contains proxy server summary details.
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

type ProxyConnections struct {
	Proxies []ProxyConnection `json:"items"`
}

type ProxyConnection struct {
	NodeID               string `json:"nodeId"`
	RemoteAddress        string `json:"remoteAddress"`
	RemotePort           int32  `json:"remotePort"`
	TimeStamp            string `json:"timeStamp"`
	ConnectionTimeMillis int64  `json:"connectionTimeMillis"`
	ClientProcessName    string `json:"clientProcessName"`
	TotalBytesReceived   int64  `json:"totalBytesReceived"`
	TotalBytesSent       int64  `json:"totalBytesSent"`
	OutgoingByteBacklog  int64  `json:"outgoingByteBacklog"`
	UUID                 string `json:"UUID"`
	Member               string `json:"member"`
	ClientRole           string `json:"clientRole"`
}

// HTTPSessionSummaries contains an array of Coherence*Web Sessions.
type HTTPSessionSummaries struct {
	HTTPSessions []HTTPSessionSummary `json:"items"`
}

// HTTPSessionSummary contains a summary of Coherence*Web Sessions.
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

// ServicesSummaries contains an array of ServiceSummary.
type ServicesSummaries struct {
	Services []ServiceSummary `json:"items"`
}

// ServiceSummary contains a summary of individual services.
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
	QuorumStatus         string `json:"quorumStatus"`

	// persistence related
	PersistenceMode                   string  `json:"persistenceMode"`
	PersistenceActiveSpaceUsed        int64   `json:"persistenceActiveSpaceUsed"`
	PersistenceBackupSpaceUsed        int64   `json:"persistenceBackupSpaceUsed"`
	PersistenceLatencyMax             int64   `json:"persistenceLatencyMax"`
	PersistenceLatencyAverage         float64 `json:"persistenceLatencyAverage"`
	PersistenceSnapshotSpaceAvailable int64   `json:"persistenceSnapshotSpaceAvailable"`

	// derived
	PersistenceLatencyAverageTotal float64
	Snapshots                      []string
	OperationStatus                string
	Idle                           bool
}

// ServicesStorageSummaries contains an array of ServiceStorageSummary.
type ServicesStorageSummaries struct {
	Services []ServiceStorageSummary `json:"items"`
}

// ServiceStorageSummary contains a storage summary for individual services.
type ServiceStorageSummary struct {
	ServiceName            string `json:"service"`
	AveragePartitionSizeKB int64  `json:"averagePartitionSizeKB"`
	MaxPartitionSizeKB     int64  `json:"maxPartitionSizeKB"`
	AverageStorageSizeKB   int64  `json:"averageStorageSizeKB"`
	MaxStorageSizeKB       int64  `json:"maxStorageSizeKB"`
	MaxLoadNodeID          int32  `json:"maxLoadNodeId"`
	FairSharePrimary       int32  `json:"fairSharePrimary"`
	FairShareBackup        int32  `json:"fairShareBackup"`
	PartitionCount         int32  `json:"partitionCount"`
	ServiceNodeCount       int32  `json:"serviceNodeCount"`
}

// HealthSummaries contains and array of HealthSummary.
type HealthSummaries struct {
	Summaries []HealthSummary `json:"items"`
}

// HealthSummary contains individual health summary details.
type HealthSummary struct {
	Type              string `json:"type"`
	SubType           string `json:"subType"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	NodeID            string `json:"nodeId"`
	ClassName         string `json:"className"`
	MemberHealthCheck bool   `json:"memberHealthCheck"`
	Ready             bool   `json:"ready"`
	Started           bool   `json:"started"`
	Live              bool   `json:"live"`
	Safe              bool   `json:"safe"`
}

// HealthSummaryShort contains summarised health across all nodes for a SubType and Name.
type HealthSummaryShort struct {
	TotalCount   int32
	SubType      string
	Name         string
	Description  string
	ReadyCount   int32
	StartedCount int32
	LiveCount    int32
	SafeCount    int32
}

// StatsSummary contains statistics summaries.
type StatsSummary struct {
	Count   int64   `json:"count"`
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Sum     float64 `json:"sum"`
}

// FederationSummaries contains an array of FederationSummary.
type FederationSummaries struct {
	Services []FederationSummary `json:"items"`
}

// FederationSummary contains Federation summary details for a service and participant.
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

	// additional replicate all stats only available in 21.12.2+ or 14.1.1. FP1
	ReplicateAllPartitionCount         StatsSummary `json:"replicateAllPartitionCount"`
	ReplicateAllPartitionErrorCount    StatsSummary `json:"replicateAllPartitionErrorCount"`
	TotalReplicateAllPartitionsUnacked StatsSummary `json:"totalReplicateAllPartitionsUnacked"`
}

// FederationDescription contains federation description details.
type FederationDescription struct {
	NodeID                                  string  `json:"nodeId"`
	State                                   string  `json:"state"`
	TotalMsgSent                            int64   `json:"totalMsgSent"`
	TotalBytesSent                          int64   `json:"totalBytesSent"`
	TotalRecordsSent                        int64   `json:"totalRecordsSent"`
	MsgNetworkRoundTripTimePercentileMillis int64   `json:"msgNetworkRoundTripTimePercentileMillis"`
	MsgApplyTimePercentileMillis            int64   `json:"msgApplyTimePercentileMillis"`
	RecordBacklogDelayTimePercentileMillis  int64   `json:"recordBacklogDelayTimePercentileMillis"`
	ReplicateAllPercentComplete             int64   `json:"replicateAllPercentComplete"`
	ReplicateAllTotalTime                   int64   `json:"replicateAllTotalTime"`
	CurrentBandwidth                        float64 `json:"currentBandwidth"`

	TotalMsgReceived       int64 `json:"totalMsgReceived"` // incoming
	TotalBytesReceived     int64 `json:"totalBytesReceived"`
	TotalRecordsReceived   int64 `json:"totalRecordsReceived"`
	CurrentConnectionCount int32 `json:"currentConnectionCount"`

	// additional replicate all stats only available in 21.12.2+ or 14.1.1. FP1
	ReplicateAllPartitionCount         int64 `json:"replicateAllPartitionCount"`
	ReplicateAllPartitionErrorCount    int64 `json:"replicateAllPartitionErrorCount"`
	TotalReplicateAllPartitionsUnacked int64 `json:"totalReplicateAllPartitionsUnacked"`
}

// ServiceMemberDetails contains service members details.
type ServiceMemberDetails struct {
	Services []ServiceMemberDetail `json:"items"`
}

// ServiceMemberDetail contains details about a service and node.
type ServiceMemberDetail struct {
	NodeID                 string  `json:"nodeId"`
	ThreadCount            int32   `json:"threadCount"`
	ThreadCountMin         int32   `json:"threadCountMin"`
	ThreadCountMax         int32   `json:"threadCountMax"`
	ThreadIdleCount        int32   `json:"threadIdleCount"`
	TaskCount              int32   `json:"taskCount"`
	TaskBacklog            int32   `json:"taskBacklog"`
	OwnedPartitionsPrimary int32   `json:"ownedPartitionsPrimary"`
	OwnedPartitionsBackup  int32   `json:"ownedPartitionsBackup"`
	RequestAverageDuration float32 `json:"requestAverageDuration"`
	TaskAverageDuration    float32 `json:"taskAverageDuration"`
}

// CacheSummaries contains cache summary details.
type CacheSummaries struct {
	Caches []CacheSummaryDetail `json:"items"`
}

// CacheSummaryDetail contains a summary of cache details.
type CacheSummaryDetail struct {
	ServiceName    string `json:"service"`
	CacheName      string `json:"name"`
	CacheSize      int32  `json:"size"`
	UnitsBytes     int64  `json:"unitsBytes"`
	TotalPuts      int64  `json:"totalPuts"`
	TotalGets      int64  `json:"totalGets"`
	TotalRemoves   int64  `json:"removeCount"`
	CacheHits      int64  `json:"cacheHits"`
	CacheMisses    int64  `json:"cacheMisses"`
	TotalEvictions int64  `json:"evictionCount"`
}

// TopicDetails contains topics details.
type TopicDetails struct {
	Details []TopicDetail `json:"items"`
}

// TopicDetail contains individual topic details.
type TopicDetail struct {
	ServiceName    string `json:"service"`
	TopicName      string `json:"name"`
	Members        int64  `json:"members"`
	Channels       int64  `json:"-"`
	Subscribers    int64  `json:"-"`
	PublishedCount int64  `json:"-"`
}

// TopicsMemberDetails contains topics member details.
type TopicsMemberDetails struct {
	Details []TopicsMemberDetail `json:"items"`
}

// TopicsMemberDetail contains individual detailed member information for a topic.
type TopicsMemberDetail struct {
	ServiceName         string                 `json:"service"`
	TopicName           string                 `json:"name"`
	NodeID              string                 `json:"nodeId"`
	ChannelCount        int64                  `json:"channelCount"`
	ReconnectRetry      int64                  `json:"reconnectRetry"`
	RetainConsumed      bool                   `json:"retainConsumed"`
	AllowUnownedCommits bool                   `json:"allowUnownedCommits"`
	SubscriberTimeout   int64                  `json:"subscriberTimeout"`
	ReconnectTimeout    int64                  `json:"reconnectTimeout"`
	ReconnectWait       int64                  `json:"reconnectWait"`
	PageCapacity        int64                  `json:"pageCapacity"`
	ElementCalculator   string                 `json:"elementCalculator"`
	Member              string                 `json:"member"`
	Cluster             string                 `json:"cluster"`
	Channels            map[string]interface{} `json:"channels"`

	PublishedCount             int64   `json:"publishedCount"`
	PublishedMeanRate          float64 `json:"publishedMeanRate"`
	PublishedOneMinuteRate     float64 `json:"publishedOneMinuteRate"`
	PublishedFiveMinuteRate    float64 `json:"publishedFiveMinuteRate"`
	PublishedFifteenMinuteRate float64 `json:"publishedFifteenMinuteRate"`
}

// TopicsSubscriberDetails contains topics subscriber details.
type TopicsSubscriberDetails struct {
	Details []TopicsSubscriberDetail `json:"items"`
}

// TopicsSubscriberDetail contains individual detailed subscriber information for a topic.
type TopicsSubscriberDetail struct {
	ServiceName        string                 `json:"service"`
	TopicName          string                 `json:"topic"`
	NodeID             string                 `json:"nodeId"`
	ID                 int64                  `json:"id"`
	ChannelCount       int64                  `json:"channelCount"`
	StateName          string                 `json:"stateName"`
	SubscriberGroup    string                 `json:"subscriberGroup"`
	ReceiveCompletions int64                  `json:"receiveCompletions"`
	Waits              int64                  `json:"waits"`
	ReceiveErrors      int64                  `json:"receiveErrors"`
	ReceivedCount      int64                  `json:"receivedCount"`
	Disconnections     int64                  `json:"disconnections"`
	Notifications      int64                  `json:"notifications"`
	Backlog            int64                  `json:"backlog"`
	Member             string                 `json:"member"`
	Cluster            string                 `json:"cluster"`
	Channels           map[string]interface{} `json:"channels"`
	SubType            string                 `json:"subType"`
}

// HeadsResult contains raw results from retrieve heads.
type HeadsResult struct {
	Channels map[string]interface{} `json:"heads"`
}

// HeadStats contains retrieved heads details.
type HeadStats struct {
	Channel  int64  `json:"channel"`
	Position string `json:"position"`
}

// TopicsSubscriberGroups contains details about subscriber groups.
type TopicsSubscriberGroups struct {
	Details []TopicsSubscriberGroupDetail `json:"items"`
}

// TopicsSubscriberGroupDetail contains detail about subscriber groups.
type TopicsSubscriberGroupDetail struct {
	ServiceName             string                 `json:"service"`
	TopicName               string                 `json:"topic"`
	NodeID                  string                 `json:"nodeId"`
	SubscriberGroup         string                 `json:"name"`
	ChannelCount            int64                  `json:"channelCount"`
	PolledCount             int64                  `json:"polledCount"`
	PolledMeanRate          float64                `json:"polledMeanRate"`
	PolledOneMinuteRate     float64                `json:"polledOneMinuteRate"`
	PolledFiveMinuteRate    float64                `json:"polledFiveMinuteRate"`
	PolledFifteenMinuteRate float64                `json:"polledFifteenMinuteRate"`
	Channels                map[string]interface{} `json:"channels"`
}

// ChannelDetails contains all channels details.
type ChannelDetails struct {
	Details map[string]interface{} `json:"channels"`
}

// ChannelStats contains statistics summaries for Channels.
type ChannelStats struct {
	Channel                    int64   `json:"channel"`
	PublishedCount             int64   `json:"publishedCount"`
	PublishedFifteenMinuteRate float64 `json:"publishedFifteenMinuteRate"`
	PublishedFiveMinuteRate    float64 `json:"publishedFiveMinuteRate"`
	PublishedMeanRate          float64 `json:"publishedMeanRate"`
	PublishedOneMinuteRate     float64 `json:"publishedOneMinuteRate"`
	Tail                       string  `json:"tail"`
}

// SubscriberChannelStats contains statistics summaries for channel subscribers.
type SubscriberChannelStats struct {
	Channel      int64  `json:"channel"`
	Empty        bool   `json:"empty"`
	Owned        bool   `json:"owned"`
	Head         string `json:"head"`
	LastCommit   string `json:"lastCommit"`
	LastReceived string `json:"lastReceived"`
}

// SubscriberGroupChannelStats contains statistics summaries for channel subscriber groups.
type SubscriberGroupChannelStats struct {
	Channel                              int64   `json:"channel"`
	Head                                 string  `json:"head"`
	LastCommittedPosition                string  `json:"lastCommittedPosition"`
	LastCommittedTimestamp               string  `json:"lastCommittedTimestamp"`
	LastPolledTimestamp                  string  `json:"lastPolledTimestamp"`
	OwningSubscriberID                   int64   `json:"owningSubscriberId"`
	OwningSubscriberMemberID             int64   `json:"owningSubscriberMemberId"`
	OwningSubscriberMemberNotificationID int64   `json:"owningSubscriberMemberNotificationId"`
	OwningSubscriberMemberUUID           string  `json:"owningSubscriberMemberUuid"`
	PolledCount                          int64   `json:"polledCount"`
	PolledMeanRate                       float64 `json:"polledMeanRate"`
	PolledOneMinuteRate                  float64 `json:"polledOneMinuteRate"`
	PolledFiveMinuteRate                 float64 `json:"polledFiveMinuteRate"`
	PolledFifteenMinuteRate              float64 `json:"polledFifteenMinuteRate"`
	RemainingUnpolledMessages            int64   `json:"remainingUnpolledMessages"`
}

// CacheDetails contains cache details
type CacheDetails struct {
	Details []CacheDetail `json:"items"`
}

// CacheDetail contains individual cache details for a cache, tier and node.
type CacheDetail struct {
	NodeID        string `json:"nodeId"`
	Tier          string `json:"tier"`
	UnitsBytes    int64  `json:"unitsBytes"`
	Units         int64  `json:"units"`
	UnitFactor    int64  `json:"unitFactor"`
	CacheSize     int32  `json:"size"`
	TotalPuts     int64  `json:"totalPuts"`
	TotalGets     int64  `json:"totalGets"`
	TotalRemoves  int64  `json:"removeCount"`
	CacheHits     int64  `json:"cacheHits"`
	CacheMisses   int64  `json:"cacheMisses"`
	Evictions     int64  `json:"evictionCount"`
	StoreReads    int64  `json:"storeReads"`
	StoreWrites   int64  `json:"storeWrites"`
	StoreFailures int64  `json:"storeFailures"`

	LocksGranted                   int64    `json:"locksGranted"`
	LocksPending                   int64    `json:"locksPending"`
	ListenerRegistrations          int64    `json:"listenerRegistrations"`
	ListenerKeyCount               int64    `json:"listenerKeyCount"`
	ListenerFilterCount            int64    `json:"listenerFilterCount"`
	MaxQueryDurationMillis         int64    `json:"maxQueryDurationMillis"`
	MaxQueryDescription            string   `json:"maxQueryDescription"`
	NonOptimizedQueryAverageMillis float64  `json:"nonOptimizedQueryAverageMillis"`
	OptimizedQueryAverageMillis    float64  `json:"optimizedQueryAverageMillis"`
	IndexTotalUnits                int64    `json:"indexTotalUnits"`
	IndexingTotalMillis            int64    `json:"indexingTotalMillis"`
	IndexInfo                      []string `json:"indexInfo"`
}

// ServiceCaches contains a list of service cache.
type ServiceCaches struct {
	Details []ServiceCache `json:"items"`
}

// ServiceCache contains an individual service cache mapping.
type ServiceCache struct {
	ServiceName string `json:"service"`
	Name        string `json:"name"`
}

// CacheStoreDetails contains cache details.
type CacheStoreDetails struct {
	Details []CacheStoreDetail `json:"items"`
}

// CacheStoreDetail contains the cache store information.
type CacheStoreDetail struct {
	NodeID                  string `json:"nodeId"`
	Tier                    string `json:"tier"`
	QueueSize               int64  `json:"queueSize"`
	StoreAverageBatchSize   int64  `json:"storeAverageBatchSize"`
	StoreWrites             int64  `json:"storeWrites"`
	StoreAverageWriteMillis int64  `json:"storeAverageWriteMillis"`
	StoreWriteMillis        int64  `json:"storeWriteMillis"`
	StoreFailures           int64  `json:"storeFailures"`
	StoreReads              int64  `json:"storeReads"`
	StoreAverageReadMillis  int64  `json:"storeAverageReadMillis"`
	StoreReadMillis         int64  `json:"storeReadMillis"`
	PersistenceType         string `json:"persistenceType"`
}

// GenericDetails contains a slice of generic Json structures.
type GenericDetails struct {
	Details []interface{} `json:"items"`
}

// PersistenceCoordinator contains details about a persistence coordinator.
type PersistenceCoordinator struct {
	Idle              bool     `json:"idle"`
	OperationStatus   string   `json:"operationStatus"`
	Snapshots         []string `json:"snapshots"`
	CoordinatorNodeID int32    `json:"coordinatorId"`
}

// Machine contains machine details.
type Machine struct {
	MachineName             string      `json:"operationStatus"`
	AvailableProcessors     int32       `json:"availableProcessors"`
	SystemLoadAverage       float32     `json:"systemLoadAverage"` // check first
	SystemCPULoad           interface{} `json:"systemCpuLoad"`     // check second
	TotalPhysicalMemorySize int64       `json:"totalPhysicalMemorySize"`
	FreePhysicalMemorySize  int64       `json:"freePhysicalMemorySize"`
	Arch                    string      `json:"arch"`
	Name                    string      `json:"name"`
	Version                 string      `json:"version"`
}

// Reporters contains reporter details.
type Reporters struct {
	Reporters []Reporter `json:"items"`
}

// Reporter contains individual node reporter details.
type Reporter struct {
	NodeID           string  `json:"nodeId"`
	State            string  `json:"state"`
	OutputPath       string  `json:"outputPath"`
	ConfigFile       string  `json:"configFile"`
	LastReport       string  `json:"lastReport"`
	LastRunMillis    int32   `json:"runLastMillis"`
	CurrentBatch     int32   `json:"currentBatch"`
	IntervalSeconds  int32   `json:"intervalSeconds"`
	RunAverageMillis float64 `json:"runAverageMillis"`
	AutoStart        bool    `json:"autoStart"`
}

// ElasticDataValues contains elastic data details.
type ElasticDataValues struct {
	ElasticData []ElasticData `json:"items"`
}

// ElasticData contains elastic data information for a node and type.
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

// DefaultDependency holds the default dependencies for starting a Cache server.
type DefaultDependency struct {
	GroupID     string
	Artifact    string
	IsCoherence bool
	Version     string
}

// Links contains any links returned via HTTP.
type Links struct {
	Links []Link `json:"links"`
}

// Link contains link details.
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// ItemLinks contains links for an item.
type ItemLinks struct {
	Links []Links `json:"items"`
}

// Snapshots contains snapshots for services.
type Snapshots struct {
	ServiceName string   `json:"serviceName"`
	Snapshots   []string `json:"snapshots"`
}

// Archives contains archived snapshots.
type Archives struct {
	Snapshots []string `json:"archives"`
}

// StatusValues contains JFR status result.
type StatusValues struct {
	Status []string `json:"status"`
}

// SingleStatusValue contains a single JFR status result.
type SingleStatusValue struct {
	Status string `json:"status"`
}

// Distributions contains scheduled distributions.
type Distributions struct {
	ScheduledDistributions string `json:"scheduledDistributions"`
}

// Description contains description for an item.
type Description struct {
	Description string `json:"description"`
}
