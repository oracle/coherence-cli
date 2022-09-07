/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const setDefaultHeapMsg = "Default heap is now set to "
const getDefaultHeapMsg = "Current default heap: "
const clearDefaultHeapMsg = "Default heap has been cleared"
const invalidDefaultHeapValue = "you must provide a value"

// setDefaultHeapCmd represents the set default-heap command
var setDefaultHeapCmd = &cobra.Command{
	Use:   "default-heap value",
	Short: "set default heap for creating and starting clusters",
	Long: `The 'set default-heap' command sets the default heap when creating and starting cluster.
Valid values are in the format suitable for -Xms such as 256m, 1g, etc.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, invalidDefaultHeapValue)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]

		viper.Set(defaultHeapKey, value)
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setDefaultHeapMsg + value)
		return nil
	},
}

// getDefaultHeapCmd represents the get default-heap command
var getDefaultHeapCmd = &cobra.Command{
	Use:   "default-heap",
	Short: "display the default heap for creating and starting clusters",
	Long:  `The 'get default-heap' displays the default heap for creating and starting clusters.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Printf("%s%v\n", getDefaultHeapMsg, Config.DefaultHeap)
		return nil
	},
}

// clearDefaultHeapCmd represents the clear default-heap command
var clearDefaultHeapCmd = &cobra.Command{
	Use:   "default-heap",
	Short: "clear the current default heap for creating and starting clusters",
	Long:  `The 'clear default-heap' clears the default heap for creating and starting clusters.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.Set(defaultHeapKey, "")
		err := WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(clearDefaultHeapMsg)
		return nil
	},
}
