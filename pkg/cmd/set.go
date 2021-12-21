/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

// setCommand represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration value",
	Long:  `The 'set' command sets the current context, debug, timeout value or to ignore invalid SSL certificates.`,
}
