/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// archiveCmd represents the archive command.
var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "archive a resource",
	Long:  `The 'archive' command archive a resource.`,
}
