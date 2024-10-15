/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// runCmd represents the run.
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run a report",
	Long:  `The 'run' command runs reports`,
}
