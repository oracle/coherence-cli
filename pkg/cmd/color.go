/*
 * Copyright (c) 2023 Oracle and/or its affiliates.
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
	setColorMsg   = "Color formatting is now set to "
	setColorError = "you can only specify 'on' or 'off'"
	getColorMsg   = "Color formatting is: "
)

// setColorCmd represents the set color command.
var setColorCmd = &cobra.Command{
	Use:   "color {on|off}}",
	Short: "set color formatting to be on or off",
	Long: `The 'set color' command sets color formatting to on or off. If 'on' then formatting
of output when using a terminal highlights columns requiring attention.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, setError)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var color string
		if value == on {
			color = on
		} else if value == off {
			color = off
		} else {
			return errors.New(setColorError)
		}

		viper.Set(colorContextKey, color)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setColorMsg + value)
		return nil
	},
}

// getColorCmd represents the get color command.
var getColorCmd = &cobra.Command{
	Use:   "color",
	Short: "display the current color formatting setting",
	Long: `The 'get color' command displays the current color formatting setting. If 'on' then formatting
of output when using a terminal highlights columns requiring attention.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var color string
		if Config.Color == "" || Config.Color == on {
			color = on
		} else {
			color = off
		}
		cmd.Printf("%s%v\n", getColorMsg, color)
		return nil
	},
}
