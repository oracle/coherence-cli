/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// restartCmd represents the restart command.
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart a resource",
	Long:  `The 'restart' command restarts various resources.`,
}
