/*
 * Copyright (c) 2023, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
)

// getConfigCmd represents the describe config command.
var getConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "display the config in .cohctl.yaml",
	Long: `The 'get config' command displays the config stored in the '.cohctl.yaml' config file
in a human readable format.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		data, err := json.Marshal(Config)
		if err != nil {
			return err
		}
		cmd.Println("CONFIG")
		cmd.Println("------")

		value, err := FormatJSONForDescribe(data, false, "Version", "Current Context",
			"Debug", "Color", "Request Timeout", "Ignore Invalid Certs", "Default Bytes Format", "Default Heap", "Use Gradle")
		if err != nil {
			return err
		}
		cmd.Println(value)

		if verboseOutput {
			cmd.Println("CLUSTER CONNECTIONS")
			cmd.Println("-------------------")

			for _, con := range Config.Clusters {
				data, err := json.Marshal(con)
				if err != nil {
					return err
				}
				value, err := FormatJSONForDescribe(data, true, "Name")
				if err != nil {
					return err
				}
				cmd.Println(value)
			}

			cmd.Println("PROFILES")
			cmd.Println("--------")
			cmd.Println(FormatProfiles(Config.Profiles))
		}
		return nil
	},
}

func init() {
	getConfigCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false,
		"include verbose output including cluster connections and profiles")
}
