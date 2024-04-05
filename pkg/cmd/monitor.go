/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// getCmd represents the monitor command.
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "monitor one or many resources",
	Long:  `The 'monitor' command monitors one or more resources.`,
}
