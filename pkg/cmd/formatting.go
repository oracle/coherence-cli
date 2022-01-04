/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/discovery"
	"github.com/oracle/coherence-cli/pkg/utils"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const sep = "~~"
const R = "R"
const L = "L"

const NodeIDColumn = "NODE ID"
const ServiceColumn = "SERVICE"
const CacheColumn = "CACHE"
const ServiceNameColumn = "SERVICE NAME"
const AddressColumn = "ADDRESS"
const PortColumn = "PORT"
const MemberColumn = "MEMBER"
const RoleColumn = "ROLE"
const ProcessColumn = "PROCESS"
const MaxHeapColumn = "MAX HEAP"
const UsedHeapColumn = "USED HEAP"
const AvailHeapColumn = "AVAIL HEAP"
const NameColumn = "NAME"

type KeyValues struct {
	Key   string
	Value interface{}
}

var printer = message.NewPrinter(language.English)

// FormatCurrentCluster will display a message indicating a cluster context is being used
func FormatCurrentCluster(clusterName string) string {
	if UsingContext {
		return fmt.Sprintf("Using cluster connection '%s' from current context.\n", clusterName)
	}
	return ""
}

// FormatCluster returns a string representing a cluster
func FormatCluster(cluster config.Cluster) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Cluster Name:    %s\n", cluster.ClusterName))
	sb.WriteString(fmt.Sprintf("Version:         %s\n", cluster.Version))
	sb.WriteString(fmt.Sprintf("Cluster Size:    %d\n", cluster.ClusterSize))
	sb.WriteString(fmt.Sprintf("License Mode:    %s\n", cluster.LicenseMode))
	sb.WriteString(fmt.Sprintf("Departure Count: %d\n", cluster.MembersDepartureCount))
	sb.WriteString(fmt.Sprintf("Running:         %v\n", cluster.Running))

	return sb.String()
}

// FormatJSONForDescribe formats a two column display for a describe command
// showAllColumns indicates if all the columns including ordered are shown
// orderedColumns are the column names, expanded, that should be displayed first for context
func FormatJSONForDescribe(jsonValue []byte, showAllColumns bool, orderedColumns ...string) (string, error) {
	var result map[string]json.RawMessage
	err := json.Unmarshal(jsonValue, &result)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal value in FormatJSONForDescribe %v", err)
	}

	var keyValues = make([]KeyValues, len(result))

	var i = 0
	for k := range result {
		var value interface{}
		err = json.Unmarshal(result[k], &value)
		if err != nil {
			return "", errors.New("Unable to unmarshal " + k)
		}
		keyValues[i] = KeyValues{Key: CreateCamelCaseLabel(k), Value: value}
		i++
	}

	// get max length of first column
	var maxLength = 0
	for _, v := range keyValues {
		length := len(v.Key)
		// ignore if we are not showing all columns and the column is not in the list
		if !showAllColumns && !utils.SliceContains(orderedColumns, v.Key) {
			continue
		}

		if length > maxLength {
			maxLength = length
		}
	}

	keyFormat := fmt.Sprintf("%%-%ds:  %%s\n", maxLength)

	var sb strings.Builder

	// display any orderedColumns first
	if len(orderedColumns) > 0 {
		for _, column := range orderedColumns {
			index := findKeyValueIndex(keyValues, column)
			if index != -1 {
				appendColumnValue(keyValues[index], &sb, keyFormat)
				// remove the column value
				keyValues = append(keyValues[:index], keyValues[index+1:]...)
			}
		}
	}

	if showAllColumns {
		// sort the rest by Key so they come out consistently
		sort.Slice(keyValues, func(p, q int) bool {
			return strings.Compare(keyValues[p].Key, keyValues[q].Key) < 0
		})

		for _, v := range keyValues {
			appendColumnValue(v, &sb, keyFormat)
		}
	}

	return sb.String(), nil
}

// FormatFederationSummary returns the federation summary in column formatted output
// the target may be destinations or origins and columns will change slightly
func FormatFederationSummary(federationSummaries []config.FederationSummary, target string) string {
	var (
		fedCount       = len(federationSummaries)
		finalAlignment []string
		suffix         = "SENT"
		participantCol = "DESTINATION"
		memberCol      = "MEMBERS"
	)

	if fedCount == 0 {
		return ""
	}

	// setup columns and alignments
	if target == origins {
		suffix = "REC"
		participantCol = "ORIGIN"
		memberCol = "MEMBERS RECEIVING"
	}

	if OutputFormat == constants.TABLE {
		if target == destinations {
			finalAlignment = []string{L, L, R, L, R, R, R, R}
		} else {
			finalAlignment = []string{L, L, R, L, R, R}
		}
	} else { // WIDE
		if target == destinations {
			finalAlignment = []string{L, L, R, L, R, R, R, R, R, R, R, R}
		} else {
			finalAlignment = []string{L, L, R, L, R, R, R, R}
		}
	}

	sort.Slice(federationSummaries, func(p, q int) bool {
		if federationSummaries[p].ServiceName < federationSummaries[q].ServiceName {
			return true
		} else if federationSummaries[p].ServiceName > federationSummaries[q].ServiceName {
			return false
		} else {
			return federationSummaries[p].ParticipantName < federationSummaries[q].ParticipantName
		}
	})

	var stringValues = make([]string, fedCount+1)

	if target == destinations {
		stringValues[0] = getColumns(ServiceColumn, participantCol, memberCol, "STATES", "DATA "+suffix,
			"MSG "+suffix, "REC "+suffix, "CURR AVG BWIDTH")
	} else {
		stringValues[0] = getColumns(ServiceColumn, participantCol, memberCol, "DATA "+suffix,
			"MSG "+suffix, "REC "+suffix)
	}

	if OutputFormat == constants.WIDE {
		if target == destinations {
			stringValues[0] = getColumns(stringValues[0], "AVG APPLY", "AVG ROUND TRIP", "AVG BACKLOG DELAY", "REPLICATE")
		} else {
			stringValues[0] = getColumns(stringValues[0], "AVG APPLY", "AVG BACKLOG DELAY")
		}
	}

	var (
		bytes     float64
		messages  float64
		records   float64
		members   int32
		bandwidth string
	)

	for i, value := range federationSummaries {
		if target == "destinations" {
			bytes = value.TotalBytesSent.Sum
			messages = value.TotalMsgSent.Sum
			records = value.TotalRecordsSent.Sum
			members = int32(len(value.State))
			bandwidth = formatMbps(float32(value.CurrentBandwidth.Average))
		} else {
			bytes = value.TotalBytesReceived.Sum
			messages = value.TotalMsgReceived.Sum
			records = value.TotalRecordsReceived.Sum
			members = int32(len(value.Member))
			bandwidth = "n/a"
		}

		if target == destinations {
			stringValues[i+1] = getColumns(value.ServiceName, value.ParticipantName,
				formatSmallInteger(members), fmt.Sprintf("%v", utils.GetUniqueValues(value.State)),
				formatMB(int32(bytes/1024/1024)), formatLargeInteger(int64(messages)),
				formatLargeInteger(int64(records)), bandwidth)
		} else {
			stringValues[i+1] = getColumns(value.ServiceName, value.ParticipantName,
				formatSmallInteger(members),
				formatMB(int32(bytes/1024/1024)), formatLargeInteger(int64(messages)),
				formatLargeInteger(int64(records)))
		}

		if OutputFormat == constants.WIDE {
			if target == destinations {
				stringValues[i+1] = getColumns(stringValues[i+1],
					formatLatency0(float32(value.MsgApplyTimePercentileMillis.Average)),
					formatLatency0(float32(value.MsgNetworkRoundTripTimePercentileMillis.Average)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis.Average)),
					formatPercent(value.ReplicateAllPercentComplete.Average/100))
			} else {
				stringValues[i+1] = getColumns(stringValues[i+1],
					formatLatency0(float32(value.MsgApplyTimePercentileMillis.Average)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis.Average)))
			}
		}
	}

	return formatLinesAllStringsWithAlignment(finalAlignment, stringValues)
}

// FormatCacheSummary returns the cache summary in column formatted output
func FormatCacheSummary(cacheSummaries []config.CacheSummaryDetail) string {
	var (
		cacheCount     = len(cacheSummaries)
		alignmentWide  = []string{L, L, R, R, R, R, R, R, R, R, R, R}
		alignment      = []string{L, L, R, R, R}
		finalAlignment []string
	)

	if cacheCount == 0 {
		return ""
	}
	if OutputFormat == constants.TABLE {
		finalAlignment = alignment
	} else {
		finalAlignment = alignmentWide
	}

	var stringValues = make([]string, cacheCount+1)

	sort.Slice(cacheSummaries, func(p, q int) bool {
		if cacheSummaries[p].ServiceName < cacheSummaries[q].ServiceName {
			return true
		} else if cacheSummaries[p].ServiceName > cacheSummaries[q].ServiceName {
			return false
		} else {
			return cacheSummaries[p].CacheName < cacheSummaries[q].CacheName
		}
	})

	// get summary details
	var totalCaches = len(cacheSummaries)
	var totalUnits int64 = 0

	stringValues[0] = getColumns(ServiceColumn, CacheColumn, "CACHE SIZE", "BYTES", "MB")

	if OutputFormat == constants.WIDE {
		stringValues[0] = getColumns(stringValues[0], "AVG SIZE",
			"TOTAL PUTS", "TOTAL GETS", "TOTAL REMOVES", "TOTAL HITS", "TOTAL MISSES", "HIT PROB")
	}

	for i, value := range cacheSummaries {
		var (
			hitProb       = 0.0
			avgSize int64 = 0
		)
		totalGets := value.TotalGets
		totalHits := value.CacheHits
		if totalGets != 0 {
			hitProb = float64(totalHits) / float64(totalGets)
		}
		totalUnits += value.UnitsBytes

		if value.CacheSize != 0 {
			avgSize = value.UnitsBytes / int64(value.CacheSize)
		}

		stringValues[i+1] = getColumns(value.ServiceName, value.CacheName, formatSmallInteger(value.CacheSize),
			formatLargeInteger(value.UnitsBytes), formatMB(int32(value.UnitsBytes)/1024/1024))

		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1], formatLargeInteger(avgSize),
				formatLargeInteger(value.TotalPuts), formatLargeInteger(totalGets),
				formatLargeInteger(value.TotalRemoves), formatLargeInteger(totalHits),
				formatLargeInteger(value.CacheMisses), formatPercent(hitProb))
		}
	}

	return fmt.Sprintf("Total Caches: %d, Total primary storage: %s\n\n", totalCaches,
		strings.TrimSpace(formatMB(int32(totalUnits/1024/1024)))) +
		formatLinesAllStringsWithAlignment(finalAlignment, stringValues)
}

// FormatTopicsSummary returns the topics summary in column formatted output
func FormatTopicsSummary(cacheSummaries []config.CacheSummaryDetail) string {
	var (
		cacheCount = len(cacheSummaries)
		alignment  = []string{L, L, R, R, R, R, R, R}
	)
	if cacheCount == 0 {
		return ""
	}

	var stringValues = make([]string, cacheCount+1)

	sort.Slice(cacheSummaries, func(p, q int) bool {
		if cacheSummaries[p].ServiceName < cacheSummaries[q].ServiceName {
			return true
		} else if cacheSummaries[p].ServiceName > cacheSummaries[q].ServiceName {
			return false
		} else {
			return cacheSummaries[p].CacheName < cacheSummaries[q].CacheName
		}
	})

	// get summary details
	var totalTopics = len(cacheSummaries)
	var totalUnits int64 = 0

	stringValues[0] = getColumns(ServiceColumn, "TOPIC", "UNCONSUMED MSG", "BYTES", "MB", "AVG SIZE",
		"PUBLISHER SENDS", "SUBSCRIBER RECEIVES")

	for i, value := range cacheSummaries {
		var avgSize int64 = 0

		totalUnits += value.UnitsBytes

		if value.CacheSize != 0 {
			avgSize = value.UnitsBytes / int64(value.CacheSize)
		}

		stringValues[i+1] = getColumns(value.ServiceName, strings.ReplaceAll(value.CacheName, "$topic$", ""),
			formatSmallInteger(value.CacheSize),
			formatLargeInteger(value.UnitsBytes), formatMB(int32(value.UnitsBytes)/1024/1024),
			formatLargeInteger(avgSize), formatLargeInteger(value.TotalPuts), formatLargeInteger(value.TotalGets))
	}

	return fmt.Sprintf("Total Topics: %d, Total primary storage: %s\n\n", totalTopics,
		strings.TrimSpace(formatMB(int32(totalUnits/1024/1024)))) +
		formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// FormatServiceMembers returns the service member details in column formatted output
func FormatServiceMembers(serviceMembers []config.ServiceMemberDetail) string {
	var (
		memberCount    = len(serviceMembers)
		alignmentWide  = []string{R, R, R, R, R, R, R, R, R, R, R, R}
		alignment      = []string{R, R, R, R, R, R}
		finalAlignment []string
	)
	if memberCount == 0 {
		return ""
	}

	var stringValues = make([]string, memberCount+1)

	sort.Slice(serviceMembers, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(serviceMembers[p].NodeID)
		nodeID2, _ := strconv.Atoi(serviceMembers[q].NodeID)
		return nodeID1 < nodeID2
	})

	stringValues[0] = getColumns(NodeIDColumn, "THREADS", "IDLE", "THREAD UTIL", "MIN THREADS", "MAX THREADS")
	if OutputFormat == constants.WIDE {
		finalAlignment = alignmentWide
		stringValues[0] = getColumns(stringValues[0], "TASK COUNT", "TASK BACKLOG", "PRIMARY OWNED",
			"BACKUP OWNED", "REQ AVG MS", "TASK AVG MS")
	} else {
		finalAlignment = alignment
	}

	for i, value := range serviceMembers {
		var nodeID, _ = strconv.Atoi(value.NodeID)
		var utilization float64 = -1
		if value.ThreadCount > 0 {
			utilization = float64(value.ThreadCount-value.ThreadIdleCount) / float64(value.ThreadCount)
		}
		stringValues[i+1] = getColumns(formatSmallInteger(int32(nodeID)), formatSmallInteger(value.ThreadCount),
			formatSmallInteger(value.ThreadIdleCount), formatPercent(utilization),
			formatSmallInteger(value.ThreadCountMin), formatSmallInteger(value.ThreadCountMax))
		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1],
				formatSmallInteger(value.TaskCount), formatSmallInteger(value.TaskCountBacklog),
				formatSmallInteger(value.OwnedPartitionsPrimary), formatSmallInteger(value.OwnedPartitionsBackup),
				formatFloat(value.RequestAverageDuration), formatFloat(value.TaskAverageDuration))
		}
	}

	return formatLinesAllStringsWithAlignment(finalAlignment, stringValues)
}

// FormatCacheDetailsSizeAndAccess returns the cache details size and access details in column formatted output
func FormatCacheDetailsSizeAndAccess(cacheDetails []config.CacheDetail) (string, error) {
	var (
		err          error
		detailsCount = len(cacheDetails)
		alignment    []string
	)
	if detailsCount == 0 {
		return "", nil
	}

	var stringValues = make([]string, detailsCount+1)

	sort.Slice(cacheDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(cacheDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(cacheDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	stringValues[0] = getColumns(NodeIDColumn, "TIER", "SIZE", "MEMORY BYTES", "MEMORY MB",
		"TOTAL PUTS", "TOTAL GETS", "TOTAL REMOVES")
	if OutputFormat == constants.WIDE {
		alignment = []string{R, L, R, R, R, R, R, R, R, R, R, R, R, R}
		stringValues[0] = getColumns(stringValues[0], "HITS", "MISSES", "HIT PROB", "STORE READS",
			"WRITES", "FAILURES")
	} else {
		alignment = []string{R, L, R, R, R, R, R, R}
	}

	for i, value := range cacheDetails {
		var (
			nodeID, _ = strconv.Atoi(value.NodeID)
			hitProb   = 0.0
		)
		totalGets := value.TotalGets
		totalHits := value.CacheHits
		if totalGets != 0 {
			hitProb = float64(totalHits) / float64(totalGets)
		}

		stringValues[i+1] = getColumns(formatSmallInteger(int32(nodeID)), value.Tier,
			formatSmallInteger(value.CacheSize), formatLargeInteger(value.UnitsBytes),
			formatMB(int32(value.UnitsBytes)/1024/1024), formatLargeInteger(value.TotalPuts),
			formatLargeInteger(totalGets), formatLargeInteger(value.TotalRemoves))
		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1], formatLargeInteger(totalHits),
				formatLargeInteger(value.CacheMisses), formatPercent(hitProb),
				formatLargeInteger(value.StoreReads), formatLargeInteger(value.StoreWrites),
				formatLargeInteger(value.StoreFailures))
		}
	}

	return formatLinesAllStringsWithAlignment(alignment, stringValues), err
}

// FormatCacheIndexDetails returns the cache index details
func FormatCacheIndexDetails(cacheDetails []config.CacheDetail) string {
	var (
		sb                        = strings.Builder{}
		totalIndexUnits     int64 = 0
		totalIndexingMillis int64 = 0
	)

	for _, value := range cacheDetails {
		if value.Tier == "back" {
			totalIndexingMillis += value.IndexingTotalMillis
			totalIndexUnits += value.IndexTotalUnits
			var nodeString = "Node:" + value.NodeID + ": "
			format := fmt.Sprintf("%%-%ds  %%s\n", len(nodeString))
			for i, v := range value.IndexInfo {
				var node = nodeString
				if i > 0 {
					node = ""
				}
				sb.WriteString(fmt.Sprintf(format, node, v))

			}
		}
	}

	return "Total Indexing Bytes:  " + formatLargeInteger(totalIndexUnits) + "\n" +
		"Total Indexing:        " + formatMB(int32(totalIndexUnits/1024/1024)) + "\n" +
		"Total Indexing Millis: " + formatLargeInteger(totalIndexingMillis) + "\n" +
		"\n" +
		sb.String()
}

// FormatCacheDetailsStorage returns the cache storage details in column formatted output
func FormatCacheDetailsStorage(cacheDetails []config.CacheDetail) (string, error) {
	var (
		err          error
		detailsCount = len(cacheDetails)
		alignment    []string
	)
	if detailsCount == 0 {
		return "", nil
	}

	var stringValues = make([]string, detailsCount+1)

	sort.Slice(cacheDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(cacheDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(cacheDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	stringValues[0] = getColumns(NodeIDColumn, "TIER", "LOCKS GRANTED", "LOCKS PENDING", "LISTENERS",
		"MAX QUERY MS", "MAX QUERY DESC")
	if OutputFormat == constants.WIDE {
		stringValues[0] = getColumns(stringValues[0], "NO OPT AVG", "OPT AVG", "INDEX BYTES",
			"INDEX MB", "INDEXING MILLIS")
		alignment = []string{R, L, R, R, R, R, L, R, R, R, R, R}
	} else {
		alignment = []string{R, L, R, R, R, R, L}
	}

	for i, value := range cacheDetails {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		stringValues[i+1] = getColumns(formatSmallInteger(int32(nodeID)), value.Tier,
			formatLargeInteger(value.LocksGranted), formatLargeInteger(value.LocksPending),
			formatLargeInteger(value.ListenerRegistrations), formatLargeInteger(value.MaxQueryDurationMillis),
			value.MaxQueryDescription)
		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1], formatFloat(float32(value.NonOptimizedQueryAverageMillis)),
				formatFloat(float32(value.OptimizedQueryAverageMillis)), formatLargeInteger(value.IndexTotalUnits),
				formatMB(int32(value.IndexTotalUnits/1024/1024)), formatLargeInteger(value.IndexingTotalMillis))
		}
	}

	return formatLinesAllStringsWithAlignmentMax(alignment, stringValues, 40), err
}

// FormatDiscoveredClusters returns the discovered clusters in the column formatted output
func FormatDiscoveredClusters(clusters []discovery.DiscoveredCluster) string {
	var (
		count = len(clusters)
		i     = 0
	)
	if count == 0 {
		return ""
	}

	var stringValues = make([]string, count+1)

	stringValues[0] = getColumns("CONNECTION", "CLUSTER NAME", "HOST", "NS PORT", "URL")

	for _, value := range clusters {
		if value.SelectedURL != "" {
			stringValues[i+1] = getColumns(value.ConnectionName, value.ClusterName, value.Host, formatPort(int32(value.NSPort)), value.SelectedURL)
			i++
		}
	}
	return formatLinesAllStringsWithAlignment([]string{L, L, L, R, L}, stringValues)
}

// FormatClusterConnections returns the cluster information in a column formatted output
func FormatClusterConnections(clusters []ClusterConnection) string {
	var (
		clusterCount   = len(clusters)
		currentContext string
	)
	if clusterCount == 0 {
		return ""
	}

	var stringValues = make([]string, clusterCount+1)

	stringValues[0] = getColumns("CONNECTION", "TYPE", "URL", "VERSION", "CLUSTER NAME", "TYPE", "CTX")

	for i, value := range clusters {
		currentContext = ""
		if Config.CurrentContext == value.Name {
			currentContext = "*"
		}
		stringValues[i+1] = getColumns(value.Name, value.ConnectionType, value.ConnectionURL,
			value.ClusterVersion, value.ClusterName, value.ClusterType, currentContext)
	}

	return formatLinesAllStrings(stringValues)
}

// FormatMembers returns the member's information in a column formatted output
func FormatMembers(members []config.Member, verbose bool) string {
	var (
		memberCount    = len(members)
		alignmentWide  = []string{R, L, L, R, L, L, L, L, L, R, R, R, R, R}
		alignment      = []string{R, L, L, R, L, L, R, R, R}
		finalAlignment []string
	)

	if OutputFormat == constants.TABLE {
		finalAlignment = alignment
	} else {
		finalAlignment = alignmentWide
	}

	if memberCount == 0 {
		return ""
	}

	var stringValues = make([]string, memberCount+1)

	sort.Slice(members, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(members[p].NodeID)
		nodeID2, _ := strconv.Atoi(members[q].NodeID)
		return nodeID1 < nodeID2
	})

	var totalMaxMemoryMB int32 = 0
	var totalAvailMemoryMB int32 = 0

	stringValues[0] = getColumns(NodeIDColumn, AddressColumn, PortColumn, ProcessColumn, MemberColumn, RoleColumn)

	if OutputFormat == constants.WIDE {
		stringValues[0] = getColumns(stringValues[0], "MACHINE", "RACK", "SITE", "PUBLISHER", "RECEIVER")
	}
	stringValues[0] = getColumns(stringValues[0], MaxHeapColumn, UsedHeapColumn, AvailHeapColumn)

	for i, value := range members {
		var nodeID, _ = strconv.Atoi(value.NodeID)
		totalAvailMemoryMB += value.MemoryAvailableMB
		totalMaxMemoryMB += value.MemoryMaxMB

		stringValues[i+1] = getColumns(formatSmallInteger(int32(nodeID)), value.UnicastAddress,
			formatPort(value.UnicastPort), value.ProcessName, value.MemberName, value.RoleName)

		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1], value.MachineName, value.RackName, value.SiteName,
				formatPublisherReceiver(value.PublisherSuccessRate), formatPublisherReceiver(value.ReceiverSuccessRate))
		}

		stringValues[i+1] = getColumns(stringValues[i+1], formatMB(value.MemoryMaxMB),
			formatMB(value.MemoryMaxMB-value.MemoryAvailableMB), formatMB(value.MemoryAvailableMB))
	}

	totalUsedMB := totalMaxMemoryMB - totalAvailMemoryMB
	availablePercent := float32(totalAvailMemoryMB) / float32(totalMaxMemoryMB) * 100

	result :=
		fmt.Sprintf("Total cluster members: %d\n", memberCount) +
			fmt.Sprintf("Cluster Heap - Total: %s, Used: %s, Available: %s (%4.1f%%)\n",
				strings.TrimSpace(formatMB(totalMaxMemoryMB)),
				strings.TrimSpace(formatMB(totalUsedMB)),
				strings.TrimSpace(formatMB(totalAvailMemoryMB)), availablePercent)

	if verbose {
		result += formatLinesAllStringsWithAlignment(finalAlignment, stringValues)
	}
	return result
}

// FormatExecutors returns the executor's information in a column formatted output
func FormatExecutors(executors []config.Executor, summary bool) string {
	var (
		executorCount = len(executors)
		alignment     = []string{L, R, R, R, R, L}
		header        = "MEMBER COUNT"
	)
	if executorCount == 0 {
		return ""
	}

	if !summary {
		header = MemberColumn
	}

	var stringValues = make([]string, executorCount+1)

	sort.Slice(executors, func(p, q int) bool {
		return strings.Compare(executors[p].Name, executors[q].Name) > 0
	})

	stringValues[0] = getColumns(NameColumn, header, "IN PROGRESS", "COMPLETED", "REJECTED", "DESCRIPTION")

	var (
		totalRunningTasks   int64
		totalCompletedTasks int64
	)

	for i, value := range executors {
		var columnValue = value.MemberID
		if summary {
			columnValue = fmt.Sprintf("%d", value.MemberCount)
		}
		totalRunningTasks += value.TasksInProgressCount
		totalCompletedTasks += value.TasksCompletedCount
		stringValues[i+1] = getColumns(value.Name, columnValue,
			formatLargeInteger(value.TasksInProgressCount), formatLargeInteger(value.TasksCompletedCount),
			formatLargeInteger(value.TasksRejectedCount), value.Description)
	}

	return fmt.Sprintf("Total executors: %d\nRunning tasks:   %s\nCompleted tasks: %s\n\n",
		executorCount, formatLargeInteger(totalRunningTasks), formatLargeInteger(totalCompletedTasks)) +
		formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// FormatElasticData formats the elastic data summary
func FormatElasticData(edData []config.ElasticData, summary bool) string {
	var edCount = len(edData)
	if edCount == 0 {
		return ""
	}

	var (
		column1   = NameColumn
		alignment = []string{L, R, R, R, R, R, R, R, R, R}
	)

	sort.Slice(edData, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(edData[p].NodeID)
		nodeID2, _ := strconv.Atoi(edData[q].NodeID)
		return nodeID1 < nodeID2
	})

	var stringValues = make([]string, edCount+1)

	// if we are not a summary then change column 1
	if !summary {
		column1 = NodeIDColumn
		alignment[0] = R
	}

	stringValues[0] = getColumns(column1, "USED FILES", "TOTAL FILES", "% USED", "MAX FILE SIZE",
		"USED SPACE", "COMMITTED", "HIGHEST LOAD", "COMPACTIONS", "EXHAUSTIVE")

	for i, data := range edData {
		var (
			percentUsed  = float64(data.FileCount) / float64(data.MaxJournalFilesNumber)
			committed    = int64(data.FileCount) * data.MaxFileSize
			column1Value string
		)
		if summary {
			column1Value = data.Name
		} else {
			nodeID, _ := strconv.Atoi(data.NodeID) //nolint
			column1Value = formatSmallInteger(int32(nodeID))
		}
		stringValues[i+1] = getColumns(column1Value, formatSmallInteger(data.FileCount), formatSmallInteger(data.MaxJournalFilesNumber),
			formatPercent(percentUsed), formatMB(int32(data.MaxFileSize/1024/1024)),
			formatMB(int32(data.TotalDataSize)/1024/1024), formatMB(int32(committed/1024/1024)),
			formatLargeFloat(float64(data.HighestLoadFactor)),
			formatLargeInteger(data.CompactionCount), formatLargeInteger(data.ExhaustiveCompactionCount))
	}

	return formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// FormatReporters returns the reporters' info in a column formatted output
func FormatReporters(reporters []config.Reporter) string {
	var (
		memberCount = len(reporters)
		maxLength   = 40
	)
	if memberCount == 0 {
		return ""
	}

	var stringValues = make([]string, memberCount+1)

	sort.Slice(reporters, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(reporters[p].NodeID)
		nodeID2, _ := strconv.Atoi(reporters[q].NodeID)
		return nodeID1 < nodeID2
	})

	stringValues[0] = getColumns(NodeIDColumn, "STATE", "CONFIG FILE", "OUTPUT PATH",
		"BATCH#", "LAST REPORT", "LAST RUN", "AVG RUN", "AUTOSTART")

	for i, value := range reporters {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		stringValues[i+1] = getColumns(formatSmallInteger(int32(nodeID)), value.State, value.ConfigFile,
			value.OutputPath, formatSmallInteger(value.CurrentBatch), value.LastReport,
			formatSmallInteger(value.LastRunMillis)+"ms", formatLargeFloat(value.RunAverageMillis)+"ms",
			fmt.Sprintf("%v", value.AutoStart))
	}

	if OutputFormat == constants.WIDE {
		maxLength = 0
	}

	return formatLinesAllStringsWithAlignmentMax([]string{R, L, L, L, R, L, R, R, L}, stringValues, maxLength)
}

// FormatServices returns the services' information in a column formatted output
func FormatServices(services []config.ServiceSummary) string {
	var (
		serviceCount   = len(services)
		alignmentWide  = []string{L, L, R, L, R, R, R, R, R, L, L}
		alignment      = []string{L, L, R, L, R, R}
		finalAlignment []string
	)
	if serviceCount == 0 {
		return ""
	}

	var stringValues = make([]string, serviceCount+1)

	sort.Slice(services, func(p, q int) bool {
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) > 0
	})

	stringValues[0] = getColumns(ServiceNameColumn, "TYPE", "MEMBERS", "STATUS HA", "STORAGE", "PARTITIONS")
	if OutputFormat == constants.WIDE {
		finalAlignment = alignmentWide
		stringValues[0] = getColumns(stringValues[0], "ENDANGERED", "VULNERABLE", "UNBALANCED", "STATUS", "SUSPENDED")
	} else {
		finalAlignment = alignment
	}

	for i, value := range services {
		var (
			status    = "Safe"
			suspended = "n/a"
		)
		if value.StorageEnabledCount == -1 || value.StatusHA == "n/a" {
			status = "n/a"
		} else if value.StatusHA == "ENDANGERED" {
			status = "StatusHA is ENDANGERED"
		} else if value.PartitionsEndangered > 0 {
			status = fmt.Sprintf("%d partitions are endangered", value.PartitionsEndangered)
		} else if value.PartitionsVulnerable > 0 {
			status = fmt.Sprintf("%d partitions are vulnerable", value.PartitionsVulnerable)
		} else if value.PartitionsUnbalanced > 0 {
			status = fmt.Sprintf("%d partitions are unbalanced", value.PartitionsUnbalanced)
		}

		if value.QuorumStatus == "Suspended" {
			suspended = "yes"
		} else {
			if utils.IsDistributedCache(value.ServiceType) {
				suspended = "no"
			}
		}

		stringValues[i+1] = getColumns(value.ServiceName, value.ServiceType, formatSmallInteger(value.MemberCount),
			value.StatusHA, formatSmallInteger(value.StorageEnabledCount), formatSmallInteger(value.PartitionsAll))

		if OutputFormat == constants.WIDE {
			stringValues[i+1] = getColumns(stringValues[i+1], formatSmallInteger(value.PartitionsEndangered),
				formatSmallInteger(value.PartitionsVulnerable),
				formatSmallInteger(value.PartitionsUnbalanced), status, suspended)
		}

	}

	return formatLinesAllStringsWithAlignment(finalAlignment, stringValues)
}

// FormatMachines returns the machine's information in a column formatted output
func FormatMachines(machines []config.Machine) string {
	var serviceCount = len(machines)
	if serviceCount == 0 {
		return ""
	}

	var stringValues = make([]string, serviceCount+1)

	sort.Slice(machines, func(p, q int) bool {
		return strings.Compare(machines[p].MachineName, machines[q].MachineName) > 0
	})

	var (
		load        float32
		percentFree float64
	)

	stringValues[0] = getColumns("MACHINE", "PROCESSORS", "LOAD", "TOTAL MEMORY", "FREE MEMORY",
		"% FREE", "OS", "ARCH", "VERSION")

	for i, value := range machines {
		if value.SystemLoadAverage >= 0 {
			load = value.SystemLoadAverage
		} else {
			load = value.SystemCPULoad
		}

		percentFree = float64(value.FreePhysicalMemorySize) / float64(value.TotalPhysicalMemorySize)

		stringValues[i+1] = getColumns(value.MachineName, formatSmallInteger(value.AvailableProcessors),
			formatFloat(load), formatMB(int32(value.TotalPhysicalMemorySize/1024/1024)),
			formatMB(int32(value.FreePhysicalMemorySize/1024/1024)),
			formatPercent(percentFree), value.Name, value.Arch, value.Version)
	}

	return formatLinesAllStringsWithAlignment([]string{L, R, R, R, R, R, L, L, L}, stringValues)
}

// FormatHTTPSessions returns the Coherence*Web information in a column formatted output
func FormatHTTPSessions(sessions []config.HTTPSessionSummary, isSummary bool) string {
	var (
		serviceCount = len(sessions)
		alignment    []string
		header       = getColumns("TYPE", "SESSION TIMEOUT", CacheColumn, "OVERFLOW",
			"AVG SIZE", "TOTAL REAPED", "AVG DURATION", "LAST REAP", "UPDATES")
	)
	if serviceCount == 0 {
		return ""
	}

	var stringValues = make([]string, serviceCount+1)

	sort.Slice(sessions, func(p, q int) bool {
		if sessions[p].AppID == sessions[q].AppID {
			nodeID1, _ := strconv.Atoi(sessions[p].NodeID)
			nodeID2, _ := strconv.Atoi(sessions[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(sessions[p].AppID, sessions[q].AppID) > 0
	})

	if !isSummary {
		header = getColumns(NodeIDColumn, header)
		alignment = []string{R, L, R, L, L, R, R, R, R, R}
	} else {
		header = getColumns("APPLICATION", header)
		alignment = []string{L, L, R, L, L, R, R, R, R, R}
	}

	stringValues[0] = header

	var i = 0
	for _, value := range sessions {
		if isSummary {
			header = getColumns(value.AppID)
		} else {
			header = getColumns(value.NodeID)
		}

		header = getColumns(header, value.Type, formatSmallInteger(value.SessionTimeout), value.SessionCacheName,
			value.OverflowCacheName, formatSmallInteger(value.SessionAverageSize),
			formatLargeInteger(value.ReapedSessionsTotal), formatLargeInteger(value.AverageReapDuration),
			formatLargeInteger(value.LastReapDuration), formatLargeInteger(value.SessionUpdates))
		stringValues[i+1] = header
		i++
	}

	return formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// FormatPersistenceServices returns the services' persistence information in a column formatted output
// if isSummary then leave out storage count
func FormatPersistenceServices(services []config.ServiceSummary, isSummary bool) string {
	var (
		serviceCount = len(services)
		alignment    []string
	)
	if serviceCount == 0 {
		return ""
	}

	var stringValues = make([]string, serviceCount+1)

	sort.Slice(services, func(p, q int) bool {
		if services[p].ServiceName == services[q].ServiceName {
			nodeID1, _ := strconv.Atoi(services[p].NodeID)
			nodeID2, _ := strconv.Atoi(services[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) > 0
	})

	var (
		averageAverageLatency float64
		totalActiveSpaceUsed  int64
		header                string
	)

	if isSummary {
		alignment = []string{L, R, L, R, R, R, R, R, L}
		header = getColumns(ServiceNameColumn, "STORAGE COUNT", "PERSISTENCE MODE", "ACTIVE BYTES",
			"ACTIVE SPACE", "AVG LATENCY", "MAX LATENCY", "SNAPSHOTS", "STATUS")
	} else {
		alignment = []string{R, L, R, R, R, R}
		header = getColumns(NodeIDColumn, "PERSISTENCE MODE", "ACTIVE BYTES",
			"ACTIVE SPACE", "AVG LATENCY", "MAX LATENCY")
	}

	stringValues[0] = header

	var i = 0
	for _, value := range services {
		if !value.StorageEnabled {
			continue
		}

		if isSummary {
			if value.PersistenceLatencyAverageTotal == 0 {
				averageAverageLatency = 0
			} else {
				averageAverageLatency = value.PersistenceLatencyAverageTotal / float64(value.StorageEnabledCount)
			}
		} else {
			// not a summary so just set the averageAverageLatency to the latency value
			averageAverageLatency = value.PersistenceLatencyAverage
		}
		totalActiveSpaceUsed += value.PersistenceActiveSpaceUsed

		if isSummary {
			header = getColumns(value.ServiceName, formatSmallInteger(value.StorageEnabledCount))
		} else {
			header = getColumns(value.NodeID)
		}

		header = getColumns(header, value.PersistenceMode,
			formatLargeInteger(max(0, value.PersistenceActiveSpaceUsed)),
			formatMB(int32(max(0, value.PersistenceActiveSpaceUsed)/1024/1024)),
			formatLatency(float32(averageAverageLatency)),
			formatLargeInteger(max(value.PersistenceLatencyMax, 0))+"ms")

		if isSummary {
			header = getColumns(header, formatSmallInteger(int32(len(value.Snapshots))), value.OperationStatus)
		}

		stringValues[i+1] = header
		i++
	}

	return fmt.Sprintf("Total Active Space Used: %s\n\n", formatMB(int32(max(totalActiveSpaceUsed, 0)/1024/1024))) +
		formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// FormatSnapshots returns the snapshots in a formatted output
func FormatSnapshots(serviceSnapshots []config.Snapshots, archived bool) string {
	var (
		snapshotLen    = len(serviceSnapshots)
		snapshotHeader = "SNAPSHOT NAME"
	)
	if snapshotLen == 0 {
		return ""
	}

	sort.Slice(serviceSnapshots, func(p, q int) bool {
		return strings.Compare(serviceSnapshots[p].ServiceName, serviceSnapshots[q].ServiceName) > 0
	})

	var stringValues = make([]string, 0)

	if archived {
		snapshotHeader = "ARCHIVED " + snapshotHeader
	}
	stringValues = append(stringValues, getColumns(ServiceColumn, snapshotHeader))

	for _, service := range serviceSnapshots {
		snapshots := service.Snapshots
		sort.Slice(snapshots, func(p, q int) bool {
			return strings.Compare(snapshots[p], snapshots[q]) > 0
		})
		for _, value := range snapshots {
			stringValues = append(stringValues, getColumns(service.ServiceName, value))
		}

	}
	return formatLinesAllStrings(stringValues)
}

// FormatProxyServers returns the proxy servers' information in a column formatted output
// protocol is either tcp or http and will display a different format based upon this
func FormatProxyServers(services []config.ProxySummary, protocol string) string {
	// get the number of proxies matching the protocol
	var (
		serviceCount = 0
		alignment    []string
	)
	for _, value := range services {
		if protocol == value.Protocol {
			serviceCount++
		}
	}

	if serviceCount == 0 {
		return ""
	}

	var stringValues = make([]string, serviceCount+1)

	sort.Slice(services, func(p, q int) bool {
		if services[p].ServiceName == services[q].ServiceName {
			nodeID1, _ := strconv.Atoi(services[p].NodeID)
			nodeID2, _ := strconv.Atoi(services[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) > 0
	})

	// common header
	stringValues[0] = getColumns(NodeIDColumn, "HOST IP", ServiceNameColumn)

	if protocol == "tcp" {
		stringValues[0] = getColumns(stringValues[0], "CONNECTIONS", "BYTES SENT", "BYTES REC")
		if OutputFormat == constants.WIDE {
			stringValues[0] = getColumns(stringValues[0],
				"MSG SENT", "MSG RCV", "BYTES BACKLOG", "MSG BACKLOG", "UNAUTH")
			alignment = []string{L, L, L, R, R, R, R, R, R, R, R}
		} else {
			alignment = []string{L, L, L, R, R, R}
		}

	} else {
		stringValues[0] = getColumns(stringValues[0], "SERVER TYPE", "REQUESTS", "ERRORS")
		if OutputFormat == constants.WIDE {
			stringValues[0] = getColumns(stringValues[0], "1xx", "2xx", "3xx", "4xx", "5xx")
			alignment = []string{L, L, L, L, R, R, R, R, R, R, R}
		} else {
			alignment = []string{L, L, L, L, R, R}
		}
	}

	i := 0
	for _, value := range services {
		if protocol != value.Protocol {
			continue
		}
		// common values
		stringValues[i+1] = getColumns(value.NodeID, value.HostIP, value.ServiceName)

		if protocol == "tcp" {
			stringValues[i+1] = getColumns(stringValues[i+1], formatLargeInteger(value.ConnectionCount),
				formatLargeInteger(value.TotalBytesSent), formatLargeInteger(value.TotalBytesReceived))
			if OutputFormat == constants.WIDE {
				stringValues[i+1] = getColumns(stringValues[i+1], formatLargeInteger(value.TotalMessagesSent),
					formatLargeInteger(value.TotalMessagesReceived), formatLargeInteger(value.OutgoingByteBacklog),
					formatLargeInteger(value.OutgoingMessageBacklog), formatLargeInteger(value.UnAuthConnectionAttempts))
			}
		} else {
			stringValues[i+1] = getColumns(stringValues[i+1], value.HTTPServerType,
				formatLargeInteger(value.TotalRequestCount), formatLargeInteger(value.TotalErrorCount))
			if OutputFormat == constants.WIDE {
				stringValues[i+1] = getColumns(stringValues[i+1], formatLargeInteger(value.ResponseCount1xx),
					formatLargeInteger(value.ResponseCount2xx), formatLargeInteger(value.ResponseCount3xx),
					formatLargeInteger(value.ResponseCount4xx), formatLargeInteger(value.ResponseCount5xx))
			}
		}
		i++
	}

	return formatLinesAllStringsWithAlignment(alignment, stringValues)
}

// formatLinesAllStrings outputs the array of strings (which contain headers)
// as formatted fixed width columns adjusted for the max size of the data elements
func formatLinesAllStrings(stringValues []string) string {
	return formatLinesAllStringsWithAlignment(make([]string, 0), stringValues)
}

// formatLinesAllStringsWithAlignmentMax outputs the array of strings (which contain headers)
// as formatted fixed width columns adjusted for the max size of the data elements
// the alignment slice, if present, contains either L, R to indicate the alignment of the columns
// this variant will truncate the values to max len
func formatLinesAllStringsWithAlignmentMax(alignment, stringValues []string, maxLen int) string {
	// retrieve max column lengths
	var (
		columnLengths = getMaxColumnLengths(stringValues)
		numberColumns = len(columnLengths)
		alignmentLen  = len(alignment)
		hasAlignments = alignmentLen > 0
		align         string
		truncate      = make([]bool, numberColumns)
	)

	// check if any columns > max len and >= 10 as why bother...
	if maxLen > 0 {
		for i, value := range columnLengths {
			if value >= 10 && value > maxLen {
				truncate[i] = true
				columnLengths[i] = maxLen
			}
		}
	}

	// silently turn off alignments if the values don't match
	if hasAlignments && numberColumns != alignmentLen {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: number of columns: %d, alignment length: %d\n",
			numberColumns, alignmentLen)
		hasAlignments = false
	}

	// create an array of string formats only once to use throughout
	var stringFormats = make([]string, numberColumns)
	for i := range stringFormats {
		align = fmt.Sprintf("%%-%ds", columnLengths[i]) // default to left
		if hasAlignments && alignment[i] == R {
			// align right
			align = fmt.Sprintf("%%%ds", columnLengths[i])
		}
		stringFormats[i] = align
	}

	var sb strings.Builder

	for _, value := range stringValues {
		if value == "" {
			continue
		}
		// split the values
		var entry = strings.Split(value, sep)

		// format each individual field
		for i, e := range entry {
			actualValue := fmt.Sprintf(stringFormats[i], e)
			if truncate[i] && len(actualValue) > columnLengths[i] {
				// truncate the value to max len -3 and append three ...
				actualValue = actualValue[:maxLen-3] + "..."
			}
			sb.WriteString(actualValue)
			if i < numberColumns-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatLinesAllStringsWithAlignment outputs the array of strings (which contain headers)
// as formatted fixed width columns adjusted for the max size of the data elements
// the alignment slice, if present, contains either L, R to indicate the alignment of the columns
func formatLinesAllStringsWithAlignment(alignment, stringValues []string) string {
	return formatLinesAllStringsWithAlignmentMax(alignment, stringValues, 0)
}

// maxInt returns the maximum of two values
func max(v1 int64, v2 int64) int64 {
	if v1 >= v2 {
		return v1
	}
	return v2
}

// formatSmallInteger formats a small integer
func formatSmallInteger(value int32) string {
	return printer.Sprintf("%d", value)
}

// formatPort formats a small integer with a max length
func formatPort(value int32) string {
	return fmt.Sprintf("%d", value)
}

// formatLargeInteger formats a large integer
func formatLargeInteger(value int64) string {
	return printer.Sprintf("%d", value)
}

// formatFloat formats a small float
func formatFloat(value float32) string {
	return printer.Sprintf("%.4f", value)
}

// formatLargeFloat formats a large float
func formatLargeFloat(value float64) string {
	return printer.Sprintf("%.4f", value)
}

// formatLatency formats a float latency
func formatLatency(value float32) string {
	return printer.Sprintf("%.3fms", value)
}

// formatLatency formats a float latency
func formatLatency0(value float32) string {
	return printer.Sprintf("%.0fms", value)
}

// formatMbps formats a Mbps
func formatMbps(value float32) string {
	return printer.Sprintf("%.1fMbps", value)
}

// formatPublisherReceiver formats a packet publisher/ receiver
func formatPublisherReceiver(value float32) string {
	return printer.Sprintf("%-.3f", value)
}

// formatPercent formats a percent value
func formatPercent(value float64) string {
	if value == -1 {
		return "n/a"
	}
	return strings.TrimSpace(printer.Sprintf("%6.2f%%", value*100))
}

func formatMB(memoryMB int32) string {
	var memory = float32(memoryMB)
	if memory < 1024 {
		return printer.Sprintf("%-.0fMB", memory)
	}

	memory /= 1024
	if memory < 1024 {
		return printer.Sprintf("%-.3fGB", memory)
	}

	memory /= 1024
	return printer.Sprintf("%-.3fTB", memory)
}

// getMaxColumnLengths returns an array representing the max lengths of columns
// delimited with the sep
func getMaxColumnLengths(values []string) []int {
	if len(values) == 0 {
		return make([]int, 0)
	}
	// find the number of values from the first entry
	var splits = strings.Split(values[0], sep)
	var numValues = len(splits)

	var lengths = make([]int, numValues)

	for _, value := range values {
		for j, entry := range strings.Split(value, sep) {
			if len(entry) > lengths[j] {
				lengths[j] = len(entry)
			}
		}
	}
	return lengths
}

// CreateCamelCaseLabel creates a camel case label from a field, e.g.
// unicastListener becomes "Unicast Listener"
func CreateCamelCaseLabel(field string) string {
	// special cases
	if field == "UID" {
		return "UID"
	}
	if field == "UUID" {
		return "UUID"
	}
	if field == "statusHA" {
		return "Status HA"
	}
	if field == "HAStatus" {
		return "HA Status"
	}
	if field == "HATarget" {
		return "HA Target"
	}
	if field == "HAStatusCode" {
		return "HA Status Code"
	}
	var sb strings.Builder
	if len(field) == 0 {
		return ""
	}

	var data = []rune(field)
	var length = len(field)
	var skip = 0

	for i, c := range data {
		if skip > 0 {
			skip--
			continue
		}
		if i == 0 {
			// change to uppercase
			sb.WriteString(strings.ToUpper(string(c)))
		} else {
			// check if uppercase and add space if the next char is not uppercase too
			if unicode.IsUpper(c) {
				sb.WriteString(" ")
				// check if MB (special case)
				if c == 'M' && i < length-1 && data[i+1] == 'B' {
					sb.WriteString("MB")
					skip = 1
				} else if c == 'K' && i < length-1 && data[i+1] == 'B' {
					sb.WriteString("KB")
					skip = 1
				} else if c == 'T' && i < length-2 && data[i+1] == 'T' && data[i+2] == 'L' {
					sb.WriteString("TTL")
					skip = 2
				} else {
					sb.WriteString(string(c))
				}
			} else {
				sb.WriteString(string(c))
			}
		}
	}
	return sb.String()
}

// findKeyValueIndex finds the index where the key matches
func findKeyValueIndex(keyValues []KeyValues, column string) int {
	for i, v := range keyValues {
		if v.Key == column {
			return i
		}
	}
	return -1
}

// appendColumnValue appends a column value taking into account if it breaks over multiple lines
func appendColumnValue(v KeyValues, sb *strings.Builder, keyFormat string) {
	value := fmt.Sprintf("%v", v.Value)
	if strings.Contains(value, "\n") {
		// remove newline at beginning
		if strings.Index(value, "\n") == 0 {
			value = value[1:]
		}
		// if the string contains a newline then pad the beginning of each line
		for i, str := range strings.Split(value, "\n") {
			if i == 0 {
				sb.WriteString(fmt.Sprintf(keyFormat, v.Key, str))
			} else {
				sb.WriteString(fmt.Sprintf(keyFormat, "", str))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf(keyFormat, v.Key, value))
	}
}

// getColumns returns all the values separated by sep string
func getColumns(values ...string) string {
	var (
		length = len(values)
		sb     = strings.Builder{}
	)

	for i, value := range values {
		sb.WriteString(value)
		if i < length-1 {
			sb.WriteString(sep)
		}
	}

	return sb.String()
}
