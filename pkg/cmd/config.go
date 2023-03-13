/*
 * Copyright (c) 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
)

// describeConfigCmd represents the describe comfig command
var describeConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "describe the config in .cohctl.yaml",
	Long: `The 'describe config' command describes the config stored in the '.cohctl.yaml' config file
in a human readable format.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
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
	describeConfigCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false,
		"include verbose output including cluster connections and profiles")
}
