/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// forceCmd represents the force command.
var forceCmd = &cobra.Command{
	Use:   "force",
	Short: "force recovery for a persistence service",
	Long:  `The 'force' command forces recovery.`,
}
