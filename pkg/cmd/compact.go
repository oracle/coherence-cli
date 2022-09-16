/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// compactCmd represents the compact command
var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "compact an elastic-data resource",
	Long:  `The 'compact' command compacts elastic-data  resources.`,
}
