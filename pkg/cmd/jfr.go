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
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

var (
	NodeID          string
	duration        int32
	jfrRoleName     string
	jfrDumpFileName string
)

const (
	jfrNameUse = "jfr name"
	supplyJFR  = "you must provide a JFR name"
)

// getJfrsCmd represents the get jfrs command.
var getJfrsCmd = &cobra.Command{
	Use:   "jfrs",
	Short: "display Java Flight Recordings for a cluster",
	Long:  `The 'get jfrs' command displays the Java Flight Recordings for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		return executeJFROperation(cmd, "", fetcher.GetJFRs, dataFetcher, "")
	},
}

// describeJfrCmd represents the describe jfr command.
var describeJfrCmd = &cobra.Command{
	Use:   jfrNameUse,
	Short: "describe a Java Flight Recording (JFR)",
	Long:  `The 'describe jfr' command shows information related to a Java Flight Recording (JFR).`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyJFR)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		return executeJFROperation(cmd, args[0], fetcher.CheckJFR, dataFetcher, "")
	},
}

// startJfrCmd represents the start jfr command.
var startJfrCmd = &cobra.Command{
	Use:   jfrNameUse,
	Short: "start a Java Flight Recording (JFR) for all or selected members",
	Long: `The 'start jfr' command starts a Java Flight Recording all or selected members.
You can specify either a node id or role. If you do not specify either, then the JFR will 
be run for all members. The default duration is 60 seconds and you can specify a value
of 0 to make the recording continuous.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyJFR)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
			nodeIds     []string
			jfrName     = args[0]
			jfrType     string
			jfrMessage  string
			target      = ""
			data        []byte
			finalResult string
		)

		if duration < 0 {
			return errors.New("duration must be greater or equal to zero")
		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		// check for node id
		if NodeID != "" {
			jfrType = fetcher.JfrTypeNode
			if !utils.IsValidInt(NodeID) {
				return fmt.Errorf("invalid node id %s", NodeID)
			}

			nodeIds, err = GetNodeIds(dataFetcher)
			if err != nil {
				return err
			}

			if !utils.SliceContains(nodeIds, NodeID) {
				return fmt.Errorf("node id %s does not exist on this cluster", NodeID)
			}
			jfrMessage = "node id " + NodeID
			target = NodeID
		} else if jfrRoleName != all {
			jfrType = fetcher.JfrTypeRole
			jfrMessage = "role " + jfrRoleName
			target = jfrRoleName
		} else {
			// must be cluster wide
			jfrType = fetcher.JfrTypeCluster
			nodeIds, err = GetNodeIds(dataFetcher)
			if err != nil {
				return err
			}
			jfrMessage = fmt.Sprintf("all %d members", len(nodeIds))
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to start a JFR named %s for %s of duration: %d seconds? (y/n) ",
			jfrName, jfrMessage, duration)) {
			return nil
		}

		data, err = dataFetcher.StartJFR(jfrName, outputDirectory, jfrType, target, duration)
		if err != nil {
			return err
		}

		finalResult, err = getJFResults(jfrType, data)
		if err != nil {
			return err
		}
		cmd.Println(finalResult)

		return nil
	},
}

// stopJfrCmd represents the start jfr command.
var stopJfrCmd = &cobra.Command{
	Use:   jfrNameUse,
	Short: "stop a Java Flight Recording (JFR) for all or selected members",
	Long: `The 'stop jfr' command stops a Java Flight Recording all or selected members.
You can specify either a node or leave the node blank to stop for all nodes.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyJFR)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		return executeJFROperation(cmd, args[0], fetcher.StopJFR, dataFetcher, "")
	},
}

// dumpJfrCmd represents the dump jfr command.
var dumpJfrCmd = &cobra.Command{
	Use:   jfrNameUse,
	Short: "dump a Java Flight Recording (JFR) for all or selected members",
	Long: `The 'dump jfr' command dumps a Java Flight Recording all or selected members.
A JFR command mut be in progress for this to succeed.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyJFR)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
		)
		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		return executeJFROperation(cmd, args[0], fetcher.DumpJFR, dataFetcher, jfrDumpFileName)
	},
}

// executeJFROperation executes a jfrStop, jfrDump, jfrCheck or  command.
func executeJFROperation(cmd *cobra.Command, jfrName, operation string, dataFetcher fetcher.Fetcher, filename string) error {
	var (
		err         error
		NodeIds     []string
		jfrType     string
		jfrMessage  string
		target      = ""
		data        []byte
		finalResult string
	)

	// check for node id
	if NodeID != "" {
		jfrType = fetcher.JfrTypeNode
		if !utils.IsValidInt(NodeID) {
			return fmt.Errorf("invalid node id %s", NodeID)
		}

		NodeIds, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if !utils.SliceContains(NodeIds, NodeID) {
			return fmt.Errorf("node id %s does not exist on this cluster", NodeID)
		}
		jfrMessage = "node id " + NodeID
		target = NodeID
	} else {
		// must be cluster wide
		jfrType = fetcher.JfrTypeCluster
		NodeIds, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}
		jfrMessage = fmt.Sprintf("all %d members", len(NodeIds))
	}

	if operation != fetcher.CheckJFR && !automaticallyConfirm {
		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to run %s on a JFR named %s for %s ? (y/n) ", operation, jfrName, jfrMessage)) {
			return nil
		}
	}

	// ensure watch enabled cannot be set for anything other than fetcher.GetJFRs
	if operation != fetcher.GetJFRs && isWatchEnabled() {
		watchEnabled = false
	}

	for {
		if operation == fetcher.StopJFR {
			data, err = dataFetcher.StopJFR(jfrName, jfrType, target)
		} else if operation == fetcher.DumpJFR {
			data, err = dataFetcher.DumpJFR(jfrName, jfrType, target, filename)
		} else if operation == fetcher.CheckJFR {
			data, err = dataFetcher.CheckJFR(jfrName, jfrType, target)
		} else if operation == fetcher.GetJFRs {
			data, err = dataFetcher.CheckJFR("", jfrType, "")
		}
		if err != nil {
			return err
		}

		finalResult, err = getJFResults(jfrType, data)
		if err != nil {
			return err
		}

		// format the result
		printWatchHeader(cmd)
		cmd.Println(formatJFROutput(finalResult))

		// check to see if we should exit if we are not watching
		if !isWatchEnabled() {
			break
		}
		// we are watching so sleep and then repeat until CTRL-C
		time.Sleep(time.Duration(watchDelay) * time.Second)
	}

	return nil
}

// getJFResults returns the JFR result.
func getJFResults(jfrType string, data []byte) (string, error) {
	var (
		status       config.StatusValues
		singleStatus config.SingleStatusValue
		err          error
	)
	if jfrType == fetcher.JfrTypeNode {
		// single value
		err = json.Unmarshal(data, &singleStatus)
		if err != nil {
			return "", err
		}
		return singleStatus.Status, nil
	}
	// multiple values
	err = json.Unmarshal(data, &status)
	if err != nil {
		return "", err
	}
	finalResult := ""
	for _, value := range status.Status {
		finalResult += value
	}

	return formatJFROutput(finalResult), nil
}

// formatJFROutput formats the output of a JFR result.
func formatJFROutput(result string) string {
	var (
		finalResult = strings.Replace(strings.Replace(result, "\n\n", "\n", -1), "->", "->\n", -1)
		sb          strings.Builder
	)

	for _, line := range strings.Split(finalResult, "\n") {
		if line == "" {
			continue
		}
		if !strings.Contains(line, "->") {
			sb.WriteString("  ")
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func init() {
	nodeIDDesc := "node id to target"

	describeJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)

	startJfrCmd.Flags().StringVarP(&outputDirectory, "output-dir", "O", "", "directory on servers to output JFR's to")
	_ = startJfrCmd.MarkFlagRequired("output-dir")
	startJfrCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	startJfrCmd.Flags().StringVarP(&jfrRoleName, "role", "r", all, "role name to target")
	startJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)
	startJfrCmd.Flags().Int32VarP(&duration, "duration", "D", 60, "duration for JFR in seconds. Use 0 for continuous")

	stopJfrCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)

	dumpJfrCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	dumpJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)
	dumpJfrCmd.Flags().StringVarP(&jfrDumpFileName, "filename", "f", "", "filename for jfr dump")
	_ = dumpJfrCmd.MarkFlagRequired("filename")
}
