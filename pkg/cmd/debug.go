/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const setDebugMsg = "Debug is now set to "
const setDebugError = "you can only specify 'on' or 'off'"
const getDebugMsg = "Current debug level: "

// setDebugCmd represents the set debug command
var setDebugCmd = &cobra.Command{
	Use:   "debug {on|off}}",
	Short: "set debug messages to be on or off",
	Long: `The 'set debug' command sets debug to on or off. If 'on' then additional
information is logged in the log file (cohctl.log).`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide either on or off")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var debug bool
		if value == "on" {
			debug = true
		} else if value == "off" {
			debug = false
		} else {
			return errors.New(setDebugError)
		}

		viper.Set(debugContextKey, debug)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setDebugMsg + value)
		return nil
	},
}

// getDebugCmd represents the get debug command
var getDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "display the current debug level",
	Long: `The 'get debug' command displays the current debug level. If 'on' then 
additional information is logged in the log file.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var debug string
		if Config.Debug {
			debug = "on"
		} else {
			debug = "off"
		}
		cmd.Printf("%s%v\n", getDebugMsg, debug)
		return nil
	},
}
