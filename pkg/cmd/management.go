/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

const (
	expiryDelay   = "expiryDelay"
	refreshPolicy = "refreshPolicy"
)

var (
	attributeNameMgmt    string
	attributeValueMgmt   string
	validAttributesMgmt  = []string{expiryDelay, refreshPolicy}
	validRefreshPolicies = []string{"refresh-ahead", "refresh-behind", "refresh-expired", "refresh-onquery"}
)

// getManagementCmd represents the get management command
var getManagementCmd = &cobra.Command{
	Use:   "management",
	Short: "display management information for a cluster",
	Long:  `The 'get management' command displays the management information for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			jsonData    []byte
			connection  string
			err         error
			value       string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			printWatchHeader(cmd)

			jsonData, err = dataFetcher.GetManagementJSON()
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(jsonData, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(jsonData))
			} else {
				cmd.Println(FormatCurrentCluster(connection))
				value, err = FormatJSONForDescribe(jsonData, true,
					"Refresh Policy", "Expiry Delay")
				if err != nil {
					return err
				}

				cmd.Println(value)
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching services so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// setManagementCmd represents the set member command
var setManagementCmd = &cobra.Command{
	Use:   "management",
	Short: "set a management attribute for the cluster",
	Long: `The 'set management' command sets a management attribute for the cluster.
The following attribute names are allowed: expiryDelay and refreshPolicy.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			connection  string
			err         error
			actualValue interface{}
			intValue    int
		)

		if !utils.SliceContains(validAttributesMgmt, attributeNameMgmt) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of %v",
				attributeNameMgmt, validAttributesMgmt)
		}

		if attributeNameMgmt == refreshPolicy {
			// this is the only attribute that is a string
			if !utils.SliceContains(validRefreshPolicies, attributeValueMgmt) {
				return fmt.Errorf("attribute value for %s must be one of %v", refreshPolicy, validRefreshPolicies)
			}
			actualValue = attributeValueMgmt
		} else {
			// convert to an int
			intValue, err = strconv.Atoi(attributeValueMgmt)
			if err != nil {
				return fmt.Errorf("invalid integer value of %s for attribute %s", attributeValueMgmt, attributeNameMgmt)
			}

			actualValue = intValue
			// carry out some basic validation
			if attributeNameMgmt == expiryDelay && intValue <= 0 {
				return fmt.Errorf("value for attribute %s must be greater than zero", attributeNameMgmt)
			}
		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the value of attribute %s to %s? (y/n) ",
			attributeNameMgmt, attributeValueMgmt)) {
			return nil
		}

		_, err = dataFetcher.SetManagementAttribute(attributeNameMgmt, actualValue)
		if err != nil {
			return err
		}
		cmd.Println("operation completed")

		return nil
	},
}

func init() {
	setManagementCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setManagementCmd.Flags().StringVarP(&attributeNameMgmt, "attribute", "a", "", "attribute name to set")
	_ = setManagementCmd.MarkFlagRequired("attribute")
	setManagementCmd.Flags().StringVarP(&attributeValueMgmt, "value", "v", "", "attribute value to set")
	_ = setManagementCmd.MarkFlagRequired("value")
}
