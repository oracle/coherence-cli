/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// startCmd represents the start command.
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start a resource",
	Long:  `The 'start' command starts various resources.`,
}
