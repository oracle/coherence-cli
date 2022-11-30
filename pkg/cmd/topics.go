/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SupplyTopicMessage = "you must provide a topic"
const NoTopicForService = "there are no topics for service %s"
const TopicDoesNotExist = "a topic named %s does not exist for service %s"
const nodeIDMessage = "node id to show channels for"

var (
	topicsNodeID    int32
	subscriber      int64
	subscriberGroup string
)

// getTopicsCmd represents the get topics command
var getTopicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "display topics for a cluster",
	Long:  `The 'get topics' command displays topics for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			connection  string
			dataFetcher fetcher.Fetcher
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var (
				topicsDetails           config.TopicDetails
				topicsMemberDetails     []config.TopicsMemberDetail
				topicsSubscriberDetails []config.TopicsSubscriberDetail
			)

			// get the topics and services
			topicsDetails, err = getTopics(dataFetcher, serviceName)
			if err != nil {
				return err
			}

			if len(topicsDetails.Details) == 0 {
				return fmt.Errorf(NoTopicForService, serviceName)
			}

			topicsMemberDetails, err = getTopicsMembers(dataFetcher, topicsDetails)
			if err != nil {
				return err
			}

			topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, topicsDetails)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSON) {
				json, err := json.Marshal(topicsDetails)
				if err != nil {
					return err
				}
				if OutputFormat == constants.JSONPATH {
					result, err := utils.GetJSONPathResults(json, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					if err != nil {
						return err
					}
					cmd.Println(string(json))
				}
			} else {
				printWatchHeader(cmd)
				var sb strings.Builder

				cmd.Println(FormatCurrentCluster(connection))

				enrichTopicsSummary(&topicsDetails, topicsMemberDetails, topicsSubscriberDetails)

				sb.WriteString(FormatTopicsSummary(topicsDetails.Details))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getSubscribersCmd represents the get subscribers command
var getSubscribersCmd = &cobra.Command{
	Use:   "subscribers topic-name",
	Short: "display subscribers for a topic and service",
	Long:  `The 'get subscribers' command displays subscribers for a topic and service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                 error
			connection          string
			dataFetcher         fetcher.Fetcher
			subscriberTopicName = args[0]
			selectedDetails     config.TopicDetails
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, subscriberTopicName)
		if err != nil {
			return err
		}

		for {
			var topicsSubscriberDetails []config.TopicsSubscriberDetail

			topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSON) {
				topicsResult, err := dataFetcher.GetTopicsSubscribersJSON(serviceName, subscriberTopicName)
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, topicsResult); err != nil {
					return err
				}
			} else {
				printWatchHeader(cmd)
				var sb strings.Builder

				cmd.Println(FormatCurrentCluster(connection))

				sb.WriteString(getTopicsHeader(serviceName, subscriberTopicName) + "\n")
				sb.WriteString(FormatTopicsSubscribers(topicsSubscriberDetails))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getSubscriberGroupsCmd represents the get subscriber-groups command
var getSubscriberGroupsCmd = &cobra.Command{
	Use:   "subscriber-groups topic-name",
	Short: "display subscriber-groups for a topic and service",
	Long:  `The 'get subscribers' command displays subscriber-groups for a topic and service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err             error
			connection      string
			dataFetcher     fetcher.Fetcher
			topicName       = args[0]
			selectedDetails config.TopicDetails
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, topicName)
		if err != nil {
			return err
		}

		for {
			var subscriberGroupDetails []config.TopicsSubscriberGroupDetail

			subscriberGroupDetails, err = getTopicsSubscriberGroups(dataFetcher, selectedDetails)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSON) {
				subscriberGroupResult, err := dataFetcher.GetTopicsSubscriberGroupsJSON(serviceName, topicName)
				if err != nil {
					return err
				}
				if OutputFormat == constants.JSONPATH {
					result, err := utils.GetJSONPathResults(subscriberGroupResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					if err != nil {
						return err
					}
					cmd.Println(string(subscriberGroupResult))
				}
			} else {
				printWatchHeader(cmd)
				var sb strings.Builder

				cmd.Println(FormatCurrentCluster(connection))

				sb.WriteString(getTopicsHeader(serviceName, topicName) + "\n")
				sb.WriteString(FormatTopicsSubscriberGroups(subscriberGroupDetails))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getTopicsDetails returns selected topic details for a given service and topic.
// typically used when a topic and service is selected
func getTopicsDetails(dataFetcher fetcher.Fetcher, topicServiceName, topicName string) (config.TopicDetails, error) {
	// get the topics and services
	topicsDetails, err := getTopics(dataFetcher, topicServiceName)
	if err != nil {
		return topicsDetails, err
	}

	if len(topicsDetails.Details) == 0 {
		return topicsDetails, fmt.Errorf(NoTopicForService, topicServiceName)
	}

	index := getTopicDetailIndex(topicsDetails, topicServiceName, topicName)
	if index == -1 {
		return topicsDetails, fmt.Errorf(TopicDoesNotExist, topicName, topicServiceName)
	}
	// ensure the chosen topic/ service is selected
	selectedDetails := config.TopicDetails{Details: make([]config.TopicDetail, 1)}
	selectedDetails.Details[0] = topicsDetails.Details[index]

	return selectedDetails, nil
}

// describeTopicCmd represents the describe topic command
var describeTopicCmd = &cobra.Command{
	Use:   "topic topic-name",
	Short: "describe a topic",
	Long:  `The 'describe topic' command shows information related to a topic and service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                     error
			connection              string
			dataFetcher             fetcher.Fetcher
			topicName               = args[0]
			topicsDetails           config.TopicDetails
			topicsDetail            config.TopicDetail
			topicsSubscriberDetails []config.TopicsSubscriberDetail
			topicsMemberDetails     []config.TopicsMemberDetail
			subscriberGroupDetails  []config.TopicsSubscriberGroupDetail
			jsonResult              []byte
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// get the topics and services
		topicsDetails, err = getTopics(dataFetcher, serviceName)
		if err != nil {
			return err
		}

		if len(topicsDetails.Details) == 0 {
			return fmt.Errorf(NoTopicForService, serviceName)
		}

		index := getTopicDetailIndex(topicsDetails, serviceName, topicName)
		if index == -1 {
			return fmt.Errorf(TopicDoesNotExist, topicName, serviceName)
		}

		// ensure the chosen topic/ service is selected
		selectedDetails := config.TopicDetails{Details: make([]config.TopicDetail, 1)}
		selectedDetails.Details[0] = topicsDetails.Details[index]

		// retrieve the topics member information for summarising
		topicsMemberDetails, err = getTopicsMembers(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		subscriberGroupDetails, err = getTopicsSubscriberGroups(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		enrichTopicsSummary(&topicsDetails, topicsMemberDetails, topicsSubscriberDetails)

		topicsDetail = topicsDetails.Details[index]
		jsonResult, err = json.Marshal(topicsDetail)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSON) {
			topicsSubscribers, err := dataFetcher.GetTopicsSubscribersJSON(serviceName, topicName)
			if err != nil {
				return err
			}
			topicsMembers, err := dataFetcher.GetTopicsMembersJSON(serviceName, topicName)
			if err != nil {
				return err
			}
			finalResult, err := utils.CombineByteArraysForJSON([][]byte{jsonResult, topicsSubscribers,
				topicsMembers},
				[]string{"topics", "subscribers", "members"})

			if OutputFormat == constants.JSONPATH {
				result, err := utils.GetJSONPathResults(finalResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else {
				if err != nil {
					return err
				}
				cmd.Println(string(finalResult))
			}
		} else {
			var sb strings.Builder
			sb.WriteString(FormatCurrentCluster(connection))

			sb.WriteString("\nTOPIC DETAILS\n")
			sb.WriteString("-------------\n")
			value, err := FormatJSONForDescribe(jsonResult, true, "Name", "Service")
			if err != nil {
				return err
			}

			sb.WriteString(value)

			sb.WriteString("\nMEMBERS\n")
			sb.WriteString("-------\n")

			sb.WriteString(FormatTopicsMembers(topicsMemberDetails))

			sb.WriteString("\nSUBSCRIBERS\n")
			sb.WriteString("-----------\n")

			sb.WriteString(FormatTopicsSubscribers(topicsSubscriberDetails))

			sb.WriteString("\nSUBSCRIBER GROUPS\n")
			sb.WriteString("-----------------\n")
			sb.WriteString(FormatTopicsSubscriberGroups(subscriberGroupDetails))

			cmd.Println(sb.String())
		}

		return nil
	},
}

// getTopicMembersCmd represents the get topic-members command
var getTopicMembersCmd = &cobra.Command{
	Use:   "topic-members topic-name",
	Short: "display members for a topic",
	Long:  `The 'get topic-members' command displays members for topic and service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                 error
			connection          string
			dataFetcher         fetcher.Fetcher
			topicName           = args[0]
			selectedDetails     config.TopicDetails
			topicsMemberDetails []config.TopicsMemberDetail
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, topicName)
		if err != nil {
			return err
		}

		for {
			// retrieve the topics member information for summarising
			topicsMemberDetails, err = getTopicsMembers(dataFetcher, selectedDetails)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSON) {
				topicsResult, err := dataFetcher.GetTopicsSubscribersJSON(serviceName, topicName)
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, topicsResult); err != nil {
					return err
				}
			} else {
				printWatchHeader(cmd)
				var sb strings.Builder
				cmd.Println(FormatCurrentCluster(connection))

				sb.WriteString(getTopicsHeader(serviceName, topicName) + "\n")

				sb.WriteString(FormatTopicsMembers(topicsMemberDetails))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getMemberChannelsCmd represents the get member-channels command
var getMemberChannelsCmd = &cobra.Command{
	Use:   "member-channels topic-name",
	Short: "display channel details for a topic, service and node",
	Long:  `The 'get member-channels' command displays channel details for a topic, service and node.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                 error
			connection          string
			dataFetcher         fetcher.Fetcher
			topicName           = args[0]
			selectedDetails     config.TopicDetails
			topicsMemberDetails []config.TopicsMemberDetail
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, topicName)
		if err != nil {
			return err
		}

		// retrieve the topics member information for summarising
		topicsMemberDetails, err = getTopicsMembers(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		// validate the node
		nodeIndex := -1
		for i, v := range topicsMemberDetails {
			nodeID, _ := strconv.ParseInt(v.NodeID, 10, 64)
			if int32(nodeID) == topicsNodeID {
				nodeIndex = i
				break
			}
		}

		if nodeIndex == -1 {
			return fmt.Errorf("unable to find node %d for topic %s and service %s", topicsNodeID, topicName, serviceName)
		}

		for {
			if strings.Contains(OutputFormat, constants.JSON) {
				topicsResult, err := dataFetcher.GetTopicsSubscribersJSON(serviceName, topicName)
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, topicsResult); err != nil {
					return err
				}
			} else {
				// retrieve the topics member information for summarising
				topicsMemberDetails, err = getTopicsMembers(dataFetcher, selectedDetails)
				if err != nil {
					return err
				}

				printWatchHeader(cmd)
				var sb strings.Builder
				cmd.Println(FormatCurrentCluster(connection))

				numChannels := len(topicsMemberDetails[nodeIndex].Channels)

				sb.WriteString(fmt.Sprintf("Service:      %s\n", serviceName))
				sb.WriteString(fmt.Sprintf("Topic:        %s\n", topicName))
				sb.WriteString(fmt.Sprintf("Node ID:      %d\n", topicsNodeID))
				sb.WriteString(fmt.Sprintf("ChannelCount: %d\n\n", numChannels))

				sb.WriteString(FormatChannelStats(generateChannelStats(topicsMemberDetails[nodeIndex].Channels)))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getSubscriberChannelsCmd represents the get subscriber-channels command
var getSubscriberChannelsCmd = &cobra.Command{
	Use:   "subscriber-channels topic-name",
	Short: "display channel details for a topic, service, node and subscriber id",
	Long:  `The 'get subscriber-channels' command displays channel details for a topic, service, node and subscriber id.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                     error
			connection              string
			dataFetcher             fetcher.Fetcher
			topicName               = args[0]
			selectedDetails         config.TopicDetails
			topicsSubscriberDetails []config.TopicsSubscriberDetail
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, topicName)
		if err != nil {
			return err
		}

		topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		// validate the node
		nodeIndex := -1
		for i, v := range topicsSubscriberDetails {
			nodeID, _ := strconv.ParseInt(v.NodeID, 10, 64)
			if int32(nodeID) == topicsNodeID && v.ID == subscriber {
				nodeIndex = i
				break
			}
		}

		if nodeIndex == -1 {
			return fmt.Errorf("unable to find node %d for topic %s and service %s and subscriber id %d",
				topicsNodeID, topicName, serviceName, subscriber)
		}

		for {
			if strings.Contains(OutputFormat, constants.JSON) {
				topicsResult, err := dataFetcher.GetTopicsSubscribersJSON(serviceName, topicName)
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, topicsResult); err != nil {
					return err
				}
			} else {
				topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
				if err != nil {
					return err
				}

				printWatchHeader(cmd)
				var sb strings.Builder
				cmd.Println(FormatCurrentCluster(connection))

				numChannels := len(topicsSubscriberDetails[nodeIndex].Channels)

				sb.WriteString(fmt.Sprintf("Service:          %s\n", serviceName))
				sb.WriteString(fmt.Sprintf("Topic:            %s\n", topicName))
				sb.WriteString(fmt.Sprintf("Node ID:          %d\n", topicsNodeID))
				sb.WriteString(fmt.Sprintf("ChannelCount:     %d\n", numChannels))
				sb.WriteString(fmt.Sprintf("Subscriber Group: %d\n\n", subscriber))

				sb.WriteString(FormatSubscriberChannelStats(generateSubscriberChannelStats(topicsSubscriberDetails[nodeIndex].Channels)))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getSubscriberGroupChannelsCmd represents the get subscriber-channels command
var getSubscriberGroupChannelsCmd = &cobra.Command{
	Use:   "sub-grp-channels topic-name",
	Short: "display channel details for a topic, service, node and subscriber group",
	Long:  `The 'get sub-grp-channels' command displays channel details for a topic, service, node and subscriber group.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, SupplyTopicMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err                          error
			connection                   string
			dataFetcher                  fetcher.Fetcher
			topicName                    = args[0]
			selectedDetails              config.TopicDetails
			topicsSubscriberGroupDetails []config.TopicsSubscriberGroupDetail
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		selectedDetails, err = getTopicsDetails(dataFetcher, serviceName, topicName)
		if err != nil {
			return err
		}

		topicsSubscriberGroupDetails, err = getTopicsSubscriberGroups(dataFetcher, selectedDetails)
		if err != nil {
			return err
		}

		// validate the node
		nodeIndex := -1
		for i, v := range topicsSubscriberGroupDetails {
			nodeID, _ := strconv.ParseInt(v.NodeID, 10, 64)
			if int32(nodeID) == topicsNodeID && v.SubscriberGroup == subscriberGroup {
				nodeIndex = i
				break
			}
		}

		if nodeIndex == -1 {
			return fmt.Errorf("unable to find node %d for topic %s and service %s and subscriber group %s",
				topicsNodeID, topicName, serviceName, subscriberGroup)
		}

		for {
			if strings.Contains(OutputFormat, constants.JSON) {
				topicsResult, err := dataFetcher.GetTopicsSubscriberGroupsJSON(serviceName, topicName)
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, topicsResult); err != nil {
					return err
				}
			} else {
				topicsSubscriberGroupDetails, err = getTopicsSubscriberGroups(dataFetcher, selectedDetails)
				if err != nil {
					return err
				}

				printWatchHeader(cmd)
				var sb strings.Builder
				cmd.Println(FormatCurrentCluster(connection))

				numChannels := topicsSubscriberGroupDetails[nodeIndex].ChannelCount

				sb.WriteString(fmt.Sprintf("Service:          %s\n", serviceName))
				sb.WriteString(fmt.Sprintf("Topic:            %s\n", topicName))
				sb.WriteString(fmt.Sprintf("Node ID:          %d\n", topicsNodeID))
				sb.WriteString(fmt.Sprintf("ChannelCount:     %d\n", numChannels))
				sb.WriteString(fmt.Sprintf("Subscriber Group: %s\n\n", subscriberGroup))

				sb.WriteString(FormatSubscriberGroupChannelStats(generateSubscriberGroupChannelStats(topicsSubscriberGroupDetails[nodeIndex].Channels)))

				cmd.Println(sb.String())
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

func generateChannelStats(stats map[string]interface{}) []config.ChannelStats {
	result := make([]config.ChannelStats, 0)

	for _, value := range stats {
		myMap := value.(map[string]interface{})
		tail := myMap["Tail"].(string)

		stat := config.ChannelStats{
			Tail:                       tail,
			Channel:                    int64(myMap["Channel"].(float64)),
			PublishedCount:             int64(myMap["PublishedCount"].(float64)),
			PublishedMeanRate:          myMap["PublishedMeanRate"].(float64),
			PublishedOneMinuteRate:     myMap["PublishedOneMinuteRate"].(float64),
			PublishedFiveMinuteRate:    myMap["PublishedFiveMinuteRate"].(float64),
			PublishedFifteenMinuteRate: myMap["PublishedFifteenMinuteRate"].(float64),
		}
		result = append(result, stat)
	}

	return result
}

func generateSubscriberChannelStats(stats map[string]interface{}) []config.SubscriberChannelStats {
	result := make([]config.SubscriberChannelStats, 0)

	for _, value := range stats {
		myMap := value.(map[string]interface{})
		head := myMap["Head"].(string)
		stat := config.SubscriberChannelStats{
			Head:         head,
			Channel:      int64(myMap["Channel"].(float64)),
			Empty:        myMap["Empty"].(bool),
			Owned:        myMap["Owned"].(bool),
			LastCommit:   myMap["LastCommit"].(string),
			LastReceived: myMap["LastReceived"].(string),
		}
		result = append(result, stat)
	}

	return result
}

func generateSubscriberGroupChannelStats(stats map[string]interface{}) []config.SubscriberGroupChannelStats {
	result := make([]config.SubscriberGroupChannelStats, 0)

	for _, value := range stats {
		myMap := value.(map[string]interface{})
		head := myMap["Head"].(string)
		stat := config.SubscriberGroupChannelStats{
			Head:                                 head,
			Channel:                              int64(myMap["Channel"].(float64)),
			OwningSubscriberID:                   int64(myMap["OwningSubscriberId"].(float64)),
			OwningSubscriberMemberID:             int64(myMap["OwningSubscriberMemberId"].(float64)),
			OwningSubscriberMemberNotificationID: int64(myMap["OwningSubscriberMemberNotificationId"].(float64)),
			OwningSubscriberMemberUUID:           myMap["OwningSubscriberMemberUuid"].(string),
			LastCommittedPosition:                myMap["LastCommittedPosition"].(string),
			LastCommittedTimestamp:               myMap["LastCommittedTimestamp"].(string),
			LastPolledTimestamp:                  myMap["LastPolledTimestamp"].(string),
			PolledCount:                          int64(myMap["PolledCount"].(float64)),
			RemainingUnpolledMessages:            int64(myMap["RemainingUnpolledMessages"].(float64)),
			PolledOneMinuteRate:                  myMap["PolledOneMinuteRate"].(float64),
			PolledFiveMinuteRate:                 myMap["PolledFiveMinuteRate"].(float64),
			PolledFifteenMinuteRate:              myMap["PolledFifteenMinuteRate"].(float64),
		}
		result = append(result, stat)
	}

	return result
}

func getTopicsHeader(topicServiceName, topicName string) string {
	return "Service:  " + topicServiceName + "\n" + "Topic:    " + topicName + "\n"
}

// getTopicDetailIndex returns the index of the current topic detail in the array or -1 if not found
func getTopicDetailIndex(topicsDetails config.TopicDetails, topicService, topicName string) int {
	for i, v := range topicsDetails.Details {
		if v.ServiceName == topicService && v.TopicName == topicName {
			return i
		}
	}
	return -1
}

// enrichTopicsSummary enriches the topics details with specific information
func enrichTopicsSummary(topicsDetails *config.TopicDetails, topicsMembers []config.TopicsMemberDetail, topicsSubscribers []config.TopicsSubscriberDetail) {
	for i, topic := range topicsDetails.Details {
		// look at topics details
		for _, detail := range topicsMembers {
			if detail.TopicName == topic.TopicName && detail.ServiceName == topic.ServiceName {
				topicsDetails.Details[i].Members++
				topicsDetails.Details[i].Channels = detail.ChannelCount
				topicsDetails.Details[i].PublishedCount += detail.PublishedCount
			}
		}

		// look at topics subscribers
		for _, subscriber := range topicsSubscribers {
			var subscriberCount int64 = 0
			if subscriber.TopicName == topic.TopicName && subscriber.ServiceName == topic.ServiceName {
				subscriberCount++
			}
			topicsDetails.Details[i].Subscribers += subscriberCount
		}
	}
}

// getTopics returns topics details for a cluster and optionally filtering on the topicsServer if != ""
func getTopics(dataFetcher fetcher.Fetcher, topicsService string) (config.TopicDetails, error) {
	result := config.TopicDetails{}
	topicsResult, err := dataFetcher.GetTopicsJSON()
	if err != nil {
		return result, err
	}
	if len(topicsResult) == 0 {
		return result, nil
	}

	if err = json.Unmarshal(topicsResult, &result); err != nil {
		return result, err
	}

	// no service specified to return all
	if topicsService == "" {
		return result, nil
	}

	// service was specified, so strip out any topics that don't belong to the service
	finalTopics := make([]config.TopicDetail, 0)
	for _, v := range result.Details {
		if v.ServiceName == topicsService {
			finalTopics = append(finalTopics, config.TopicDetail{
				TopicName: v.TopicName, ServiceName: v.ServiceName,
			})
		}
	}
	result.Details = finalTopics

	return result, nil
}

// getTopicsMembers returns topics members details for topics
func getTopicsMembers(dataFetcher fetcher.Fetcher, topics config.TopicDetails) ([]config.TopicsMemberDetail, error) {
	var (
		allTopicsSummary = make([]config.TopicsMemberDetail, 0)
		errorSink        = createErrorSink()
		numServices      = len(topics.Details)
		m                = sync.RWMutex{}
		wg               sync.WaitGroup
	)

	// loop through the topics and build retrieve member details. carry out each service concurrently
	wg.Add(numServices)

	for _, topic := range topics.Details {
		go func(topicServiceName, topicName string) {
			defer wg.Done()
			topicsResult, err := dataFetcher.GetTopicsMembersJSON(topicServiceName, topicName)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					// no topics for this service, so finish with no error
					return
				}
				// must be another error so log it and finish
				errorSink.AppendError(err)
				return
			}

			// no-members
			if len(topicsResult) == 0 {
				return
			}
			topicsSummary := config.TopicsMemberDetails{}
			err = json.Unmarshal(topicsResult, &topicsSummary)
			if err != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshal topics result", err))
				return
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			allTopicsSummary = append(allTopicsSummary, topicsSummary.Details...)
		}(topic.ServiceName, topic.TopicName)
	}

	// wait for the results
	wg.Wait()

	errorList := errorSink.GetErrors()

	if len(errorList) > 0 {
		return nil, utils.GetErrors(errorList)
	}

	return allTopicsSummary, nil
}

// getTopicsSubscribers returns topics subscriber details for topics
func getTopicsSubscribers(dataFetcher fetcher.Fetcher, topics config.TopicDetails) ([]config.TopicsSubscriberDetail, error) {
	var (
		allTopicsSummary = make([]config.TopicsSubscriberDetail, 0)
		errorSink        = createErrorSink()
		numServices      = len(topics.Details)
		m                = sync.RWMutex{}
		wg               sync.WaitGroup
	)

	// loop through the topics and retrieve member details. carry out each service concurrently
	wg.Add(numServices)

	for _, topic := range topics.Details {
		go func(topicServiceName, topicName string) {
			defer wg.Done()
			topicsResult, err := dataFetcher.GetTopicsSubscribersJSON(topicServiceName, topicName)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					// no topics for this service, so finish with no error
					return
				}
				// must be another error so log it and finish
				errorSink.AppendError(err)
				return
			}

			// no-subscribers
			if len(topicsResult) == 0 {
				return
			}
			topicsSummary := config.TopicsSubscriberDetails{}
			err = json.Unmarshal(topicsResult, &topicsSummary)
			if err != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshal topics subscriber result", err))
				return
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			allTopicsSummary = append(allTopicsSummary, topicsSummary.Details...)
		}(topic.ServiceName, topic.TopicName)
	}

	// wait for the results
	wg.Wait()

	errorList := errorSink.GetErrors()

	if len(errorList) > 0 {
		return nil, utils.GetErrors(errorList)
	}

	return allTopicsSummary, nil
}

// getTopicsSubscriberGroups returns topics subscriber group details for topics
func getTopicsSubscriberGroups(dataFetcher fetcher.Fetcher, topics config.TopicDetails) ([]config.TopicsSubscriberGroupDetail, error) {
	var (
		allSubscriberGroupSummary = make([]config.TopicsSubscriberGroupDetail, 0)
		errorSink                 = createErrorSink()
		numServices               = len(topics.Details)
		m                         = sync.RWMutex{}
		wg                        sync.WaitGroup
	)

	// loop through the topics and build retrieve member details. carry out each service concurrently
	wg.Add(numServices)

	for _, topic := range topics.Details {
		go func(topicServiceName, topicName string) {
			defer wg.Done()
			topicsResult, err := dataFetcher.GetTopicsSubscriberGroupsJSON(topicServiceName, topicName)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					// no topics for this service, so finish with no error
					return
				}
				// must be another error so log it and finish
				errorSink.AppendError(err)
				return
			}

			// no-subscriber groups
			if len(topicsResult) == 0 {
				return
			}
			subscriberGroupSummary := config.TopicsSubscriberGroups{}
			err = json.Unmarshal(topicsResult, &subscriberGroupSummary)
			if err != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshal topics subscriber groups result", err))
				return
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			allSubscriberGroupSummary = append(allSubscriberGroupSummary, subscriberGroupSummary.Details...)
		}(topic.ServiceName, topic.TopicName)
	}

	// wait for the results
	wg.Wait()

	errorList := errorSink.GetErrors()

	if len(errorList) > 0 {
		return nil, utils.GetErrors(errorList)
	}

	return allSubscriberGroupSummary, nil
}

// processJSONOutput processes JSON output and either outputs the JSONPath or JSON results.
func processJSONOutput(cmd *cobra.Command, jsonData []byte) error {
	var (
		err    error
		result string
	)
	if OutputFormat == constants.JSONPATH {
		result, err = utils.GetJSONPathResults(jsonData, OutputFormat)
		if err != nil {
			return err
		}
		cmd.Println(result)
		return nil
	}
	cmd.Println(string(jsonData))
	return nil
}

func init() {
	getTopicsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)

	getSubscribersCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getSubscribersCmd.MarkFlagRequired(serviceNameOption)

	getSubscriberGroupsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getSubscriberGroupsCmd.MarkFlagRequired(serviceNameOption)

	describeTopicCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = describeTopicCmd.MarkFlagRequired(serviceNameOption)

	getTopicMembersCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getTopicMembersCmd.MarkFlagRequired(serviceNameOption)

	getMemberChannelsCmd.Flags().Int32VarP(&topicsNodeID, "node", "n", 0, nodeIDMessage)
	_ = getMemberChannelsCmd.MarkFlagRequired("node")
	getMemberChannelsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getMemberChannelsCmd.MarkFlagRequired(serviceNameOption)

	getSubscriberChannelsCmd.Flags().Int32VarP(&topicsNodeID, "node", "n", 0, nodeIDMessage)
	_ = getSubscriberChannelsCmd.MarkFlagRequired("node")
	getSubscriberChannelsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getSubscriberChannelsCmd.MarkFlagRequired(serviceNameOption)
	getSubscriberChannelsCmd.Flags().Int64VarP(&subscriber, "subscriber", "S", 0, "subscriber id")
	_ = getSubscriberChannelsCmd.MarkFlagRequired("subscriber")

	getSubscriberGroupChannelsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = getSubscriberGroupChannelsCmd.MarkFlagRequired(serviceNameOption)
	getSubscriberGroupChannelsCmd.Flags().StringVarP(&subscriberGroup, "subscriber-group", "G", "", "subscriber group")
	_ = getSubscriberGroupChannelsCmd.MarkFlagRequired("subscriber-group")
	getSubscriberGroupChannelsCmd.Flags().Int32VarP(&topicsNodeID, "node", "n", 0, nodeIDMessage)
	_ = getSubscriberGroupChannelsCmd.MarkFlagRequired("node")
}
