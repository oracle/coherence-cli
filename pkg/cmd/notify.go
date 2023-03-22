/*
 * Copyright (c) 2022, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// notifyCmd represents the notify command.
var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "notify a resource",
	Long:  `The 'notify' command notifies a resource.`,
}
