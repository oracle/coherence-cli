/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package main

import (
	"fmt"
	"github.com/oracle/coherence-cli/pkg/cmd"
	"os"
)

var (
	// Version is the cohctl version injected by the Go linker at build time
	Version string
	// Commit is the git commit hash injected by the Go linker at build time
	Commit string
	// Date is the build timestamp injected by the Go linker at build time
	Date string
)

// main is the main entry point to Coherence CLI
func main() {
	if os.Getenv("CLI_DISABLED") != "" {
		fmt.Println("cohctl has been disabled from running in the Coherence Operator")
		return
	}
	cmd.Execute(Version, Date, Commit)
}
