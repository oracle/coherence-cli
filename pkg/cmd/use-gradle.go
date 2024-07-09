/*
 * Copyright (c) 2022, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	setUseGradleMsg   = "Use Gradle is now set to "
	setSetGradleError = "you can only specify 'true' or 'false'"
	getUseGradleMsg   = "Use Gradle: "
)

// setUseGradleCmd represents the set use-gradle command.
var setUseGradleCmd = &cobra.Command{
	Use:   "use-gradle {true|false}",
	Short: "set whether to use gradle for dependency management",
	Long: `The 'set use-gradle' command sets whether to use gradle for dependency management.
This setting only affects when you create a cluster. If set to false, the default of Maven will be used.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide either true or false")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var useGradle bool
		if value == stringTrue {
			useGradle = true
		} else if value == stringFalse {
			useGradle = false
		} else {
			return errors.New(setSetGradleError)
		}

		viper.Set(useGradleContextKey, useGradle)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setUseGradleMsg + value)
		return nil
	},
}

// getUseGradleCmd represents the get use-gradle.
var getUseGradleCmd = &cobra.Command{
	Use:   "use-gradle",
	Short: "display the current setting for using gradle for dependency management",
	Long: `The 'get use-gradle' command displays the current setting for whether to 
use gradle for dependency management. If set to false, the default of Maven is used.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Printf("%s%v\n", getUseGradleMsg, Config.UseGradle)
		return nil
	},
}
