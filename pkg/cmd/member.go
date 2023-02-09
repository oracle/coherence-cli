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
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	numThreadDumps    int32
	delayBetweenDumps int32
	outputDirectory   string
	verbose           bool

	dumpRoleName    string
	configureRole   string
	roleName        string
	threadDumpRole  string
	threadDump      bool
	extendedInfo    string
	attributeName   string
	attributeValue  string
	validAttributes = []string{"loggingLevel", "resendDelay", "sendAckDelay",
		"trafficJamCount", "trafficJamDelay", "loggingLimit,", "loggingFormat"}

	tracingRatio float32
)

const dumpClusterHeap = "dump cluster heap"
const logClusterState = "log cluster state"
const configureTracing = "configure tracing"
const unableToDecode = "unable to decode member details"
const noNodeID = "no node with node id %s exists in this cluster"
const invalidNodeID = "invalid value for node id of %s"
const none = "none"
const roleNameDescription = "role name to display"

// getMembersCmd represents the get members command
var getMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "display members for a cluster",
	Long: `The 'get members' command displays the members for a cluster. You
can specify '-o wide' to display addition information.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return getMembers(cmd, false)
	},
}

// getMembers returns the member or network stats depending upon the value of networkStats.
func getMembers(cmd *cobra.Command, networkStats bool) error {
	var (
		dataFetcher fetcher.Fetcher
		connection  string
		result      string
		err         error
	)

	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	for {
		var (
			members       = config.Members{}
			storage       = config.StorageDetails{}
			membersResult []byte
			storageResult []byte
		)

		membersResult, err = dataFetcher.GetMemberDetailsJSON(OutputFormat != constants.TABLE && OutputFormat != constants.WIDE)
		if err != nil {
			return err
		}

		storageResult, err = dataFetcher.GetStorageDetailsJSON()
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err = utils.GetJSONPathResults(membersResult, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(membersResult))
		} else {
			printWatchHeader(cmd)

			cmd.Println(FormatCurrentCluster(connection))
			err = json.Unmarshal(membersResult, &members)
			if err != nil {
				return utils.GetError(unableToDecode, err)
			}

			err = json.Unmarshal(storageResult, &storage)
			if err != nil {
				return utils.GetError("unable to decode storage details", err)
			}

			storageMap := utils.GetStorageMap(storage)

			var filteredMembers []config.Member

			// apply any filtering by role
			if roleName != all {
				filteredMembers = make([]config.Member, 0)
				for _, value := range members.Members {
					if value.RoleName == roleName {
						filteredMembers = append(filteredMembers, value)
					}
				}
			} else {
				filteredMembers = make([]config.Member, len(members.Members))
				copy(filteredMembers, members.Members)
			}

			if networkStats {
				cmd.Println(FormatNetworkStatistics(filteredMembers))
			} else {
				cmd.Print(FormatMembers(filteredMembers, true, storageMap))
			}
		}

		// check to see if we should exit if we are not watching
		if !isWatchEnabled() {
			break
		}
		// we are watching services so sleep and then repeat until CTRL-C
		time.Sleep(time.Duration(watchDelay) * time.Second)
	}

	return nil
}

// getMemberStats represents the get network-stats command
var getMemberStats = &cobra.Command{
	Use:   "member-stats",
	Short: "display member network statistics for a cluster",
	Long:  `The 'get member-stats' command displays the member network statistic for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return getMembers(cmd, true)
	},
}

// describeMemberCmd represents the describe member command
var describeMemberCmd = &cobra.Command{
	Use:   "member node-id",
	Short: "describe a member",
	Long: `The 'describe member' command shows information related to a specific member.
To display extended information about a member, the -X option can be specified with a comma
separated list of platform entries to display for. For example:

   cohctl describe member 1 -X g1OldGeneration,g1EdenSpace

would display information related to G1 old generation and Eden space. 

Full list of options are JVM dependant, but can include the full values or part of the following:
  compressedClassSpace, operatingSystem, metaSpace, g1OldGen, g1SurvivorSpace, g1CodeHeapProfiledNMethods, 
  g1CodeHeapNonNMethods, g1OldGeneration g1MetaSpaceManager, g1YoungGeneration, g1EdenSpace,
  g1CodeCacheManager, psScavenge, psEdenSpace, psMarkSweep, codeCache, psOldGen, psSurvivorSpace,
  runtime
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			members          = config.Members{}
			result           []byte
			err              error
			dataFetcher      fetcher.Fetcher
			extendedData     [][]byte
			connection       string
			finalResult      []byte
			finalExtended    []byte
			extendedInfoList []string
		)

		nodeID := args[0]
		if !utils.IsValidInt(nodeID) {
			return fmt.Errorf("invalid node id %s", nodeID)
		}

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		result, err = dataFetcher.GetMemberDetailsJSON(false)
		if err != nil {
			return err
		}

		err = json.Unmarshal(result, &members)
		if err != nil {
			return err
		}

		// check to see the member is valid
		var found bool

		for _, value := range members.Members {
			if value.NodeID == nodeID {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unable to find member with nodeId = %s", nodeID)
		}

		// we have a valid member id so get the details from fetcher
		result, err = dataFetcher.GetSingleMemberDetailsJSON(nodeID)
		if err != nil {
			return err
		}

		// retrieve the links for the extended info
		if extendedInfo != none {
			// retrieve the extended data
			extendedInfoList = strings.Split(extendedInfo, ",")
			extendedData, err = dataFetcher.GetExtendedMemberInfoJSON(result, nodeID, extendedInfoList)
			if err != nil {
				return err
			}
		}

		if isJSONPathOrJSON() {
			if len(extendedData) > 0 {
				finalExtended, err = utils.CombineByteArraysForJSON(extendedData, extendedInfoList)
				if err != nil {
					return err
				}
			}
			finalResult, err = utils.CombineByteArraysForJSON([][]byte{result, finalExtended}, []string{"member", "extended"})
			if err != nil {
				return err
			}

			if err = processJSONOutput(cmd, finalResult); err != nil {
				return err
			}
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			cmd.Println("MEMBER DETAILS")
			cmd.Println("--------------")
			value, err := FormatJSONForDescribe(result, true, "Node Id", "Unicast Address",
				"Role Name", "Machine Name", "Rack Name", "Site Name")
			if err != nil {
				return err
			}
			cmd.Println(value)

			if threadDump {
				data, err := dataFetcher.GetThreadDump(nodeID)
				if err != nil {
					return err
				}

				threadDump, err := UnmarshalThreadDump(data)
				if err != nil {
					return err
				}
				cmd.Println("\nTHREAD DUMP")
				cmd.Println("-----------")
				cmd.Println(threadDump)
			}

			if extendedInfo != none {
				var extendedValue string
				cmd.Println("\nEXTENDED INFORMATION")
				cmd.Println("--------------------")
				// add any extended data
				if len(extendedData) > 0 {
					for _, value := range extendedData {
						if len(value) > 0 {
							extendedValue, err = FormatJSONForDescribe(value, true)
							if err != nil {
								return err
							}
							cmd.Println(extendedValue)
						}
					}
				}
			}
		}

		return nil
	},
}

// setMemberCmd represents the set member command
var setMemberCmd = &cobra.Command{
	Use:   "member {node-ids|all}",
	Short: "set a member attribute for one or more members",
	Long: `The 'set member' command sets an attribute for one or more member nodes.
You can specify 'all' to change the value for all nodes, or specify a comma separated
list of node ids. The following attribute names are allowed:
loggingLevel, resendDelay, sendAckDelay, trafficJamCount, trafficJamDelay, loggingLimit
or loggingFormat.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a comma separated list of node id's or 'all' for all nodes")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher    fetcher.Fetcher
			connection     string
			err            error
			nodeIds        []string
			nodeIDArray    []string
			confirmMessage string
			errorSink      = createErrorSink()
			wg             sync.WaitGroup
			loggingNodeIds = args[0]
			actualValue    interface{}
			intValue       int
		)

		if !utils.SliceContains(validAttributes, attributeName) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of\n%v",
				attributeName, validAttributes)
		}

		if attributeName == "loggingFormat" {
			// this is the only attribute that is a string
			actualValue = attributeValue
		} else {
			// convert to an int
			intValue, err = strconv.Atoi(attributeValue)
			if err != nil {
				return fmt.Errorf("invalid integer value of %s for attribute %s", attributeValue, attributeName)
			}

			actualValue = intValue
			// carry out some basic validation
			if attributeName == "loggingLevel" && (intValue < 1 || intValue > 9) {
				return fmt.Errorf("log level must be betweeen 1 and 9")
			} else if intValue <= 0 {
				return fmt.Errorf("value for attribute %s must be greater than zero", attributeName)
			}

		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if loggingNodeIds == all {
			nodeIds = append(nodeIds, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIds))
		} else {
			if nodeIds, err = getNodeIDs(loggingNodeIds, nodeIDArray); err != nil {
				return err
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIds))
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
			attributeName, attributeValue, confirmMessage)) {
			return nil
		}

		nodeCount := len(nodeIds)
		wg.Add(nodeCount)

		for _, value := range nodeIds {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.SetMemberAttribute(nodeId, attributeName, actualValue)
				if err1 != nil {
					errorSink.AppendError(err1)
				}
			}(value)
		}

		wg.Wait()
		errorList := errorSink.GetErrors()

		if len(errorList) > 0 {
			return utils.GetErrors(errorList)
		}
		cmd.Println(OperationCompleted)

		return nil
	},
}

// dumpClusterHeapCmd represents the dump cluster-heap command
var dumpClusterHeapCmd = &cobra.Command{
	Use:   "cluster-heap",
	Short: "dump the cluster heap for all members or a specific role",
	Long: `The 'dump cluster-heap' command issues a heap dump for all members or the selected role
by using the -r flag.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueClusterCommand(cmd, dumpClusterHeap)
	},
}

// configureTracingCmd represents the configure tracing command
var configureTracingCmd = &cobra.Command{
	Use:   "tracing",
	Short: "configure tracing for all members or a specific role",
	Long: `The 'configure tracing' command configures tracing for all members or the selected role
by using the -r flag. You can specify a tracingRatio of -1 to turn off tracing.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueClusterCommand(cmd, configureTracing)
	},
}

// getTracingCmd represents the get tracing command
var getTracingCmd = &cobra.Command{
	Use:   "tracing",
	Short: "display tracing for all members",
	Long:  `The 'get tracing' command displays tracing status for all members.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
			members     = config.Members{}
		)

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			printWatchHeader(cmd)

			membersResult, err := dataFetcher.GetMemberDetailsJSON(OutputFormat != constants.TABLE && OutputFormat != constants.WIDE)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(membersResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(membersResult))
			} else {
				cmd.Println(FormatCurrentCluster(connection))
				err = json.Unmarshal(membersResult, &members)
				if err != nil {
					return utils.GetError(unableToDecode, err)
				}

				cmd.Print(FormatTracing(members.Members))
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching services so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// logClusterStateCmd represents the log cluster-state command
var logClusterStateCmd = &cobra.Command{
	Use:   "cluster-state",
	Short: "Logs the cluster state via a thread dump, for all members or a specific role",
	Long: `The 'log cluster-state' command logs a full thread dump and outstanding
polls, in the logs files, for all members or the selected role by using the -r flag.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueClusterCommand(cmd, logClusterState)
	},
}

// shutdownMemberCmd represents the shutdown member command
var shutdownMemberCmd = &cobra.Command{
	Use:   "member node-id",
	Short: "shutdown a members services",
	Long: `The 'shutdown member' command shuts down all the clustered services that are
running on a specific member via a controlled shutdown. If the services were started using
DefaultCacheServer, then they will be restarted.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
			nodeIDArray []string
			nodeID      = args[0]
		)

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if !utils.IsValidInt(nodeID) {
			return fmt.Errorf(invalidNodeID, nodeID)
		}

		if !utils.SliceContains(nodeIDArray, nodeID) {
			return fmt.Errorf(noNodeID, nodeID)
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to shutdown member %s? (y/n) ", nodeID)) {
			return nil
		}

		_, err = dataFetcher.ShutdownMember(nodeID)
		if err != nil {
			return err
		}

		cmd.Println(OperationCompleted)
		return nil
	},
}

// issueClusterCommand issues a variety of cluster commands
func issueClusterCommand(cmd *cobra.Command, command string) error {
	var (
		dataFetcher   fetcher.Fetcher
		connection    string
		membersResult []byte
		err           error
		members       = config.Members{}
		roleCount     = 0
		message       string
		tracing       = ""
		role          = dumpRoleName
	)

	// retrieve the current context or the value from "-c"
	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	membersResult, err = dataFetcher.GetMemberDetailsJSON(false)
	if err != nil {
		return err
	}

	err = json.Unmarshal(membersResult, &members)
	if err != nil {
		return utils.GetError(unableToDecode, err)
	}

	cmd.Println(FormatCurrentCluster(connection))

	if command == configureTracing {
		role = configureRole
		// validate the value for tracingRatio
		if tracingRatio != -1.0 &&
			(tracingRatio < 0 || tracingRatio > 1.0) {
			return fmt.Errorf("tracing ratio must be either -1.0 or between 0 and 1.0")
		}
		tracing = fmt.Sprintf(" to tracing ratio %v", tracingRatio)
	}

	if role != all && role != "" {
		// validate the role
		for _, value := range members.Members {
			if value.RoleName == role {
				roleCount++
			}
		}
		if roleCount == 0 {
			return fmt.Errorf("unable to find members with role name of %s", role)
		}
		message = fmt.Sprintf("%d members with role %s", roleCount, role)
	} else {
		message = fmt.Sprintf("all %d members", len(members.Members))
	}

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to %s%s for %s? (y/n) ", command, tracing, message)) {
		return nil
	}

	if command == dumpClusterHeap {
		_, err = dataFetcher.DumpClusterHeap(role)
	} else if command == configureTracing {
		_, err = dataFetcher.ConfigureTracing(role, tracingRatio)
	} else {
		_, err = dataFetcher.LogClusterState(role)
	}
	if err != nil {
		return err
	}

	cmd.Println("Operation completed. Please see cache server log file for more information")

	return nil
}

// retrieveThreadDumpsCmd represents the retrieve thread-dumps command
var retrieveThreadDumpsCmd = &cobra.Command{
	Use:   "thread-dumps [node-ids]",
	Short: "generate and retrieve thread dumps for all or selected members",
	Long: `The 'get thead-dumps' command generates and retrieves thread dumps for all or selected 
members and places them in the specified output directory. You are also able to specify 
a role to retrieve thread dumps for.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if threadDumpRole == all && len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a comma separated list of node id's, 'all' for all nodes, or specify a role")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher   fetcher.Fetcher
			connection    string
			err           error
			nodeIds       []string
			nodesToDump   string
			wg            sync.WaitGroup
			errorSink     = createErrorSink()
			nodeIDArray   []string
			membersResult []byte
			members       = config.Members{}
		)

		if len(args) != 0 {
			nodesToDump = args[0]
		}

		if delayBetweenDumps < 5 {
			return errors.New("delay must be 5 seconds or above")
		}

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		// validate the output directory
		if !utils.DirectoryExists(outputDirectory) {
			return fmt.Errorf("the output directory '%s' does not exist or is not writabled", outputDirectory)
		}

		// retrieve the nodes to validate against
		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		// retrieve and validate the members
		if nodesToDump == all {
			nodeIds = append(nodeIds, nodeIDArray...)
		} else {
			if threadDumpRole != all {
				// a role other than "all" has been specified so get all the nodes
				// that match the role
				membersResult, err = dataFetcher.GetMemberDetailsJSON(false)
				if err != nil {
					return err
				}

				err = json.Unmarshal(membersResult, &members)
				if err != nil {
					return utils.GetError(unableToDecode, err)
				}

				for _, member := range members.Members {
					if member.RoleName == threadDumpRole {
						nodeIds = append(nodeIds, member.NodeID)
					}
				}
				if len(nodeIds) == 0 {
					return fmt.Errorf("unable to find any nodes with role %s", threadDumpRole)
				}
			} else {
				nodeIds = strings.Split(nodesToDump, ",")
				for _, value := range nodeIds {
					if !utils.IsValidInt(value) {
						return fmt.Errorf(invalidNodeID, value)
					}

					if !utils.SliceContains(nodeIDArray, value) {
						return fmt.Errorf(noNodeID, value)
					}
				}
			}
		}

		var numNodes = len(nodeIds)

		if numNodes >= 4 {
			cmd.Printf("Warning: running thread dump in parallel across %d nodes may cause excessive load.\n",
				numNodes)
		}

		cmd.Printf("This operation will take at least %d seconds.\n", (numThreadDumps-1)*delayBetweenDumps)

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to retrieve %d thread dumps, each %d seconds apart for %d node(s)? (y/n) ",
			numThreadDumps, delayBetweenDumps, len(nodeIds))) {
			return nil
		}

		// run the thread dump captures in parallel for each node
		nodeCount := len(nodeIds)

		wg.Add(nodeCount)
		for i, value := range nodeIds {
			go func(n string, last bool) {
				defer wg.Done()
				err1 := generateThreadDumps(n, dataFetcher, cmd, last)
				if err1 != nil {
					errorSink.AppendError(err1)
				}
			}(value, i == nodeCount-1)
		}

		wg.Wait()
		errorList := errorSink.GetErrors()

		if len(errorList) == 0 {
			cmd.Println("\nAll thread dumps completed and written to " + outputDirectory)
		} else if len(errorList) == 1 {
			return errorList[0]
		} else {
			// multiple errors
			for _, value := range errorList {
				cmd.Println(value)
			}
			return errors.New("multiple errors occurred, see the log file")
		}

		return nil
	},
}

// generateThreadDumps generates the required number of thread dumps for a node
func generateThreadDumps(nodeID string, dataFetcher fetcher.Fetcher,
	cmd *cobra.Command, isLast bool) error {
	var (
		i          int32
		fileName   string
		data       []byte
		err        error
		file       *os.File
		threadDump string
	)

	for i = 1; i <= numThreadDumps; i++ {
		fileName = filepath.Join(outputDirectory, GetFileName(nodeID, i))
		data, err = dataFetcher.GetThreadDump(nodeID)
		if err != nil {
			return utils.GetError("unable to get thread dump for node "+nodeID, err)
		}

		threadDump, err = UnmarshalThreadDump(data)
		if err != nil {
			return err
		}

		// write the thread dump to the file
		file, err = os.Create(fileName)
		if err != nil {
			return err
		}
		_, err = file.WriteString(threadDump)
		if err != nil {
			return err
		}
		_ = file.Close()

		// display progress
		message := fmt.Sprintf("Completed %d of %d (%3.2f%%)", i, numThreadDumps, float32(i)/float32(numThreadDumps)*100)
		if verbose {
			cmd.Printf("Thread dump iteration %d for node %s written to %s\n", i, nodeID, fileName)
			if isLast {
				cmd.Println(message)
			}
		} else if isLast {
			if isWindows() {
				cmd.Println(message)
			} else {
				cmd.Print(fmt.Sprintf("\033[G\033[K%s", message))
			}
			if numThreadDumps == i {
				cmd.Print()
			}
		}

		if i != numThreadDumps {
			time.Sleep(time.Duration(delayBetweenDumps) * time.Second)
		}
	}
	return nil
}

// GetFileName returns a file name for the thread dump
func GetFileName(nodeID string, iteration int32) string {
	return fmt.Sprintf("thread-dump-node-%s-%d.log", nodeID, iteration)
}

func init() {
	var roleDescription = "role name to run for"

	getMembersCmd.Flags().StringVarP(&roleName, "role", "r", all, roleNameDescription)

	getMemberStats.Flags().StringVarP(&roleName, "role", "r", all, roleNameDescription)

	describeMemberCmd.Flags().BoolVarP(&threadDump, "thread-dump", "D", false, "include a thread dump")
	describeMemberCmd.Flags().StringVarP(&extendedInfo, "extended", "X", none, "include extended information such as g1OldGen, etc. See --help")

	setMemberCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setMemberCmd.Flags().StringVarP(&attributeName, "attribute", "a", "", "attribute name to set")
	_ = setMemberCmd.MarkFlagRequired("attribute")
	setMemberCmd.Flags().StringVarP(&attributeValue, "value", "v", "", "attribute value to set")
	_ = setMemberCmd.MarkFlagRequired("value")

	dumpClusterHeapCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	dumpClusterHeapCmd.Flags().StringVarP(&dumpRoleName, "role", "r", all, roleDescription)

	logClusterStateCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	logClusterStateCmd.Flags().StringVarP(&dumpRoleName, "role", "r", all, roleDescription)

	retrieveThreadDumpsCmd.Flags().Int32VarP(&numThreadDumps, "number", "n", 5, "number of thread dumps to retrieve")
	retrieveThreadDumpsCmd.Flags().Int32VarP(&delayBetweenDumps, "dump-delay", "D", 10, "delay between each thread dump")
	retrieveThreadDumpsCmd.Flags().StringVarP(&outputDirectory, "output-dir", "O", "", "existing directory to output thread dumps to")
	retrieveThreadDumpsCmd.Flags().StringVarP(&threadDumpRole, "role", "r", all, roleDescription)
	_ = retrieveThreadDumpsCmd.MarkFlagRequired("output-dir")
	retrieveThreadDumpsCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	retrieveThreadDumpsCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "produces verbose output")

	configureTracingCmd.Flags().StringVarP(&configureRole, "role", "r", "", "role name to configure tracing for")
	configureTracingCmd.Flags().Float32VarP(&tracingRatio, "tracingRatio", "t", 1.0, "tracing ratio to set. -1.0 turns off tracing")
	_ = configureTracingCmd.MarkFlagRequired("tracingRatio")
	configureTracingCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	shutdownMemberCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}
