/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strconv"
)

const (
	setTimeoutMsg     = "Timeout is now set to "
	getTimeoutMessage = "Current timeout: "
)

// setTimeoutCmd represents the set timeout command
var setTimeoutCmd = &cobra.Command{
	Use:   "timeout value",
	Short: "set request timeout",
	Long:  `The 'set timeout' command sets the request timeout (in seconds) for any HTTP requests.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a positive integer value for timeout in seconds")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid value for timeout of %s", args[0])
		}
		viper.Set(requestTimeoutKey, timeout)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Printf("%s%d\n", setTimeoutMsg, timeout)
		return nil
	},
}

// getTimeoutCmd represents the get timeout command
var getTimeoutCmd = &cobra.Command{
	Use:   "timeout",
	Short: "display the current request timeout",
	Long:  `The 'get timeout' command displays the current request timeout (in seconds) for any HTTP requests.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Printf("%s%v\n", getTimeoutMessage, Config.RequestTimeout)
		return nil
	},
}
