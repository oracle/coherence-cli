/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
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
	setIgnoreCertsMsg   = "Value is now set to "
	setIgnoreCertsError = "you can only specify 'true' or 'false'"
)

var (
	getIgnoreCertsMsg = "Current setting: "
)

// getIgnoreCertsCmd represents the get ignore-certs.
var getIgnoreCertsCmd = &cobra.Command{
	Use:   "ignore-certs",
	Short: "display the current setting for ignoring invalid SSL Certificates",
	Long: `The 'get ignore-certs' command displays the current setting for ignoring 
invalid SSL Certificates. If 'true' then invalid certificates such as self signed will be allowed. 
You should only use this option when you are sure of the identify of the target server.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var value string
		if Config.IgnoreInvalidCerts {
			value = stringTrue
		} else {
			value = stringFalse
		}
		cmd.Printf("%s%v\n", getIgnoreCertsMsg, value)
		return nil
	},
}

// setIgnoreInvalidCertsCmd represents the set ignore-certs command.
var setIgnoreCertsCmd = &cobra.Command{
	Use:   "ignore-certs {true|false}",
	Short: "set current setting for ignoring invalid SSL Certificates",
	Long: `The 'set ignore-certs' set the current setting for ignoring
invalid SSL Certificates. If 'true' then invalid certificates such as self signed will be allowed.
You should only use this option when you are sure of the identify of the target server.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide either true or false")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var setting bool
		if value == stringTrue {
			setting = true
		} else if value == stringFalse {
			setting = false
		} else {
			return errors.New(setIgnoreCertsError)
		}

		viper.Set(ignoreCertsContextKey, setting)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setIgnoreCertsMsg + value)
		return nil
	},
}
