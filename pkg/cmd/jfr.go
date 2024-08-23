/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
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
	settingsFile    = "default"
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

		return executeJFROperation(cmd, "", fetcher.GetJFRs, dataFetcher, "", connection)
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

		return executeJFROperation(cmd, args[0], fetcher.CheckJFR, dataFetcher, "", connection)
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
			nodeIDs     []string
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

			nodeIDs, err = GetNodeIDs(dataFetcher)
			if err != nil {
				return err
			}

			if !utils.SliceContains(nodeIDs, NodeID) {
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
			nodeIDs, err = GetNodeIDs(dataFetcher)
			if err != nil {
				return err
			}
			jfrMessage = fmt.Sprintf("all %d members", len(nodeIDs))
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to start a JFR named %s for %s of duration: %d seconds using settings file %s? (y/n) ",
			jfrName, jfrMessage, duration, settingsFile)) {
			return nil
		}

		data, err = dataFetcher.StartJFR(jfrName, outputDirectory, jfrType, target, duration, settingsFile)
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

		return executeJFROperation(cmd, args[0], fetcher.StopJFR, dataFetcher, "", connection)
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

		return executeJFROperation(cmd, args[0], fetcher.DumpJFR, dataFetcher, jfrDumpFileName, connection)
	},
}

// executeJFROperation executes a jfrStop, jfrDump, jfrCheck or  command.
func executeJFROperation(cmd *cobra.Command, jfrName, operation string, dataFetcher fetcher.Fetcher, filename, connection string) error {
	var (
		err         error
		nodeIDs     []string
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

		nodeIDs, err = GetNodeIDs(dataFetcher)
		if err != nil {
			return err
		}

		if !utils.SliceContains(nodeIDs, NodeID) {
			return fmt.Errorf("node id %s does not exist on this cluster", NodeID)
		}
		jfrMessage = "node id " + NodeID
		target = NodeID
	} else {
		// must be cluster wide
		jfrType = fetcher.JfrTypeCluster
		nodeIDs, err = GetNodeIDs(dataFetcher)
		if err != nil {
			return err
		}
		jfrMessage = fmt.Sprintf("all %d members", len(nodeIDs))
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
		cmd.Println(FormatCurrentCluster(connection))
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
	startJfrCmd.Flags().StringVarP(&settingsFile, "settings-file", "s", "default", "settings file to use, options are \"profile\" or a specific file")
	startJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)
	startJfrCmd.Flags().Int32VarP(&duration, "duration", "D", 60, "duration for JFR in seconds. Use 0 for continuous")

	stopJfrCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)

	dumpJfrCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	dumpJfrCmd.Flags().StringVarP(&NodeID, "node", "n", "", nodeIDDesc)
	dumpJfrCmd.Flags().StringVarP(&jfrDumpFileName, "filename", "f", "", "filename for jfr dump")
	_ = dumpJfrCmd.MarkFlagRequired("filename")
}
