/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// clearCmd represents the clear command.
var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "clear a context",
	Long:  `The 'clear context' command clears the current context for running commands in.`,
}
