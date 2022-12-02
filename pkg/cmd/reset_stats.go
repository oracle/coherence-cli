/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
	"sync"
)

var (
	resetNodeIDs string
)

// resetStatsCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "reset statistics for various resources",
	Long:  `The 'reset-stats' command resets statistics for various resources.`,
}

// resetMemberStatsCmd represents the reset member-stats command
var resetMemberStatsCmd = &cobra.Command{
	Use:   "member-stats",
	Short: "reset statistics for all or a specific member",
	Long:  `The 'reset member-stats' command resets member statistics for all or a comma separated list of member IDs.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetMembers)
	},
}

// resetReporterStatsCmd represents the reset reporter-stats command
var resetReporterStatsCmd = &cobra.Command{
	Use:   "reporter-stats",
	Short: "reset reporter statistics for all or a specific reporter",
	Long:  `The 'reset reporter-stats' command resets reporter statistics for all or a comma separated list of member IDs.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetReporters)
	},
}

// resetRAMJournalStatsCmd represents the reset ramjournal-stats command
var resetRAMJournalStatsCmd = &cobra.Command{
	Use:   "ramjournal-stats",
	Short: "reset statistics for all ram journals",
	Long:  `The 'reset ramjournal-stats' command resets ram journal statistics.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetRAMJournal)
	},
}

// resetFlashJournalStatsCmd represents the reset flashjournal-stats command
var resetFlashJournalStatsCmd = &cobra.Command{
	Use:   "flashjournal-stats",
	Short: "reset statistics for all flash journals",
	Long:  `The 'reset flashjournal-stats' command resets flash journal statistics.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetFlashJournal)
	},
}

// resetExecutorStatsCmd represents the reset executor-stats command
var resetExecutorStatsCmd = &cobra.Command{
	Use:   "executor-stats executor-name",
	Short: "reset statistics for an executor",
	Long:  `The 'reset executor-stats' command resets executor statistics for a specific executor.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideExecutorName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetExecutor)
	},
}

// resetServiceStatsCmd represents the reset service-stats command
var resetServiceStatsCmd = &cobra.Command{
	Use:   "service-stats service-name",
	Short: "reset services statistics for all service members or specific service members",
	Long:  `The 'reset service-stats' command resets service statistics for all service or a comma separated list of member IDs.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideServiceName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetService)
	},
}

// resetCacheStatsCmd represents the reset cache-stats command
var resetCacheStatsCmd = &cobra.Command{
	Use:   "cache-stats cache-name",
	Short: "reset cache statistics for all cache members or specific cache members",
	Long:  `The 'reset cache-stats' command resets cache statistics for all cache members or a comma separated list of member IDs.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetCache)
	},
}

// resetFederationStatsCmd represents the reset federation-stats command
var resetFederationStatsCmd = &cobra.Command{
	Use:   "federation-stats service-name",
	Short: "reset federation statistics for all federation or specific federation members",
	Long:  `The 'reset federation-stats' command resets federation statistics for all members or a comma separated list of member IDs.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideServiceName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueResetStatsCommand(cmd, args, fetcher.ResetFederation)
	},
}

// issueResetStatsCommand validates the resetStatistics command
func issueResetStatsCommand(cmd *cobra.Command, args []string, operation string) error {
	var (
		confirmMessage string
		err            error
		dataFetcher    fetcher.Fetcher
		connection     string
		nodeIDArray    []string
		nodeIds        []string
		message        string
	)

	// retrieve the current context or the value from "-c"
	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	// validate the nodes
	nodeIDArray, err = GetNodeIds(dataFetcher)
	if err != nil {
		return err
	}

	if resetNodeIDs == "all" {
		nodeIds = append(nodeIds, nodeIDArray...)
		confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIds))
	} else {
		nodeIds = strings.Split(resetNodeIDs, ",")
		for _, value := range nodeIds {
			if !utils.IsValidInt(value) {
				return fmt.Errorf("invalid value for node id of %s", value)
			}

			if !utils.SliceContains(nodeIDArray, value) {
				return fmt.Errorf("no node with node id %s exists in this cluster", value)
			}
		}
		confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIds))
	}

	cmd.Println(FormatCurrentCluster(connection))

	// validate the operation can be carried out on this version
	err = isOperationSupported(operation, resetNodeIDs, dataFetcher)
	if err != nil {
		return err
	}

	// do extra validation
	if operation == fetcher.ResetRAMJournal || operation == fetcher.ResetFlashJournal {
		var (
			queryType = "ram"
			result    []byte
		)
		if operation == fetcher.ResetFlashJournal {
			queryType = "flash"
		}

		result, err = dataFetcher.GetElasticDataDetails(queryType)
		if err != nil {
			return err
		}

		if len(result) == 0 {
			return fmt.Errorf("no journals of type %s exist in this cluster", queryType)
		}
	}

	if operation == fetcher.ResetCache {
		var (
			found       bool
			cacheResult []byte
		)

		// validate service name
		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf("unable to find service with service name '%s'", serviceName)
		}

		// validate cache and service name
		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, args[0])
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" || len(cacheResult) == 0 {
			return fmt.Errorf("no cache named %s exists for service %s", args[0], serviceName)
		}

		message = fmt.Sprintf("Are you sure you want to reset %s statistics for cache %s, service %s for %s? (y/n) ",
			operation, args[0], serviceName, confirmMessage)

		// reset the args to include cache and service
		args = []string{args[0], serviceName}
	} else if operation == fetcher.ResetFederation {
		var (
			found bool
		)

		// validate service name
		found, err = ServiceExists(dataFetcher, args[0])
		if !found || err != nil {
			return fmt.Errorf("unable to find service with service name '%s'", args[0])
		}

		message = fmt.Sprintf("Are you sure you want to reset %s statistics for service %s, participant %s, type %s for %s? (y/n) ",
			operation, args[0], participant, describeFederationType, confirmMessage)

		// reset the args to include service, participant and type
		args = []string{args[0], participant, describeFederationType}
	} else if operation == fetcher.ResetExecutor {
		var (
			executors      config.Executors
			finalExecutors = config.Executors{}
		)
		// validate the executor

		executors, err = getExecutorDetails(dataFetcher, false)
		if err != nil {
			return err
		}

		if len(executors.Executors) == 0 {
			return errors.New(cannotFindExecutors)
		}

		// filter out any executors without the name
		for _, value := range executors.Executors {
			if value.Name == args[0] {
				finalExecutors.Executors = append(finalExecutors.Executors, value)
			}
		}

		if len(finalExecutors.Executors) == 0 {
			return fmt.Errorf("unable to find executor with name %s", args[0])
		}

		message = fmt.Sprintf("Are you sure you want to reset %s statistics for exeutor %s? (y/n) ", operation, args[0])

		// force resetNodeID to "all"
		resetNodeIDs = "all"
	} else {
		message = fmt.Sprintf("Are you sure you want to reset %s statistics for %s? (y/n) ", operation, confirmMessage)
	}

	// confirm the operation
	if !confirmOperation(cmd, message) {
		return nil
	}

	// for operations for all members, these can be done using one operation and for others,
	// they must be done in parallel. Reset federation can only be done via nodeID
	if resetNodeIDs == "all" && operation != fetcher.ResetFederation {
		_, err = dataFetcher.InvokeResetStatistics(operation, "all", args)
		if err != nil {
			return err
		}
	} else {
		// carry out the node changes concurrently and wait for the results
		var (
			errorSink = createErrorSink()
			wg        sync.WaitGroup
		)

		wg.Add(len(nodeIds))

		for _, value := range nodeIds {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.InvokeResetStatistics(operation, nodeId, args)
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
	}

	if err != nil {
		return err
	}

	cmd.Println(OperationCompleted)

	return nil
}

func init() {
	setResetFlags(resetMemberStatsCmd)
	setResetFlags(resetReporterStatsCmd)
	setResetFlags(resetServiceStatsCmd)
	setResetFlags(resetCacheStatsCmd)
	setResetFlags(resetFederationStatsCmd)

	resetRAMJournalStatsCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	resetFlashJournalStatsCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	resetExecutorStatsCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	resetCacheStatsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = resetCacheStatsCmd.MarkFlagRequired(serviceNameOption)

	resetFederationStatsCmd.Flags().StringVarP(&participant, "participant", "p", "all", participantMessage)
	resetFederationStatsCmd.Flags().StringVarP(&describeFederationType, "type", "T", outgoing, "type to describe "+outgoing+" or "+incoming)
	_ = resetFederationStatsCmd.MarkFlagRequired("participant")
	_ = resetFederationStatsCmd.MarkFlagRequired("type")

}

// setResetFlags sets common flags for reset operations
func setResetFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&resetNodeIDs, "node", "n", "all", commaSeparatedIDMessage)
	cmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}

// isOperationSupported returns true if the operation is supported for the given coherence
// version prefix. Some operations were not introduced in 14.1.1.0 and 12.2.1.X releases
func isOperationSupported(operation, resetNodeIDs string, dataFetcher fetcher.Fetcher) error {
	var (
		err            error
		cluster        = config.Cluster{}
		clusterResult  []byte
		clusterVersion string
	)

	// get the cluster details to retrieve the Coherence version
	clusterResult, err = dataFetcher.GetClusterDetailsJSON()
	if err != nil {
		return err
	}

	err = json.Unmarshal(clusterResult, &cluster)
	if err != nil {
		return utils.GetError("unable to decode cluster details", err)
	}

	clusterVersion = cluster.Version

	// version 14.1.1.0.X and 12.2.1.x versions have limited support for resetStatistics
	// all other versions are assumed to be supported
	if strings.Contains(clusterVersion, "14.1.1"+".0") || strings.Contains(clusterVersion, "12.2.1.") {
		if operation == fetcher.ResetMembers && resetNodeIDs == "all" {
			return fmt.Errorf("you can only reset member statistics for an individual node in Coherence version %s", clusterVersion)
		}
		if operation == fetcher.ResetRAMJournal || operation == fetcher.ResetFlashJournal {
			return fmt.Errorf("you cannot reset flash or ram journal in Coherence version %s", clusterVersion)
		}
		if operation == fetcher.ResetService && resetNodeIDs == "all" {
			return fmt.Errorf("you can only reset service statistics for an individual node in Coherence version %s", clusterVersion)
		}
		if operation == fetcher.ResetCache && resetNodeIDs == "all" {
			return fmt.Errorf("you can only reset cache statistics for an individual node in Coherence version %s", clusterVersion)
		}
		if operation == fetcher.ResetExecutor {
			return fmt.Errorf("you cannot reset executor statistics in Coherence version %s", clusterVersion)
		}
	}

	return nil
}
