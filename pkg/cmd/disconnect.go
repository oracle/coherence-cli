/*
 * Copyright (c) 2022, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// disconnectCmd represents the disconnect command.
var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "disconnect a resource",
	Long:  `The 'disconnect' command disconnects a resource.`,
}
