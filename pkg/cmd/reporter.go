/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
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
	"strings"
	"time"
)

// getReportersCmd represents the get reporters command
var getReportersCmd = &cobra.Command{
	Use:   "reporters",
	Short: "Display reporters for a cluster",
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
	Short: "Describe a reporter",
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
	Short: "Start a reporter on a node",
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
	Short: "Stops a reporter on a node",
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
