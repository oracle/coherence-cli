/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// getLogsCmd represents the get logs command
var getLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "display the current 'cohctl' log file contents",
	Long:  `The 'get logs' command displays the current contents of the 'cohctl' log file.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(logFilePath)
		if err != nil {
			return fmt.Errorf("unable to display logfile %s: %v", logFilePath, err)
		}
		cmd.Println(string(data))
		return nil
	},
}
