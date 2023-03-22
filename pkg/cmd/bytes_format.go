/*
 * Copyright (c) 2022, 2023 Oracle and/or its affiliates.
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
	setDefaultBytesMsg   = "Bytes format is now set to "
	getDefaultBytesMsg   = "Current bytes format: "
	clearDefaultBytesMsg = "Default bytes format has been cleared"
	invalidBytesValue    = "you must provide either 'k', 'm', 'g' or 't'"
	bytesFormatK         = "k"
	bytesFormatM         = "m"
	bytesFormatG         = "g"
	bytesFormatT         = "t"
)

// setBytesFormatCmd represents the set bytes-format command.
var setBytesFormatCmd = &cobra.Command{
	Use:   "bytes-format {k|m|g|t}",
	Short: "set default bytes format for displaying memory or disk based sizes",
	Long: `The 'set bytes-format' command sets the default format for displaying memory or disk based sizes.
Valid values are k - kilobytes, m - megabytes, g - gigabytes or t - terabytes. If not specified the default will be b - bytes.
The default value will be overridden if you specify the -k, -m, -g or --tb options.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, invalidBytesValue)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		if value != bytesFormatK && value != bytesFormatM && value != bytesFormatG && value != bytesFormatT {
			return errors.New(invalidBytesValue)
		}

		viper.Set(defaultBytesFormatKey, value)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setDefaultBytesMsg + value)
		return nil
	},
}

// getBytesFormatCmd represents the get bytes-format command.
var getBytesFormatCmd = &cobra.Command{
	Use:   "bytes-format",
	Short: "display the current format for displaying memory or disk based sizes",
	Long:  `The 'get bytes-format' displays the current format for displaying memory or disk based sizes.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Printf("%s%v\n", getDefaultBytesMsg, Config.DefaultBytesFormat)
		return nil
	},
}

// clearBytesFormatCmd represents the clear bytes-format command.
var clearBytesFormatCmd = &cobra.Command{
	Use:   "bytes-format",
	Short: "clear the current format for displaying memory or disk based sizes",
	Long:  `The 'clear bytes-format' clears the current format for displaying memory or disk based sizes.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.Set(defaultBytesFormatKey, "")
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(clearDefaultBytesMsg)
		return nil
	},
}
