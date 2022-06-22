/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
)

// getEnvironmentCmd represents the get environment command
var getEnvironmentCmd = &cobra.Command{
	Use:   "environment node-id",
	Short: "display the member environment",
	Long: `The 'get environment' command returns the environment information for a member.
This includes details of the JVM as well as system properties.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a node id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		type environment struct {
			Environment string `json:"environment"`
		}

		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
			response    []byte
			nodeIDArray []string
			nodeID      = args[0]
			env         environment
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

		response, err = dataFetcher.GetEnvironment(nodeID)
		if err != nil {
			return err
		}

		if len(response) > 0 {
			err = json.Unmarshal(response, &env)
			if err != nil {
				return err
			}
			cmd.Println(env.Environment)
		}

		return nil
	},
}
