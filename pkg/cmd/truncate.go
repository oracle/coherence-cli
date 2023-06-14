/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// truncateCmd represents the truncate command.
var truncateCmd = &cobra.Command{
	Use:   "truncate",
	Short: "truncates resources",
	Long:  `The 'truncate' command truncates resources.`,
}
