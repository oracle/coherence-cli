/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// recoverCmd represents the recover command.
var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "recover a resource",
	Long:  `The 'recover' command recovers a resource.`,
}
