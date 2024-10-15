/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
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

const (
	reporterConfigFile      = "configFile"
	reporterCurrentBatch    = "currentBatch"
	reporterIntervalSeconds = "intervalSeconds"
	reporterOutputPath      = "outputPath"
	reporterUse             = "reporter node-id"
	provideNodeID           = "you must provide a node id"
)

var (
	validReporterAttributes = []string{reporterConfigFile, reporterCurrentBatch, reporterIntervalSeconds, reporterOutputPath}
	reporterAttributeName   string
	reporterAttributeValue  string
	reporterNodID           int
)

// getReportersCmd represents the get reporters command.
var getReportersCmd = &cobra.Command{
	Use:   "reporters",
	Short: "display reporters for a cluster",
	Long:  `The 'get reporters' command displays the reporters for the cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var reporters = config.Reporters{}

			reportersResult, err := dataFetcher.GetReportersJSON()
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(reportersResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(reportersResult))
			} else {
				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))
				err = json.Unmarshal(reportersResult, &reporters)
				if err != nil {
					return utils.GetError("unable to unmarshall reporter result", err)
				}

				cmd.Print(FormatReporters(reporters.Reporters))
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

// describeReporterCmd represents the describe reporter command.
var describeReporterCmd = &cobra.Command{
	Use:               reporterUse,
	Short:             "describe a reporter",
	Long:              `The 'describe reporter' command shows information related to a particular reporter.`,
	ValidArgsFunction: completionNodeID,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideNodeID)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			jsonData    []byte
			err         error
			dataFetcher fetcher.Fetcher
			connection  string
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

		jsonData, err = dataFetcher.GetReporterJSON(nodeID)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			jsonPathResult, err := utils.GetJSONPathResults(jsonData, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(jsonPathResult)
			return nil
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(jsonData))
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			cmd.Println("REPORTER DETAILS")
			cmd.Println("---------------")

			value, err := FormatJSONForDescribe(jsonData, true, "Node Id")
			if err != nil {
				return err
			}
			cmd.Println(value)
		}

		return nil
	},
}

// runReportCmd represents the run report command.
var runReportCmd = &cobra.Command{
	Use:   "report report-name",
	Short: "run a report and return the output",
	Long: `The 'run report' command runs a report on a specific node and returns the report output in JSON. 
The report name should not include the .xml extension and will have the 'report' prefix added. E.g. 
'report-node' will expand to 'reports/report-node.xml'. A HTTP 400 will be returned if the report name is not valid.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a report name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			jsonData    []byte
			err         error
			dataFetcher fetcher.Fetcher
			found       = false
		)

		// retrieve the current context or the value from "-c"
		_, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// validate the nodeID
		nodeIDArray, err := GetClusterNodeIDs(dataFetcher)
		if err != nil {
			return err
		}
		for _, v := range nodeIDArray {
			i, _ := strconv.Atoi(v)
			if i == reporterNodID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unable to find node id %v", reporterNodID)
		}

		jsonData, err = dataFetcher.RunReportJSON(args[0], reporterNodID)
		if err != nil {
			return err
		}

		// output format cannot be table
		if OutputFormat == constants.TABLE {
			OutputFormat = constants.JSON
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			jsonPathResult, err := utils.GetJSONPathResults(jsonData, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(jsonPathResult)
			return nil
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(jsonData))
		}

		return nil
	},
}

// startReporterCmd represents the start reporter command.
var startReporterCmd = &cobra.Command{
	Use:               reporterUse,
	Short:             "start a reporter on a node",
	Long:              `The 'start reporter' command starts the Coherence reporter on the specified node.`,
	ValidArgsFunction: completionNodeID,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideNodeID)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueReporterCommand(args[0], "start", cmd)
	},
}

// stopReporterCmd represents the stop reporter command.
var stopReporterCmd = &cobra.Command{
	Use:               reporterUse,
	Short:             "stop a reporter on a node",
	Long:              `The 'stop reporter' command stops the Coherence reporter on the specified node.`,
	ValidArgsFunction: completionNodeID,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideNodeID)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueReporterCommand(args[0], "stop", cmd)
	},
}

// setReporterCmd represents the set reporter command.
var setReporterCmd = &cobra.Command{
	Use:   "reporter {node-ids|all}",
	Short: "set a reporter attribute for one or more members",
	Long: `The 'set reporter' command sets an attribute for one or more reporter nodes.
You can specify 'all' to change the value for all nodes, or specify a comma separated
list of node ids. The following attribute names are allowed:
configFile, currentBatch, intervalSeconds or outputPath.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a comma separated list of node id's or 'all' for all nodes")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher     fetcher.Fetcher
			connection      string
			err             error
			nodeIDs         []string
			nodeIDArray     []string
			confirmMessage  string
			errorSink       = createErrorSink()
			wg              sync.WaitGroup
			reporterNodeIDs = args[0]
			actualValue     interface{}
			intValue        int
		)

		if !utils.SliceContains(validReporterAttributes, reporterAttributeName) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of\n%v",
				reporterAttributeName, validReporterAttributes)
		}

		if reporterAttributeName == reporterConfigFile || reporterAttributeName == reporterOutputPath {
			actualValue = reporterAttributeValue
		} else {
			// convert to an int
			intValue, err = strconv.Atoi(reporterAttributeValue)
			if err != nil {
				return fmt.Errorf("invalid integer value of %s for attribute %s", reporterAttributeValue, reporterAttributeName)
			}

			actualValue = intValue
			// carry out some basic validation
			if intValue <= 0 {
				return fmt.Errorf("value for attribute %s must be greater than zero", reporterAttributeName)
			}

		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		nodeIDArray, err = GetClusterNodeIDs(dataFetcher)
		if err != nil {
			return err
		}

		if reporterNodeIDs == all {
			nodeIDs = append(nodeIDs, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d reporter nodes", len(nodeIDs))
		} else {
			nodeIDs = strings.Split(reporterNodeIDs, ",")
			for _, value := range nodeIDs {
				if !utils.IsValidInt(value) {
					return fmt.Errorf("invalid value for reporter node id of %s", value)
				}

				if !utils.SliceContains(nodeIDArray, value) {
					return fmt.Errorf("no node with node id %s exists in this cluster", value)
				}
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIDs))
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
			reporterAttributeName, reporterAttributeValue, confirmMessage)) {
			return nil
		}

		nodeCount := len(nodeIDs)
		wg.Add(nodeCount)

		for _, value := range nodeIDs {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.SetReporterAttribute(nodeId, reporterAttributeName, actualValue)
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

func init() {
	setReporterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	startReporterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopReporterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setReporterCmd.Flags().StringVarP(&reporterAttributeName, "attribute", "a", "", "attribute name to set")
	_ = setReporterCmd.MarkFlagRequired("attribute")
	setReporterCmd.Flags().StringVarP(&reporterAttributeValue, "value", "v", "", "attribute value to set")
	_ = setReporterCmd.MarkFlagRequired("value")

	runReportCmd.Flags().IntVarP(&reporterNodID, "node", "n", 0, "node to run report on")
	_ = runReportCmd.MarkFlagRequired("node")
}
