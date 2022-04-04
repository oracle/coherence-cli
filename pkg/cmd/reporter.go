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

const reporterConfigFile = "configFile"
const reporterCurrentBatch = "currentBatch"
const reporterIntervalSeconds = "intervalSeconds"
const reporterOutputPath = "outputPath"

var (
	validReporterAttributes = []string{reporterConfigFile, reporterCurrentBatch, reporterIntervalSeconds, reporterOutputPath}
	reporterAttributeName   string
	reporterAttributeValue  string
)

// getReportersCmd represents the get reporters command
var getReportersCmd = &cobra.Command{
	Use:   "reporters",
	Short: "display reporters for a cluster",
	Long:  `The 'get reporters' command displays the reporters for the cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			reporters   = config.Reporters{}
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			if watchEnabled {
				cmd.Println("\n" + time.Now().String())
			}

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
				cmd.Println(FormatCurrentCluster(connection))
				err = json.Unmarshal(reportersResult, &reporters)
				if err != nil {
					return utils.GetError("unable to unmarshall reporter result", err)
				}

				cmd.Print(FormatReporters(reporters.Reporters))
			}

			// check to see if we should exit if we are not watching
			if !watchEnabled {
				break
			}
			// we are watching services so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// describeReporterCmd represents the describe reporter command
var describeReporterCmd = &cobra.Command{
	Use:   "reporter node-id",
	Short: "describe a reporter",
	Long:  `The 'describe reporter' command shows information related to a particular reporter.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
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

// startReporterCmd represents the start reporter command
var startReporterCmd = &cobra.Command{
	Use:   "reporter node-id",
	Short: "start a reporter on a node",
	Long:  `The 'start reporter' command starts the Coherence reporter on the specified node.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueReporterCommand(args[0], "start", cmd)
	},
}

// stopReporterCmd represents the stop reporter command
var stopReporterCmd = &cobra.Command{
	Use:   "reporter node-id",
	Short: "stop a reporter on a node",
	Long:  `The 'stop reporter' command stops the Coherence reporter on the specified node.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueReporterCommand(args[0], "stop", cmd)
	},
}

// setReporterCmd represents the set reporter command
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
			nodeIds         []string
			nodeIDArray     []string
			confirmMessage  string
			response        string
			errorSink       = createErrorSink()
			wg              sync.WaitGroup
			reporterNodeIds = args[0]
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

		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if reporterNodeIds == "all" {
			nodeIds = append(nodeIds, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d reporter nodes", len(nodeIds))
		} else {
			nodeIds = strings.Split(reporterNodeIds, ",")
			for _, value := range nodeIds {
				if !utils.IsValidInt(value) {
					return fmt.Errorf("invalid value for reporter node id of %s", value)
				}

				if !utils.SliceContains(nodeIDArray, value) {
					return fmt.Errorf("no node with node id %s exists in this cluster", value)
				}
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIds))
		}

		if !automaticallyConfirm {
			cmd.Printf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
				reporterAttributeName, reporterAttributeValue, confirmMessage)
			_, err = fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
		}

		nodeCount := len(nodeIds)
		wg.Add(nodeCount)

		for _, value := range nodeIds {
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
		cmd.Println("operation completed")

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
}
