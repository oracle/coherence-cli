/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// createCmd represents the clear command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource",
	Long:  `The 'create' command creates various resources.`,
}
