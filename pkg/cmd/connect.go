/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// connectCmd represents the disconnect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect a resource",
	Long:  `The 'connect' command connects a resource.`,
}
