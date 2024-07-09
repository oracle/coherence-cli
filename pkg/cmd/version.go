/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
	"runtime"
	"strings"
)

var checkForUpdates bool

const (
	stableURL = "https://oracle.github.io/coherence-cli/stable.txt"
	updateURL = "https://github.com/oracle/coherence-cli/releases"
)

// versionCmd represents the version command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show version information",
	Long: `The 'get version' command displays version and build details for the Coherence-CLI.
Use the '-u' option to check for updates.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, _ []string) {
		var (
			err           error
			latestVersion string
		)

		cmd.Printf("Coherence Command Line Interface\nCLI Version:  %s\nDate:         %s\n"+
			"Commit:       %s\nOS:           %s\nArchitecture: %s\nGo Version:   %s\n",
			Version, Date, Commit, runtime.GOOS, runtime.GOARCH, runtime.Version())

		if checkForUpdates {
			cmd.Println("\nChecking for updates...")
			latestVersion, err = getLatestVersion()
			if err != nil {
				cmd.Printf("Error: unable to check for updates: %s\n", err.Error())
				cmd.Println("If you are behind a Proxy Server then set the HTTP_PROXY environment variable")
				cmd.Println("E.g. HTTP_PROXY=http://proxy-host:proxy-port/")
			} else {
				latestVersion = strings.ReplaceAll(latestVersion, "\n", "")
				// we now have a value for the latest version
				if isVersionUpdateAvailable(Version, latestVersion) {
					cmd.Printf("A newer version of cohctl (%s) is available.\nPlease visit the following URL to update:\n%s\n",
						latestVersion, updateURL)
				} else {
					cmd.Println("You are on the latest version")
				}
			}
		}
	},
}

// isVersionUpdate returns true if there is a new stable version available.
func isVersionUpdateAvailable(haveVersion, stableVersion string) bool {
	return haveVersion != stableVersion
}

func init() {
	versionCmd.Flags().BoolVarP(&checkForUpdates, "check-updates", "u", false,
		"if true, will check for updates")
}

func getLatestVersion() (string, error) {
	response, err := GetURLContents(stableURL)
	if err != nil {
		return "", err
	}

	return string(response), nil
}
