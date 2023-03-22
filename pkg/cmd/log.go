/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// logCmd represents the log command.
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "log a resource",
	Long:  `The 'logs' command logs various resources.`,
}
