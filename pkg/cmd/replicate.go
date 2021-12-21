/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// replicateCmd represents the replicate command
var replicateCmd = &cobra.Command{
	Use:   "replicate",
	Short: "Replicate a federated service",
	Long:  `The 'replicate' command starts replicate all for federated services.`,
}
