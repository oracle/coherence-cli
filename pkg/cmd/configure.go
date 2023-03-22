/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// configureCmd represents the configure command.
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "configure tracing",
	Long:  `The 'configure' command configures tracing.`,
}
