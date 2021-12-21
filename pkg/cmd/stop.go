/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// stopCmd represents the start command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a resource",
	Long:  `The 'stop' command stops various resources.`,
}
