/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// dumpCmd represents the dump command.
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "dump a resource",
	Long:  `The 'dump' command dumps various resources.`,
}
