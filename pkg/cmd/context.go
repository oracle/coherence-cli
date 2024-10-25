/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
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
	setContextMsg          = "Current context is now "
	UnableToFindClusterMsg = "unable to find cluster with connection name "
	getContextMsg          = "Current context: "
	clearContextMessage    = "Current context was cleared"
)

// setContextCmd represents the set context command.
var setContextCmd = &cobra.Command{
	Use:               "context connection-name",
	Short:             "set the current context",
	Long:              `The 'set context' command sets the current context or connection for running commands in.`,
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cluster := args[0]
		found, _ := GetClusterConnection(cluster)
		if !found {
			return errors.New(UnableToFindClusterMsg + cluster)
		}
		return setContext(cmd, args[0])
	},
}

// setContext sets the context.
func setContext(cmd *cobra.Command, cluster string) error {
	viper.Set(currentContextKey, cluster)
	err := WriteConfig()
	if err != nil {
		return err
	}
	cmd.Println(setContextMsg + cluster)
	return nil
}

// getContextCmd represents the get context command.
var getContextCmd = &cobra.Command{
	Use:   "context",
	Short: "display the current context",
	Long:  `The 'get context' command displays the current context.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Println(getContextMsg + Config.CurrentContext)
		return nil
	},
}

// clearContextCommand represents the clear context command.
var clearContextCmd = &cobra.Command{
	Use:   "context",
	Short: "clear the current context",
	Long:  `The 'clear context' command clears the current context for running commands in.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		return clearContext(cmd)
	},
}

// clearContext clears the current context.
func clearContext(cmd *cobra.Command) error {
	viper.Set(currentContextKey, "")
	if err := WriteConfig(); err != nil {
		return err
	}
	cmd.Println(clearContextMessage)
	return nil
}
