/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
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

const (
	sep                   = "~~"
	R                     = "R"
	L                     = "L"
	NodeIDColumn          = "NODE ID"
	SubscriberIDColumn    = "SUBSCRIBER ID"
	ServiceColumn         = "SERVICE"
	CacheColumn           = "CACHE"
	PolledColumn          = "POLLED"
	HeadColumn            = "HEAD"
	ChannelsColumn        = "CHANNELS"
	ChannelColumn         = "CHANNEL"
	SubscriberGroupColumn = "SUBSCRIBER GROUP"
	PublishedColumn       = "PUBLISHED"
	MeanColumn            = "MEAN"
	OneMinuteColumn       = "1 MIN"
	FiveMinuteColumn      = "5 MIN"
	FifteenMinuteColumn   = "15 MIN"
	ServiceNameColumn     = "SERVICE NAME"
	AddressColumn         = "ADDRESS"
	PortColumn            = "PORT"
	MemberColumn          = "MEMBER"
	MembersColumn         = "MEMBERS"
	RoleColumn            = "ROLE"
	ProcessColumn         = "PROCESS"
	MaxHeapColumn         = "MAX HEAP"
	UsedHeapColumn        = "USED HEAP"
	AvailHeapColumn       = "AVAIL HEAP"
	NameColumn            = "NAME"
	avgSize               = "AVG SIZE"
	avgApply              = "AVG APPLY"
	avgBacklogDelay       = "AVG BACKLOG DELAY"
	partitions            = "PARTITIONS"
	tcp                   = "tcp"
	na                    = "n/a"
	endangered            = "ENDANGERED"
	dataSent              = "DATA SENT"
	dataRec               = "DATA REC"
)

var (
	KB int64 = 1024
	MB       = KB * KB
	GB       = MB * MB
)

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

func FormatFederationDetails(federationDetails []config.FederationDescription, target string) string {
	var (
		fedCount           = len(federationDetails)
		finalAlignment     []string
		suffix             = "SENT"
		formattingFunction = getFormattingFunction()
		table              FormattedTable
	)

	if fedCount == 0 {
		return ""
	}

	sort.Slice(federationDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(federationDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(federationDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	if OutputFormat == constants.TABLE {
		if target == destinations {
			finalAlignment = []string{R, L, R, R, R, R}
		} else {
			finalAlignment = []string{R, R, R, R, R}
		}
	} else { // WIDE
		if target == destinations {
			finalAlignment = []string{R, L, R, R, R, R, R, R, R, R, R, R, R}
		} else {
			finalAlignment = []string{R, R, R, R, R, R, R}
		}
	}

	table = newFormattedTable().WithAlignment(finalAlignment...)

	if target == destinations {
		table.WithHeader(NodeIDColumn, "STATE", "DATA "+suffix, "MSG "+suffix, "REC "+suffix, "CURR BWIDTH")
		table.AddFormattingFunction(1, federationStateFormatter)
	} else {
		suffix = "REC"
		table.WithHeader(NodeIDColumn, "CONNECTED", "DATA "+suffix, "MSG "+suffix, "REC "+suffix)
	}

	if OutputFormat == constants.WIDE {
		if target == destinations {
			table.AddHeaderColumns(avgApply, "AVG ROUND TRIP", avgBacklogDelay, "REPLICATE",
				partitions, "ERRORS", "UNACKED")
			table.AddFormattingFunction(11, errorFormatter)
			table.AddFormattingFunction(12, errorFormatter)
		} else {
			table.AddHeaderColumns(avgApply, avgBacklogDelay)
		}
	}

	var (
		bytes     int64
		messages  int64
		records   int64
		bandwidth string
	)

	for _, value := range federationDetails {
		var nodeID, _ = strconv.Atoi(value.NodeID)
		table.AddRow(formatSmallInteger(int32(nodeID)))

		if target == "destinations" {
			bytes = value.TotalBytesSent
			messages = value.TotalMsgSent
			records = value.TotalRecordsSent
			bandwidth = formatMbps(float32(value.CurrentBandwidth))
		} else {
			bytes = value.TotalBytesReceived
			messages = value.TotalMsgReceived
			records = value.TotalRecordsReceived
			bandwidth = na
		}

		if target == destinations {
			table.AddColumnsToRow(value.State,
				formattingFunction(bytes), formatLargeInteger(messages),
				formatLargeInteger(records), bandwidth)
		} else {
			table.AddColumnsToRow(formatSmallInteger(value.CurrentConnectionCount),
				formattingFunction(bytes), formatLargeInteger(messages),
				formatLargeInteger(records))
		}

		if OutputFormat == constants.WIDE {
			if target == destinations {
				table.AddColumnsToRow(
					formatLatency0(float32(value.MsgApplyTimePercentileMillis)),
					formatLatency0(float32(value.MsgNetworkRoundTripTimePercentileMillis)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis)),
					formatPercent(float64(value.ReplicateAllPercentComplete)/100.0),
					formatLargeInteger(value.ReplicateAllPartitionCount),
					formatLargeInteger(value.ReplicateAllPartitionErrorCount),
					formatLargeInteger(value.TotalReplicateAllPartitionsUnacked),
				)
			} else {
				table.AddColumnsToRow(
					formatLatency0(float32(value.MsgApplyTimePercentileMillis)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis)))
			}
		}
	}

	return table.String()
}

// FormatFederationSummary returns the federation summary in column formatted output
// the target may be destinations or origins and columns will change slightly
func FormatFederationSummary(federationSummaries []config.FederationSummary, target string) string {
	var (
		fedCount           = len(federationSummaries)
		finalAlignment     []string
		suffix             = "SENT"
		participantCol     = "DESTINATION"
		memberCol          = MembersColumn
		formattingFunction = getFormattingFunction()
		table              FormattedTable
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
			finalAlignment = []string{L, L, R, R, R, R}
		}
	} else { // WIDE
		if target == destinations {
			finalAlignment = []string{L, L, R, L, R, R, R, R, R, R, R, R, R, R, R}
		} else {
			finalAlignment = []string{L, L, R, R, R, R, R, R}
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

	table = newFormattedTable().WithAlignment(finalAlignment...)

	if target == destinations {
		table.WithHeader(ServiceColumn, participantCol, memberCol, "STATES", "DATA "+suffix,
			"MSG "+suffix, "REC "+suffix, "CURR AVG BWIDTH")
		table.AddFormattingFunction(3, federationStateFormatter)
	} else {
		table.WithHeader(ServiceColumn, participantCol, memberCol, "DATA "+suffix,
			"MSG "+suffix, "REC "+suffix)
	}

	if OutputFormat == constants.WIDE {
		if target == destinations {
			table.AddHeaderColumns(avgApply, "AVG ROUND TRIP", avgBacklogDelay, "REPLICATE",
				partitions, "ERRORS", "UNACKED")
			table.AddFormattingFunction(13, errorFormatter)
			table.AddFormattingFunction(14, errorFormatter)
		} else {
			table.AddHeaderColumns(avgApply, avgBacklogDelay)
		}
	}

	var (
		bytes     float64
		messages  float64
		records   float64
		members   int32
		bandwidth string
	)

	for _, value := range federationSummaries {
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
			bandwidth = na
		}

		if target == destinations {
			table.AddRow(value.ServiceName, value.ParticipantName,
				formatSmallInteger(members), fmt.Sprintf("%v", utils.GetUniqueValues(value.State)),
				formattingFunction(int64(bytes)), formatLargeInteger(int64(messages)),
				formatLargeInteger(int64(records)), bandwidth)
		} else {
			table.AddRow(value.ServiceName, value.ParticipantName,
				formatSmallInteger(members),
				formattingFunction(int64(bytes)), formatLargeInteger(int64(messages)),
				formatLargeInteger(int64(records)))
		}

		if OutputFormat == constants.WIDE {
			if target == destinations {
				table.AddColumnsToRow(
					formatLatency0(float32(value.MsgApplyTimePercentileMillis.Average)),
					formatLatency0(float32(value.MsgNetworkRoundTripTimePercentileMillis.Average)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis.Average)),
					formatPercent(value.ReplicateAllPercentComplete.Average/100),
					formatLargeInteger(int64(value.ReplicateAllPartitionCount.Sum)),
					formatLargeInteger(int64(value.ReplicateAllPartitionErrorCount.Sum)),
					formatLargeInteger(int64(value.TotalReplicateAllPartitionsUnacked.Sum)),
				)
			} else {
				table.AddColumnsToRow(
					formatLatency0(float32(value.MsgApplyTimePercentileMillis.Average)),
					formatLatency0(float32(value.RecordBacklogDelayTimePercentileMillis.Average)))
			}
		}
	}

	return table.String()
}

// FormatCacheSummary returns the cache summary in column formatted output
func FormatCacheSummary(cacheSummaries []config.CacheSummaryDetail) string {
	var (
		cacheCount         = len(cacheSummaries)
		finalAlignment     []string
		formattingFunction = getFormattingFunction()
	)

	if cacheCount == 0 {
		return ""
	}

	if OutputFormat == constants.TABLE {
		finalAlignment = []string{L, L, R, R}
	} else {
		finalAlignment = []string{L, L, R, R, R, R, R, R, R, R, R}
	}

	table := newFormattedTable().WithAlignment(finalAlignment...)

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
	var totalUnits int64

	table.WithHeader(ServiceColumn, CacheColumn, "COUNT", "SIZE")

	if OutputFormat == constants.WIDE {
		table.AddHeaderColumns(avgSize,
			"TOTAL PUTS", "TOTAL GETS", "TOTAL REMOVES", "TOTAL HITS", "TOTAL MISSES", "HIT PROB")
		table.AddFormattingFunction(10, hitRateFormatter)
	}

	for _, value := range cacheSummaries {
		var (
			hitProb     = 0.0
			averageSize int64
		)
		totalGets := value.TotalGets
		totalHits := value.CacheHits
		if totalGets != 0 {
			hitProb = float64(totalHits) / float64(totalGets)
		}
		totalUnits += value.UnitsBytes

		if value.CacheSize != 0 {
			averageSize = value.UnitsBytes / int64(value.CacheSize)
		}

		table.AddRow(value.ServiceName, value.CacheName, formatSmallInteger(value.CacheSize),
			formattingFunction(value.UnitsBytes))

		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(formatLargeInteger(averageSize),
				formatLargeInteger(value.TotalPuts), formatLargeInteger(totalGets),
				formatLargeInteger(value.TotalRemoves), formatLargeInteger(totalHits),
				formatLargeInteger(value.CacheMisses), formatPercent(hitProb))
		}
	}

	return fmt.Sprintf("Total Caches: %d, Total primary storage: %s\n\n", totalCaches,
		strings.TrimSpace(formattingFunction(totalUnits))) + table.String()
}

// FormatTopicsSummary returns the topics summary in column formatted output
func FormatTopicsSummary(topicDetails []config.TopicDetail) string {
	var (
		cacheCount = len(topicDetails)
	)
	if cacheCount == 0 {
		return ""
	}

	sort.Slice(topicDetails, func(p, q int) bool {
		if topicDetails[p].ServiceName < topicDetails[q].ServiceName {
			return true
		} else if topicDetails[p].ServiceName > topicDetails[q].ServiceName {
			return false
		} else {
			return topicDetails[p].TopicName < topicDetails[q].TopicName
		}
	})

	table := newFormattedTable().WithAlignment(L, L, R, R, R, R).
		WithHeader(ServiceColumn, "TOPIC", MembersColumn, ChannelsColumn, "SUBSCRIBERS", PublishedColumn)

	for _, value := range topicDetails {
		table.AddRow(value.ServiceName, value.TopicName, formatLargeInteger(value.Members),
			formatLargeInteger(value.Channels), formatLargeInteger(value.Subscribers), formatLargeInteger(value.PublishedCount))
	}

	return table.String()
}

// FormatTopicsSubscribers returns the topics subscriber details in column formatted output
func FormatTopicsSubscribers(topicsSubscribers []config.TopicsSubscriberDetail) string {
	var (
		memberCount = len(topicsSubscribers)
	)
	if memberCount == 0 {
		return ""
	}

	sort.Slice(topicsSubscribers, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(topicsSubscribers[p].NodeID)
		nodeID2, _ := strconv.Atoi(topicsSubscribers[q].NodeID)
		if nodeID1 == nodeID2 {
			return topicsSubscribers[p].ID < topicsSubscribers[q].ID
		}
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, SubscriberIDColumn, "STATE", ChannelsColumn, SubscriberGroupColumn,
		"RECEIVED", "ERRORS", "BACKLOG", "DISCONNECTS", "TYPE")
	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, R, L, R, L, R, R, R, R, L, L)
		table.AddHeaderColumns(MemberColumn)
	} else {
		table.WithAlignment(R, R, L, R, L, R, R, R, R, L)
	}
	table.AddFormattingFunction(6, errorFormatter)
	table.AddFormattingFunction(7, errorFormatter)
	table.AddFormattingFunction(8, errorFormatter)

	for _, value := range topicsSubscribers {
		var nodeID, _ = strconv.Atoi(value.NodeID)
		subGroup := value.SubscriberGroup
		if value.SubType == "Anonymous" {
			subGroup = "n/a"
		}

		table.AddRow(formatSmallInteger(int32(nodeID)), fmt.Sprintf("%v", value.ID),
			value.StateName, formatLargeInteger(value.ChannelCount), subGroup, formatLargeInteger(value.ReceivedCount),
			formatLargeInteger(value.ReceiveErrors), formatLargeInteger(value.Backlog), formatLargeInteger(value.Disconnections), value.SubType)
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(value.Member)
		}
	}

	return table.String()
}

// FormatTopicsSubscriberGroups returns the topics subscriber groups details in column formatted output
func FormatTopicsSubscriberGroups(subscriberGroups []config.TopicsSubscriberGroupDetail) string {
	var (
		count = len(subscriberGroups)
	)
	if count == 0 {
		return ""
	}

	sort.Slice(subscriberGroups, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(subscriberGroups[p].NodeID)
		nodeID2, _ := strconv.Atoi(subscriberGroups[q].NodeID)
		if subscriberGroups[p].SubscriberGroup == subscriberGroups[q].SubscriberGroup {
			return nodeID1 < nodeID2
		}
		return strings.Compare(subscriberGroups[p].SubscriberGroup, subscriberGroups[q].SubscriberGroup) < 0
	})

	table := newFormattedTable().WithHeader(SubscriberGroupColumn, NodeIDColumn, ChannelsColumn, PolledColumn, MeanColumn,
		OneMinuteColumn, FiveMinuteColumn, FifteenMinuteColumn).WithAlignment(L, R, R, R, R, R, R, R)

	for _, value := range subscriberGroups {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		table.AddRow(value.SubscriberGroup, formatSmallInteger(int32(nodeID)),
			formatLargeInteger(value.ChannelCount), formatLargeInteger(value.PolledCount),
			formatLargeFloat(value.PolledMeanRate), formatLargeFloat(value.PolledOneMinuteRate),
			formatLargeFloat(value.PolledFiveMinuteRate), formatLargeFloat(value.PolledFifteenMinuteRate))
	}

	return table.String()
}

// FormatTopicsMembers returns the topics member details in column formatted output
func FormatTopicsMembers(topicsMembers []config.TopicsMemberDetail) string {
	var (
		memberCount = len(topicsMembers)
	)
	if memberCount == 0 {
		return ""
	}

	sort.Slice(topicsMembers, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(topicsMembers[p].NodeID)
		nodeID2, _ := strconv.Atoi(topicsMembers[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, ChannelsColumn, PublishedColumn, MeanColumn, OneMinuteColumn,
		FiveMinuteColumn, FifteenMinuteColumn)
	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, R, R, R, R, R, R, R, R, R, R)
		table.AddHeaderColumns("SUB TIMEOUT", "RECON TIMEOUT", "WAIT", "PAGE CAPACITY")
	} else {
		table.WithAlignment(R, R, R, R, R, R, R)
	}

	for _, value := range topicsMembers {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		table.AddRow(formatSmallInteger(int32(nodeID)), formatLargeInteger(value.ChannelCount),
			formatLargeInteger(value.PublishedCount), formatLargeFloat(value.PublishedMeanRate),
			formatLargeFloat(value.PublishedOneMinuteRate), formatLargeFloat(value.PublishedFiveMinuteRate),
			formatLargeFloat(value.PublishedFifteenMinuteRate))
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(formatLargeInteger(value.SubscriberTimeout)+"ms",
				formatLargeInteger(value.ReconnectTimeout)+"ms", formatLargeInteger(value.ReconnectWait)+"ms",
				formatLargeInteger(value.PageCapacity))
		}
	}

	return table.String()
}

// FormatChannelStats returns the channel stats in column formatted output
func FormatChannelStats(channelStats []config.ChannelStats) string {
	var memberCount = len(channelStats)

	if memberCount == 0 {
		return ""
	}

	sort.Slice(channelStats, func(p, q int) bool {
		return channelStats[p].Channel < channelStats[q].Channel
	})

	table := newFormattedTable().WithHeader(ChannelColumn, PublishedColumn, MeanColumn, OneMinuteColumn,
		FiveMinuteColumn, FifteenMinuteColumn, "TAIL").WithAlignment(R, R, R, R, R, R, L)

	for _, value := range channelStats {
		table.AddRow(formatLargeInteger(value.Channel),
			formatLargeInteger(value.PublishedCount), formatLargeFloat(value.PublishedMeanRate),
			formatLargeFloat(value.PublishedOneMinuteRate), formatLargeFloat(value.PublishedFiveMinuteRate),
			formatLargeFloat(value.PublishedFifteenMinuteRate), value.Tail)
	}

	return table.String()
}

// FormatSubscriberChannelStats returns the subscriber channel stats in column formatted output
func FormatSubscriberChannelStats(channelStats []config.SubscriberChannelStats) string {
	var memberCount = len(channelStats)

	if memberCount == 0 {
		return ""
	}

	sort.Slice(channelStats, func(p, q int) bool {
		return channelStats[p].Channel < channelStats[q].Channel
	})

	table := newFormattedTable().WithHeader(ChannelColumn, "EMPTY", "LAST COMMIT",
		"LAST REC", "OWNED", HeadColumn).WithAlignment(R, L, L, L, L, L)

	for _, value := range channelStats {
		table.AddRow(formatLargeInteger(value.Channel),
			formatBool(value.Empty), value.LastCommit, value.LastReceived, formatBool(value.Owned), value.Head)
	}

	return table.String()
}

// FormatHeadsStats returns the subscriber heads stats in column formatted output
func FormatHeadsStats(channelStats []config.HeadStats) string {
	var memberCount = len(channelStats)

	if memberCount == 0 {
		return ""
	}

	sort.Slice(channelStats, func(p, q int) bool {
		return channelStats[p].Channel < channelStats[q].Channel
	})

	table := newFormattedTable().WithHeader(ChannelColumn, "POSITION").WithAlignment(R, L)

	for _, value := range channelStats {
		table.AddRow(formatLargeInteger(value.Channel), value.Position)
	}

	return table.String()
}

// FormatSubscriberGroupChannelStats returns the subscriber channel stats in column formatted output
func FormatSubscriberGroupChannelStats(channelStats []config.SubscriberGroupChannelStats) string {
	var (
		memberCount = len(channelStats)
	)

	if memberCount == 0 {
		return ""
	}

	sort.Slice(channelStats, func(p, q int) bool {
		return channelStats[p].Channel < channelStats[q].Channel
	})

	table := newFormattedTable().WithHeader(ChannelColumn, "OWNING SUB", MemberColumn, PolledColumn, MeanColumn,
		OneMinuteColumn, FiveMinuteColumn, FifteenMinuteColumn, HeadColumn)
	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, R, R, R, R, R, R, R, L, L, L, L)
		table.AddHeaderColumns("LAST COMMIT", "LAST TIMESTAMP", "LAST POLLED")
	} else {
		table.WithAlignment(R, R, R, R, R, R, R, R, L)
	}

	for _, value := range channelStats {
		table.AddRow(formatLargeInteger(value.Channel),
			fmt.Sprintf("%v", value.OwningSubscriberID), formatLargeInteger(value.OwningSubscriberMemberID),
			formatLargeInteger(value.PolledCount), formatLargeFloat(value.PolledMeanRate),
			formatLargeFloat(value.PolledOneMinuteRate), formatLargeFloat(value.PolledFiveMinuteRate),
			formatLargeFloat(value.PolledFifteenMinuteRate), value.Head)
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(value.LastCommittedPosition, value.LastCommittedTimestamp,
				value.LastPolledTimestamp)
		}
	}

	return table.String()
}

// FormatServiceMembers returns the service member details in column formatted output
func FormatServiceMembers(serviceMembers []config.ServiceMemberDetail) string {
	var (
		memberCount = len(serviceMembers)
	)
	if memberCount == 0 {
		return ""
	}

	sort.Slice(serviceMembers, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(serviceMembers[p].NodeID)
		nodeID2, _ := strconv.Atoi(serviceMembers[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "THREADS", "IDLE", "THREAD UTIL", "MIN THREADS", "MAX THREADS")
	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, R, R, R, R, R, R, R, R, R, R, R)
		table.AddHeaderColumns("TASK COUNT", "TASK BACKLOG", "PRIMARY OWNED",
			"BACKUP OWNED", "REQ AVG MS", "TASK AVG MS")
	} else {
		table.WithAlignment(R, R, R, R, R, R)
	}

	for _, value := range serviceMembers {
		var (
			nodeID, _           = strconv.Atoi(value.NodeID)
			utilization float64 = -1
		)

		if value.ThreadCount > 0 {
			utilization = float64(value.ThreadCount-value.ThreadIdleCount) / float64(value.ThreadCount)
		}
		table.AddRow(formatSmallInteger(int32(nodeID)), formatSmallInteger(value.ThreadCount),
			formatSmallInteger(value.ThreadIdleCount), formatPercent(utilization),
			formatSmallInteger(value.ThreadCountMin), formatSmallInteger(value.ThreadCountMax))
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(
				formatSmallInteger(value.TaskCount), formatSmallInteger(value.TaskBacklog),
				formatSmallInteger(value.OwnedPartitionsPrimary), formatSmallInteger(value.OwnedPartitionsBackup),
				formatFloat(value.RequestAverageDuration), formatFloat(value.TaskAverageDuration))
		}
	}

	return table.String()
}

// FormatCacheDetailsSizeAndAccess returns the cache details size and access details in column formatted output
func FormatCacheDetailsSizeAndAccess(cacheDetails []config.CacheDetail) string {
	var (
		detailsCount       = len(cacheDetails)
		formattingFunction = getFormattingFunction()
	)

	if detailsCount == 0 {
		return ""
	}

	sort.Slice(cacheDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(cacheDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(cacheDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "TIER", "COUNT", "SIZE",
		"TOTAL PUTS", "TOTAL GETS", "TOTAL REMOVES")
	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, L, R, R, R, R, R, R, R, R, R, R, R)
		table.AddHeaderColumns("HITS", "MISSES", "HIT PROB", "STORE READS",
			"WRITES", "FAILURES")
	} else {
		table.WithAlignment(R, L, R, R, R, R, R)
	}

	for _, value := range cacheDetails {
		var (
			nodeID, _  = strconv.Atoi(value.NodeID)
			hitProb    = 0.0
			unitsBytes = value.Units * value.UnitFactor
		)
		totalGets := value.TotalGets
		totalHits := value.CacheHits
		if totalGets != 0 {
			hitProb = float64(totalHits) / float64(totalGets)
		}

		table.AddRow(formatSmallInteger(int32(nodeID)), value.Tier,
			formatSmallInteger(value.CacheSize), formattingFunction(unitsBytes),
			formatLargeInteger(value.TotalPuts),
			formatLargeInteger(totalGets), formatLargeInteger(value.TotalRemoves))
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(formatLargeInteger(totalHits),
				formatLargeInteger(value.CacheMisses), formatPercent(hitProb),
				formatLargeInteger(value.StoreReads), formatLargeInteger(value.StoreWrites),
				formatLargeInteger(value.StoreFailures))
		}
	}

	return table.String()
}

// FormatCacheIndexDetails returns the cache index details
func FormatCacheIndexDetails(cacheDetails []config.CacheDetail) string {
	var (
		sb                  = strings.Builder{}
		totalIndexUnits     int64
		totalIndexingMillis int64
		formattingFunction  = getFormattingFunction()
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
		"Total Indexing:        " + formattingFunction(totalIndexUnits) + "\n" +
		"Total Indexing Millis: " + formatLargeInteger(totalIndexingMillis) + "\n" +
		"\n" + sb.String()
}

// FormatCacheDetailsStorage returns the cache storage details in column formatted output
func FormatCacheDetailsStorage(cacheDetails []config.CacheDetail) string {
	var (
		detailsCount       = len(cacheDetails)
		formattingFunction = getFormattingFunction()
	)
	if detailsCount == 0 {
		return ""
	}

	sort.Slice(cacheDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(cacheDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(cacheDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "TIER", "LOCKS GRANTED", "LOCKS PENDING", "KEY LISTENERS",
		"FILTER LISTENERS", "MAX QUERY MS", "MAX QUERY DESC").MaxLength(40)

	if OutputFormat == constants.WIDE {
		table.AddHeaderColumns("NO OPT AVG", "OPT AVG", "INDEX SIZE", "INDEXING MILLIS")
		table.WithAlignment(R, L, R, R, R, R, R, L, R, R, R, R)
	} else {
		table.WithAlignment(R, L, R, R, R, R, R, L)
	}

	for _, value := range cacheDetails {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		table.AddRow(formatSmallInteger(int32(nodeID)), value.Tier,
			formatLargeInteger(value.LocksGranted), formatLargeInteger(value.LocksPending),
			formatLargeInteger(value.ListenerKeyCount), formatLargeInteger(value.ListenerFilterCount),
			formatLargeInteger(value.MaxQueryDurationMillis), value.MaxQueryDescription)
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(formatFloat(float32(value.NonOptimizedQueryAverageMillis)),
				formatFloat(float32(value.OptimizedQueryAverageMillis)),
				formattingFunction(value.IndexTotalUnits), formatLargeInteger(value.IndexingTotalMillis))
		}
	}

	return table.String()
}

// FormatCacheStoreDetails returns the cache store details in column formatted output
func FormatCacheStoreDetails(cacheDetails []config.CacheStoreDetail, cache, service string, includeHeader bool) string {
	var (
		detailsCount   = len(cacheDetails)
		totalQueueSize int64
		totalFailures  int64
		cacheStoreType = ""
		header         = ""
	)
	if detailsCount == 0 {
		return ""
	}

	sort.Slice(cacheDetails, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(cacheDetails[p].NodeID)
		nodeID2, _ := strconv.Atoi(cacheDetails[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "QUEUE SIZE", "WRITES", "AVG BATCH", "AVG WRITE", "FAILURES",
		"READS", "AVG READ").WithAlignment(R, R, R, R, R, R, R, R)

	for _, value := range cacheDetails {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		if cacheStoreType == "" {
			cacheStoreType = value.PersistenceType
		}

		totalQueueSize += value.QueueSize
		totalFailures += value.StoreFailures

		table.AddRow(formatSmallInteger(int32(nodeID)),
			formatLargeInteger(value.QueueSize), formatLargeInteger(value.StoreWrites),
			formatLargeInteger(value.StoreAverageBatchSize), formatLargeInteger(value.StoreAverageWriteMillis)+"ms",
			formatLargeInteger(value.StoreFailures),
			formatLargeInteger(value.StoreReads), formatLargeInteger(value.StoreAverageReadMillis)+"ms")
	}

	if includeHeader {
		// create the header
		header =
			fmt.Sprintf("Service/Cache:        %s/%s\n", service, cache) +
				fmt.Sprintf("Cache Store Type:     %s\n", cacheStoreType)
	}
	header += fmt.Sprintf("Total Queue Size:     %s\n", formatLargeInteger(totalQueueSize)) +
		fmt.Sprintf("Total Store Failures: %s\n", formatLargeInteger(totalFailures)) + "\n"

	return header + table.String()
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

	table := newFormattedTable().WithHeader("CONNECTION", "CLUSTER NAME", "HOST", "NS PORT", "URL").
		WithAlignment(L, L, L, R, L)

	for _, value := range clusters {
		if value.SelectedURL != "" {
			table.AddRow(value.ConnectionName, value.ClusterName, value.Host, formatPort(int32(value.NSPort)), value.SelectedURL)
			i++
		}
	}
	return table.String()
}

// FormatProfiles returns the profiles in a column formatted output
func FormatProfiles(profiles []ProfileValue) string {
	var (
		profileCount = len(profiles)
	)
	if profileCount == 0 {
		return ""
	}

	table := newFormattedTable().WithHeader("PROFILE", "VALUE")

	for _, value := range profiles {
		table.AddRow(value.Name, value.Value)
	}

	return table.String()
}

// FormatClusterConnections returns the cluster information in a column formatted output
func FormatClusterConnections(clusters []ClusterConnection) string {
	var (
		clusterCount   = len(clusters)
		currentContext string
		manualCluster  string
		running        string
	)
	if clusterCount == 0 {
		return ""
	}

	sort.Slice(clusters, func(p, q int) bool {
		return strings.Compare(clusters[p].Name, clusters[q].Name) < 0
	})

	table := newFormattedTable().WithHeader("CONNECTION", "TYPE", "URL", "VERSION", "CLUSTER NAME", "TYPE", "CTX", "LOCAL")
	if OutputFormat == constants.WIDE {
		table.AddHeaderColumns("RUNNING")
	}

	for _, value := range clusters {
		currentContext = ""
		if Config.CurrentContext == value.Name {
			currentContext = "*"
		}
		if value.ManuallyCreated {
			manualCluster = stringTrue
		} else {
			manualCluster = stringFalse
		}

		columns := []string{value.Name, value.ConnectionType, value.ConnectionURL,
			value.ClusterVersion, value.ClusterName, value.ClusterType, currentContext, manualCluster}
		if OutputFormat == constants.WIDE {
			running = stringFalse
			if value.ManagementAvailable {
				running = stringTrue
			}
			columns = append(columns, running)
		}

		table.AddRow(columns...)
	}

	return table.String()
}

// FormatTracing returns the member's tracing details in a column formatted output
func FormatTracing(members []config.Member) string {
	var memberCount = len(members)

	if memberCount == 0 {
		return ""
	}

	sort.Slice(members, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(members[p].NodeID)
		nodeID2, _ := strconv.Atoi(members[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, AddressColumn, PortColumn, ProcessColumn, MemberColumn, RoleColumn,
		"TRACING ENABLED", "SAMPLING RATIO").WithAlignment(R, L, R, R, L, L, L, R)

	for _, value := range members {
		var (
			nodeID, _            = strconv.Atoi(value.NodeID)
			tracingEnabled       = stringFalse
			tracingSamplingRatio = na
		)

		if value.TracingSamplingRatio != -1 {
			tracingEnabled = stringTrue
			tracingSamplingRatio = formatPublisherReceiver(value.TracingSamplingRatio)
		}

		table.AddRow(formatSmallInteger(int32(nodeID)), value.UnicastAddress,
			formatPort(value.UnicastPort), value.ProcessName, value.MemberName, value.RoleName, tracingEnabled, tracingSamplingRatio)
	}

	return table.String()
}

// FormatHealthSummary returns member health in a short or summary view
func FormatHealthSummary(health []config.HealthSummaryShort) string {
	if len(health) == 0 {
		return ""
	}

	sort.Slice(health, func(p, q int) bool {
		return strings.Compare(health[p].Name, health[q].Name) < 0
	})

	table := newFormattedTable().WithHeader("NAME", "SUB TYPE", MembersColumn, "STARTED", "LIVE", "READY", "SAFE").
		WithAlignment(L, L, R, R, R, R, R)
	for i := 3; i <= 6; i++ {
		table.AddFormattingFunction(i, healthSummaryFormatter)
	}

	for _, value := range health {
		table.AddRow(value.Name, value.SubType, formatSmallInteger(value.TotalCount),
			getCountString(value.TotalCount, value.StartedCount),
			getCountString(value.TotalCount, value.LiveCount),
			getCountString(value.TotalCount, value.ReadyCount),
			getCountString(value.TotalCount, value.SafeCount))
	}

	return table.String()
}

func getCountString(total, ready int32) string {
	if ready == total {
		return formatSmallInteger(total)
	}
	return formatSmallInteger(ready) + "/" + formatSmallInteger(total)
}

// FormatMemberHealth returns member health in a column formatted output
func FormatMemberHealth(health []config.HealthSummary) string {
	if len(health) == 0 {
		return ""
	}
	var (
		alignmentWide  = []string{R, L, L, L, L, L, L, L, L, L}
		alignment      = []string{R, L, L, L, L, L, L, L, L}
		finalAlignment []string
	)

	if OutputFormat == constants.TABLE {
		finalAlignment = alignment
	} else {
		finalAlignment = alignmentWide
	}

	sort.Slice(health, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(health[p].NodeID)
		nodeID2, _ := strconv.Atoi(health[q].NodeID)

		if nodeID1 == nodeID2 {
			return strings.Compare(health[p].Name, health[q].Name) < 0
		}
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "NAME", "SUB TYPE", "STARTED", "LIVE", "READY", "SAFE",
		"MEMBER HEALTH", "DESCRIPTION").WithAlignment(finalAlignment...)
	for i := 3; i <= 7; i++ {
		table.AddFormattingFunction(i, healthFormatter)
	}

	if OutputFormat == constants.WIDE {
		table.AddHeaderColumns("CLASS")
	}

	for _, value := range health {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		table.AddRow(formatSmallInteger(int32(nodeID)), value.Name, value.SubType,
			formatBool(value.Started), formatBool(value.Live), formatBool(value.Ready), formatBool(value.Safe),
			formatBool(value.MemberHealthCheck), value.Description)

		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(value.ClassName)
		}
	}

	return table.String()
}

// FormatMembers returns the member's information in a column formatted output
func FormatMembers(members []config.Member, verbose bool, storageMap map[int]bool, summary bool) string {
	var (
		memberCount        = len(members)
		alignmentWide      = []string{R, L, L, R, L, L, L, L, L, R, R, L, R, R, R}
		alignment          = []string{R, L, L, R, L, L, L, R, R, R}
		finalAlignment     []string
		formattingFunction = getFormattingFunction()
		roleMap            = make(map[string]int32)
		storageCount       int
	)

	if OutputFormat == constants.TABLE {
		finalAlignment = alignment
	} else {
		finalAlignment = alignmentWide
	}

	sort.Slice(members, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(members[p].NodeID)
		nodeID2, _ := strconv.Atoi(members[q].NodeID)
		return nodeID1 < nodeID2
	})

	var (
		totalMaxMemoryMB          int32
		totalAvailMemoryMB        int32
		totalAvailStorageMemoryMB int32
		totalMaxStorageMemoryMB   int32
		availableStoragePercent   float32
	)

	table := newFormattedTable().WithHeader(NodeIDColumn, AddressColumn, PortColumn, ProcessColumn, MemberColumn, RoleColumn).
		WithAlignment(finalAlignment...)

	if OutputFormat == constants.WIDE {
		table.AddHeaderColumns("MACHINE", "RACK", "SITE", "PUBLISHER", "RECEIVER")
		table.AddFormattingFunction(9, networkStatsFormatter)
		table.AddFormattingFunction(10, networkStatsFormatter)
	}
	table.AddHeaderColumns("STORAGE", MaxHeapColumn, UsedHeapColumn, AvailHeapColumn)

	for _, value := range members {
		var (
			nodeID, _      = strconv.Atoi(value.NodeID)
			storageEnabled = utils.IsStorageEnabled(nodeID, storageMap)
		)
		totalAvailMemoryMB += value.MemoryAvailableMB
		totalMaxMemoryMB += value.MemoryMaxMB

		if storageEnabled {
			totalAvailStorageMemoryMB += value.MemoryAvailableMB
			totalMaxStorageMemoryMB += value.MemoryMaxMB
			storageCount++
		}

		table.AddRow(formatSmallInteger(int32(nodeID)), value.UnicastAddress,
			formatPort(value.UnicastPort), value.ProcessName, value.MemberName, value.RoleName)

		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(value.MachineName, value.RackName, value.SiteName,
				formatPublisherReceiver(value.PublisherSuccessRate), formatPublisherReceiver(value.ReceiverSuccessRate))
		}

		table.AddColumnsToRow(fmt.Sprintf("%v", storageEnabled), formattingFunction(int64(value.MemoryMaxMB)*MB),
			formattingFunction(int64(value.MemoryMaxMB-value.MemoryAvailableMB)*MB),
			formattingFunction(int64(value.MemoryAvailableMB)*MB))

		// summarise the roles
		val, ok := roleMap[value.RoleName]
		if !ok {
			roleMap[value.RoleName] = 1
		} else {
			roleMap[value.RoleName] = val + 1
		}
	}

	totalUsedMB := totalMaxMemoryMB - totalAvailMemoryMB
	availablePercent := float32(totalAvailMemoryMB) / float32(totalMaxMemoryMB) * 100

	totalUsedStorageMB := totalMaxStorageMemoryMB - totalAvailStorageMemoryMB

	if totalAvailStorageMemoryMB > 0 {
		availableStoragePercent = float32(totalAvailStorageMemoryMB) / float32(totalMaxStorageMemoryMB) * 100
	}

	result :=
		fmt.Sprintf("Total cluster members: %d\n", memberCount) +
			fmt.Sprintf("Storage enabled count: %d\n", storageCount) +
			fmt.Sprintf("Cluster Heap - Total: %s Used: %s Available: %s (%4.1f%%)\n",
				strings.TrimSpace(formattingFunction(int64(totalMaxMemoryMB)*MB)),
				strings.TrimSpace(formattingFunction(int64(totalUsedMB)*MB)),
				strings.TrimSpace(formattingFunction(int64(totalAvailMemoryMB)*MB)), availablePercent) +
			fmt.Sprintf("Storage Heap - Total: %s Used: %s Available: %s (%4.1f%%)\n\n",
				strings.TrimSpace(formattingFunction(int64(totalMaxStorageMemoryMB)*MB)),
				strings.TrimSpace(formattingFunction(int64(totalUsedStorageMB)*MB)),
				strings.TrimSpace(formattingFunction(int64(totalAvailStorageMemoryMB)*MB)), availableStoragePercent)

	if summary {
		tableSummary := newFormattedTable().WithHeader(RoleColumn, "COUNT").WithAlignment(L, R)
		for k, v := range roleMap {
			tableSummary.AddRow(k, formatSmallInteger(v))
		}
		return result + tableSummary.String()
	}
	if verbose {
		result += table.String()
	}
	return result
}

// FormatNetworkStatistics returns the member's network statistics in a column formatted output
func FormatNetworkStatistics(members []config.Member) string {
	var (
		alignmentWide      = []string{R, L, L, R, L, L, R, R, R, R, R, R, R, R}
		alignment          = []string{R, L, L, R, L, L, R, R, R, R, R, R, R, R}
		finalAlignment     []string
		formattingFunction = getFormattingFunction()
	)

	if OutputFormat == constants.TABLE {
		finalAlignment = alignment
	} else {
		finalAlignment = alignmentWide
	}

	sort.Slice(members, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(members[p].NodeID)
		nodeID2, _ := strconv.Atoi(members[q].NodeID)
		return nodeID1 < nodeID2
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, AddressColumn, PortColumn, ProcessColumn, MemberColumn, RoleColumn,
		"PKT SENT", "PKT REC", "RESENT", "EFFICIENCY", "SEND Q", dataSent, dataRec, "WEAKEST").
		WithAlignment(finalAlignment...)
	table.AddFormattingFunction(9, networkStatsFormatter)
	table.AddFormattingFunction(10, errorFormatter)

	for _, value := range members {
		var (
			nodeID, _ = strconv.Atoi(value.NodeID)
		)

		table.AddRow(formatSmallInteger(int32(nodeID)), value.UnicastAddress,
			formatPort(value.UnicastPort), value.ProcessName, value.MemberName, value.RoleName)

		table.AddColumnsToRow(formatLargeInteger(value.PacketsSent), formatLargeInteger(value.PacketsReceived),
			formatLargeInteger(value.PacketsResent), formatPercent(value.PacketDeliveryEfficiency),
			formatLargeInteger(value.SendQueueSize), formattingFunction(value.TransportSentBytes),
			formattingFunction(value.TransportReceivedBytes), formatSmallInteger(value.WeakestChannel))
	}

	//table.AddColumnsToRow(fmt.Sprintf("%v", storageEnabled), formattingFunction(int64(value.MemoryMaxMB)*MB),
	//	formattingFunction(int64(value.MemoryMaxMB-value.MemoryAvailableMB)*MB),
	//	formattingFunction(int64(value.MemoryAvailableMB)*MB))

	return table.String()
}

// FormatExecutors returns the executor's information in a column formatted output
func FormatExecutors(executors []config.Executor, summary bool) string {
	var (
		executorCount = len(executors)
		header        = "MEMBER COUNT"
	)
	if executorCount == 0 {
		return ""
	}

	if !summary {
		header = MemberColumn
	}

	sort.Slice(executors, func(p, q int) bool {
		return strings.Compare(executors[p].Name, executors[q].Name) < 0
	})

	table := newFormattedTable().WithHeader(NameColumn, header, "IN PROGRESS", "COMPLETED", "REJECTED", "DESCRIPTION").
		WithAlignment(L, R, R, R, R, L)
	table.AddFormattingFunction(4, errorFormatter)

	var (
		totalRunningTasks   int64
		totalCompletedTasks int64
	)

	for _, value := range executors {
		var columnValue = value.MemberID
		if summary {
			columnValue = fmt.Sprintf("%d", value.MemberCount)
		}
		totalRunningTasks += value.TasksInProgressCount
		totalCompletedTasks += value.TasksCompletedCount
		table.AddRow(value.Name, columnValue,
			formatLargeInteger(value.TasksInProgressCount), formatLargeInteger(value.TasksCompletedCount),
			formatLargeInteger(value.TasksRejectedCount), value.Description)
	}

	return fmt.Sprintf("Total executors: %d\nRunning tasks:   %s\nCompleted tasks: %s\n\n",
		executorCount, formatLargeInteger(totalRunningTasks), formatLargeInteger(totalCompletedTasks)) +
		table.String()
}

// FormatElasticData formats the elastic data summary
func FormatElasticData(edData []config.ElasticData, summary bool) string {
	var (
		edCount            = len(edData)
		formattingFunction = getFormattingFunction()
	)
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

	// if we are not a summary then change column 1
	if !summary {
		column1 = NodeIDColumn
		alignment[0] = R
	}

	table := newFormattedTable().WithHeader(column1, "USED FILES", "TOTAL FILES", "% USED", "MAX FILE SIZE",
		"USED SPACE", "COMMITTED", "HIGHEST LOAD", "COMPACTIONS", "EXHAUSTIVE").WithAlignment(L, R, R, R, R, R, R, R, R, R)

	for _, data := range edData {
		var (
			percentUsed  = float64(data.FileCount) / float64(data.MaxJournalFilesNumber)
			committed    = int64(data.FileCount) * data.MaxFileSize
			column1Value string
		)
		if summary {
			column1Value = data.Name
		} else {
			nodeID, _ := strconv.Atoi(data.NodeID)           //nolint
			column1Value = formatSmallInteger(int32(nodeID)) // #nosec G109
		}

		table.AddRow(column1Value, formatSmallInteger(data.FileCount), formatSmallInteger(data.MaxJournalFilesNumber),
			formatPercent(percentUsed), formattingFunction(data.MaxFileSize),
			formattingFunction(data.TotalDataSize), formattingFunction(committed),
			formatLargeFloat(float64(data.HighestLoadFactor)),
			formatLargeInteger(data.CompactionCount), formatLargeInteger(data.ExhaustiveCompactionCount))
	}

	return table.String()
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

	sort.Slice(reporters, func(p, q int) bool {
		nodeID1, _ := strconv.Atoi(reporters[p].NodeID)
		nodeID2, _ := strconv.Atoi(reporters[q].NodeID)
		return nodeID1 < nodeID2
	})

	if OutputFormat == constants.WIDE {
		maxLength = 0
	}

	table := newFormattedTable().WithHeader(NodeIDColumn, "STATE", "CONFIG FILE", "OUTPUT PATH",
		"BATCH#", "LAST REPORT", "LAST RUN", "AVG RUN", "INTERVAL", "AUTOSTART").
		WithAlignment(R, L, L, L, R, L, R, R, R, L).MaxLength(maxLength)

	for _, value := range reporters {
		var nodeID, _ = strconv.Atoi(value.NodeID)

		table.AddRow(formatSmallInteger(int32(nodeID)), value.State, value.ConfigFile,
			value.OutputPath, formatSmallInteger(value.CurrentBatch), value.LastReport,
			formatSmallInteger(value.LastRunMillis)+"ms", formatLargeFloat(value.RunAverageMillis)+"ms",
			formatSmallInteger(value.IntervalSeconds), fmt.Sprintf("%v", value.AutoStart))
	}

	return table.String()
}

// FormatServices returns the services' information in a column formatted output
func FormatServices(services []config.ServiceSummary) string {
	if len(services) == 0 {
		return ""
	}

	sort.Slice(services, func(p, q int) bool {
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) < 0
	})

	table := newFormattedTable().WithHeader(ServiceNameColumn, "TYPE", MembersColumn, "STATUS HA", "STORAGE", partitions)
	if OutputFormat == constants.WIDE {
		table.WithAlignment(L, L, R, L, R, R, R, R, R, L, L)
		table.AddHeaderColumns(endangered, "VULNERABLE", "UNBALANCED", "STATUS", "SUSPENDED")
		table.AddFormattingFunction(9, statusHAFormatter)
	} else {
		table.WithAlignment(L, L, R, L, R, R)
	}

	table.AddFormattingFunction(3, statusHAFormatter)

	for _, value := range services {
		var (
			status    = "Safe"
			suspended = na
		)
		if value.StorageEnabledCount == -1 || value.StatusHA == na {
			status = na
		} else if value.StatusHA == endangered {
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

		table.AddRow(value.ServiceName, value.ServiceType, formatSmallInteger(value.MemberCount),
			value.StatusHA, formatSmallInteger(value.StorageEnabledCount), formatSmallInteger(value.PartitionsAll))

		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(formatSmallInteger(value.PartitionsEndangered),
				formatSmallInteger(value.PartitionsVulnerable),
				formatSmallInteger(value.PartitionsUnbalanced), status, suspended)
		}
	}

	return table.String()
}

// FormatServicesStorage returns the services' storage information in a column formatted output
func FormatServicesStorage(services []config.ServiceStorageSummary) string {
	if len(services) == 0 {
		return ""
	}
	var formattingFunction = getFormattingFunction()

	sort.Slice(services, func(p, q int) bool {
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) < 0
	})

	table := newFormattedTable().WithHeader(ServiceNameColumn, partitions, "NODES", "AVG PARTITION", "MAX PARTITION",
		"AVG STORAGE", "MAX STORAGE NODE", "MAX NODE").WithAlignment(L, R, R, R, R, R, R, R)

	for _, value := range services {
		var maxNode = "-"
		if value.MaxLoadNodeID > 0 {
			maxNode = fmt.Sprintf("%v", value.MaxLoadNodeID)
		}
		table.AddRow(value.ServiceName, formatSmallInteger(value.PartitionCount),
			formatSmallInteger(value.ServiceNodeCount), formattingFunction(value.AveragePartitionSizeKB*KB),
			formattingFunction(value.MaxPartitionSizeKB*KB), formattingFunction(value.AverageStorageSizeKB*KB),
			formattingFunction(value.MaxStorageSizeKB*KB), maxNode)
	}

	return table.String()
}

// FormatMachines returns the machine's information in a column formatted output
func FormatMachines(machines []config.Machine) string {
	if len(machines) == 0 {
		return ""
	}
	var (
		formattingFunction = getFormattingFunction()
		load               string
		percentFree        float64
	)

	sort.Slice(machines, func(p, q int) bool {
		return strings.Compare(machines[p].MachineName, machines[q].MachineName) < 0
	})

	table := newFormattedTable().WithHeader("MACHINE", "PROCESSORS", "LOAD", "TOTAL MEMORY", "FREE MEMORY",
		"% FREE", "OS", "ARCH", "VERSION").WithAlignment(L, R, R, R, R, R, L, L, L)
	table.AddFormattingFunction(5, machineMemoryFormatting)

	for _, value := range machines {
		if value.SystemLoadAverage >= 0 {
			load = fmt.Sprintf("%v", value.SystemLoadAverage)
		} else {
			load = fmt.Sprintf("%v", value.SystemCPULoad)
		}

		percentFree = float64(value.FreePhysicalMemorySize) / float64(value.TotalPhysicalMemorySize)

		table.AddRow(value.MachineName, formatSmallInteger(value.AvailableProcessors),
			load, formattingFunction(value.TotalPhysicalMemorySize),
			formattingFunction(value.FreePhysicalMemorySize),
			formatPercent(percentFree), value.Name, value.Arch, value.Version)
	}

	return table.String()
}

// FormatHTTPSessions returns the Coherence*Web information in a column formatted output
func FormatHTTPSessions(sessions []config.HTTPSessionSummary, isSummary bool) string {
	if len(sessions) == 0 {
		return ""
	}
	var (
		header = []string{"TYPE", "SESSION TIMEOUT", CacheColumn, "OVERFLOW",
			avgSize, "TOTAL REAPED", "AVG DURATION", "LAST REAP", "UPDATES"}
	)
	sort.Slice(sessions, func(p, q int) bool {
		if sessions[p].AppID == sessions[q].AppID {
			nodeID1, _ := strconv.Atoi(sessions[p].NodeID)
			nodeID2, _ := strconv.Atoi(sessions[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(sessions[p].AppID, sessions[q].AppID) > 0
	})

	table := newFormattedTable()

	if !isSummary {
		table.WithHeader(NodeIDColumn)
		table.WithAlignment(R, L, R, L, L, R, R, R, R, R)
	} else {
		table.WithHeader("APPLICATION")
		table.WithAlignment(L, L, R, L, L, R, R, R, R, R)
	}

	table.AddHeaderColumns(header...)

	for _, value := range sessions {
		var column1 string
		if isSummary {
			column1 = value.AppID
		} else {
			column1 = value.NodeID
		}

		table.AddRow(column1, value.Type, formatSmallInteger(value.SessionTimeout), value.SessionCacheName,
			value.OverflowCacheName, formatSmallInteger(value.SessionAverageSize),
			formatLargeInteger(value.ReapedSessionsTotal), formatLargeInteger(value.AverageReapDuration),
			formatLargeInteger(value.LastReapDuration), formatLargeInteger(value.SessionUpdates))
	}

	return table.String()
}

// FormatPersistenceServices returns the services' persistence information in a column formatted output
// if isSummary then leave out storage count
func FormatPersistenceServices(services []config.ServiceSummary, isSummary bool) string {
	if len(services) == 0 {
		return ""
	}
	var (
		formattingFunction    = getFormattingFunction()
		averageAverageLatency float64
		totalActiveSpaceUsed  int64
		totalBackupSpaceUsed  int64
	)

	sort.Slice(services, func(p, q int) bool {
		if services[p].ServiceName == services[q].ServiceName {
			nodeID1, _ := strconv.Atoi(services[p].NodeID)
			nodeID2, _ := strconv.Atoi(services[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) < 0
	})

	table := newFormattedTable()

	if isSummary {
		table.WithAlignment(L, R, L, R, R, R, R, R, L)
		table.WithHeader(ServiceNameColumn, "STORAGE COUNT", "PERSISTENCE MODE",
			"ACTIVE SPACE", "BACKUP SPACE", "AVG LATENCY", "MAX LATENCY", "SNAPSHOTS", "STATUS")
	} else {
		table.WithAlignment(R, L, R, R, R, R)
		table.WithHeader(NodeIDColumn, "PERSISTENCE MODE", "ACTIVE SPACE", "BACKUP SPACE", "AVG LATENCY", "MAX LATENCY")
	}

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
		totalBackupSpaceUsed += value.PersistenceBackupSpaceUsed

		if isSummary {
			table.AddRow(value.ServiceName, formatSmallInteger(value.StorageEnabledCount))
		} else {
			table.AddRow(value.NodeID)
		}

		table.AddColumnsToRow(value.PersistenceMode,
			formattingFunction(max(0, value.PersistenceActiveSpaceUsed)),
			formattingFunction(max(0, value.PersistenceBackupSpaceUsed)),
			formatLatency(float32(averageAverageLatency)),
			formatLargeInteger(max(value.PersistenceLatencyMax, 0))+"ms")

		if isSummary {
			table.AddColumnsToRow(formatSmallInteger(int32(len(value.Snapshots))), value.OperationStatus)
		}
	}

	return fmt.Sprintf("Total Active Space Used: %s\n", formattingFunction(max(totalActiveSpaceUsed, 0))) +
		fmt.Sprintf("Total Backup Space Used: %s\n\n", formattingFunction(max(totalBackupSpaceUsed, 0))) +
		table.String()
}

// FormatSnapshots returns the snapshots in a formatted output
func FormatSnapshots(serviceSnapshots []config.Snapshots, archived bool) string {
	if len(serviceSnapshots) == 0 {
		return ""
	}
	var snapshotHeader = "SNAPSHOT NAME"

	sort.Slice(serviceSnapshots, func(p, q int) bool {
		return strings.Compare(serviceSnapshots[p].ServiceName, serviceSnapshots[q].ServiceName) < 0
	})

	if archived {
		snapshotHeader = "ARCHIVED " + snapshotHeader
	}

	table := newFormattedTable().WithHeader(ServiceColumn, snapshotHeader)

	for _, service := range serviceSnapshots {
		snapshots := service.Snapshots
		sort.Slice(snapshots, func(p, q int) bool {
			return strings.Compare(snapshots[p], snapshots[q]) < 0
		})
		for _, value := range snapshots {
			table.AddRow(service.ServiceName, value)
		}

	}
	return table.String()
}

// FormatProxyConnections returns the proxy connections in a column formatted output
func FormatProxyConnections(connections []config.ProxyConnection) string {

	if len(connections) == 0 {
		return ""
	}
	var (
		formattingFunction = getFormattingFunction()
	)

	sort.Slice(connections, func(p, q int) bool {
		return connections[p].ConnectionTimeMillis < connections[q].ConnectionTimeMillis
	})

	table := newFormattedTable().WithHeader(NodeIDColumn, "CONN MS", "CONN TIME", "REMOTE ADDR/PORT",
		dataSent, dataRec, "BACKLOG", "CLIENT PROCESS", "CLIENT ROLE")
	table.AddFormattingFunction(6, errorFormatter)

	if OutputFormat == constants.WIDE {
		table.WithAlignment(R, R, R, L, R, R, R, R, L, L)
		table.AddHeaderColumns("REMOTE MEMBER")
	} else {
		table.WithAlignment(R, R, R, L, R, R, R, R, L)
	}

	for _, value := range connections {
		table.AddRow(value.NodeID, formatLargeInteger(value.ConnectionTimeMillis),
			formatConnectionMillis(value.ConnectionTimeMillis),
			value.RemoteAddress+":"+formatPort(value.RemotePort),
			formattingFunction(value.TotalBytesSent), formattingFunction(value.TotalBytesReceived),
			formatLargeInteger(value.OutgoingByteBacklog), value.ClientProcessName, value.ClientRole)
		if OutputFormat == constants.WIDE {
			table.AddColumnsToRow(value.Member)
		}
	}

	return table.String()
}

// FormatProxyServers returns the proxy servers' information in a column formatted output
// protocol is either tcp or http and will display a different format based upon this
func FormatProxyServers(services []config.ProxySummary, protocol string) string {
	// get the number of proxies matching the protocol
	var (
		serviceCount       = 0
		formattingFunction = getFormattingFunction()
	)

	for _, value := range services {
		if protocol == value.Protocol {
			serviceCount++
		}
	}

	if serviceCount == 0 {
		return ""
	}

	sort.Slice(services, func(p, q int) bool {
		if services[p].ServiceName == services[q].ServiceName {
			nodeID1, _ := strconv.Atoi(services[p].NodeID)
			nodeID2, _ := strconv.Atoi(services[q].NodeID)
			return nodeID1 < nodeID2
		}
		return strings.Compare(services[p].ServiceName, services[q].ServiceName) < 0
	})

	// common header
	table := newFormattedTable().WithHeader(NodeIDColumn, "HOST IP", ServiceNameColumn)

	if protocol == tcp {
		table.AddHeaderColumns("CONNECTIONS", dataSent, dataRec)
		if OutputFormat == constants.WIDE {
			table.AddHeaderColumns("MSG SENT", "MSG RCV", "BYTES BACKLOG", "MSG BACKLOG", "UNAUTH")
			table.WithAlignment(L, L, L, R, R, R, R, R, R, R, R)
			table.AddFormattingFunction(9, errorFormatter)
			table.AddFormattingFunction(10, errorFormatter)
		} else {
			table.WithAlignment(L, L, L, R, R, R)
		}
	} else {
		table.AddHeaderColumns("SERVER TYPE", "REQUESTS", "ERRORS")
		table.AddFormattingFunction(5, errorFormatter)
		if OutputFormat == constants.WIDE {
			table.AddHeaderColumns("1xx", "2xx", "3xx", "4xx", "5xx")
			table.WithAlignment(L, L, L, L, R, R, R, R, R, R, R)
			table.AddFormattingFunction(9, errorFormatter)
			table.AddFormattingFunction(10, errorFormatter)
		} else {
			table.WithAlignment(L, L, L, L, R, R)
		}
	}

	for _, value := range services {
		if protocol != value.Protocol {
			continue
		}
		// common values
		table.AddRow(value.NodeID, value.HostIP, value.ServiceName)

		if protocol == tcp {
			table.AddColumnsToRow(formatLargeInteger(value.ConnectionCount),
				formattingFunction(value.TotalBytesSent), formattingFunction(value.TotalBytesReceived))
			if OutputFormat == constants.WIDE {
				table.AddColumnsToRow(formatLargeInteger(value.TotalMessagesSent),
					formatLargeInteger(value.TotalMessagesReceived), formatLargeInteger(value.OutgoingByteBacklog),
					formatLargeInteger(value.OutgoingMessageBacklog), formatLargeInteger(value.UnAuthConnectionAttempts))
			}
		} else {
			table.AddColumnsToRow(value.HTTPServerType,
				formatLargeInteger(value.TotalRequestCount), formatLargeInteger(value.TotalErrorCount))
			if OutputFormat == constants.WIDE {
				table.AddColumnsToRow(formatLargeInteger(value.ResponseCount1xx),
					formatLargeInteger(value.ResponseCount2xx), formatLargeInteger(value.ResponseCount3xx),
					formatLargeInteger(value.ResponseCount4xx), formatLargeInteger(value.ResponseCount5xx))
			}
		}
	}

	return table.String()
}

func getFormattingFunction() func(bytesValue int64) string {
	// first check for a specific override of the format
	if kbFormat {
		return formatKBOnly
	}
	if mbFormat {
		return formatMBOnly
	}
	if gbFormat {
		return formatGBOnly
	}
	if bFormat {
		return formatBytesOnly
	}

	// then, check for default bytes format in config if none was set
	if Config.DefaultBytesFormat == bytesFormatK {
		return formatKBOnly
	}
	if Config.DefaultBytesFormat == bytesFormatM {
		return formatMBOnly
	}
	if Config.DefaultBytesFormat == bytesFormatG {
		return formatGBOnly
	}

	return formatBytesOnly
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
		return na
	}
	return strings.TrimSpace(printer.Sprintf("%6.2f%%", value*100))
}

func formatBytesOnly(bytesValue int64) string {
	return printer.Sprintf("%-0d", bytesValue)
}

func formatKBOnly(bytesValue int64) string {
	return printer.Sprintf("%-0d KB", bytesValue/1024)
}

func formatMBOnly(bytesValue int64) string {
	return printer.Sprintf("%-0d MB", bytesValue/1024/1024)
}

func formatGBOnly(bytesValue int64) string {
	return printer.Sprintf("%-.1f GB", float64(bytesValue)/1024/1024/1024)
}

func formatBool(boolValue bool) string {
	return printer.Sprintf("%v", boolValue)
}

func formatConnectionMillis(millis int64) string {
	ms := millis % 1000
	seconds := millis / 1000
	mins := seconds / 60
	hours := mins / 60
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm %02ds", days, hours%24, mins%60, seconds%60)
	}
	if hours > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", hours, mins%60, seconds%60)
	}
	if mins > 0 {
		return fmt.Sprintf("%02dm %02ds", mins, seconds%60)
	}

	return fmt.Sprintf("%d.%.1ds", seconds, ms/100)
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

var _ FormattedTable = &formattedTable{}

type formatter func(string) string

// FormattedTable defines a formatted table of information.
type FormattedTable interface {
	WithAlignment(...string) FormattedTable
	WithHeader(...string) FormattedTable
	MaxLength(int) FormattedTable
	AddColumnsToRow(...string)
	AddHeaderColumns(...string)
	AddRow(...string)
	AddFormattingFunction(int, formatter)
	String() string
}

// formattedTable is an implementation of a FormattedTable.
type formattedTable struct {
	header           []string
	rows             [][]string
	alignment        []string
	maxLen           int
	columnFormatters map[int]formatter
}

// newFormattedTable returns a new formatted table.
func newFormattedTable() FormattedTable {
	table := &formattedTable{}
	table.rows = [][]string{}
	table.columnFormatters = make(map[int]formatter, 0)
	return table
}

// WithAlignment sets the alignment for the table.
func (t *formattedTable) WithAlignment(alignment ...string) FormattedTable {
	t.alignment = alignment
	return t
}

// WithHeader sets the header to the used by the table.
func (t *formattedTable) WithHeader(header ...string) FormattedTable {
	t.header = header
	return t
}

// MaxLength sets the maximum length of values in the table unless -o wide is used.
func (t *formattedTable) MaxLength(maxLen int) FormattedTable {
	t.maxLen = maxLen
	return t
}

// AddRow adds a row to the table.
func (t *formattedTable) AddRow(newRow ...string) {
	t.rows = append(t.rows, newRow)
}

// AddFormattingFunction adds a formatting function to a column.
func (t *formattedTable) AddFormattingFunction(col int, f formatter) {
	t.columnFormatters[col] = f
}

// AddColumnsToRow adds columns to the last row. Typically used for -o wide.
func (t *formattedTable) AddColumnsToRow(newColumns ...string) {
	lastRowNum := len(t.rows)
	if lastRowNum == 0 {
		return
	}
	t.rows[lastRowNum-1] = append(t.rows[lastRowNum-1], newColumns...)
}

// AddHeaderColumns adds columns to the header row (0). Typically used for -o wide.
func (t *formattedTable) AddHeaderColumns(newColumns ...string) {
	if (len(t.header)) == 0 {
		return
	}
	t.header = append(t.header, newColumns...)
}

// String returns a string representation of the table.
func (t *formattedTable) String() string {
	var (
		columnLengths = t.getMaxColumnLen()
		sb            strings.Builder
		numberColumns = len(columnLengths)
		alignmentLen  = len(t.alignment)
		hasAlignments = alignmentLen > 0
		align         string
		truncate      = make([]bool, numberColumns)
	)

	// check if any columns > max len and >= 10 as why bother...
	if t.maxLen > 0 {
		for i, value := range columnLengths {
			if value >= 10 && value > t.maxLen {
				truncate[i] = true
				columnLengths[i] = t.maxLen
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
		if hasAlignments && t.alignment[i] == R {
			align = fmt.Sprintf("%%%ds", columnLengths[i]) // align right
		}
		stringFormats[i] = align
	}

	for r, row := range t.getCombined() {
		// format each individual column entry
		for i, e := range row {
			actualValue := fmt.Sprintf(stringFormats[i], e)

			if r > 0 && Config.Color == on {
				// check for formatting function after first row
				if f, ok := t.columnFormatters[i]; ok {
					actualValue = f(actualValue)
				}
			}

			if truncate[i] && len(actualValue) > columnLengths[i] {
				// truncate the value to max len -3 and append three ...
				actualValue = actualValue[:t.maxLen-3] + "..."
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

// getMaxColumnLengthsNew returns an array representing the max lengths of columns
// delimited with the sep
func (t *formattedTable) getMaxColumnLen() []int {
	columns := t.getCombined()

	// find the number of values from the first entry
	var (
		numValues = len(t.header)
		lengths   = make([]int, numValues)
	)

	for _, value := range columns {
		for j, entry := range value {
			if len(entry) > lengths[j] {
				lengths[j] = len(entry)
			}
		}
	}
	return lengths
}

// getCombined returns the combined header and rows.
func (t *formattedTable) getCombined() [][]string {
	columns := make([][]string, 0)
	columns = append(columns, t.header)
	columns = append(columns, t.rows...)

	return columns
}
