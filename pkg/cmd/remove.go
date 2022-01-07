/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// addCommand represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove a resource",
	Long:  `The 'remove' command removes a resource.`,
}
