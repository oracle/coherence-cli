/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
)

// var getHTTPProxiesCmd represents the getHTTPProxiesCmd command
var getHTTPProxiesCmd = &cobra.Command{
	Use:   "http-servers",
	Short: "Display http proxy services for a cluster",
	Long:  `The 'get http-servers' command displays the list of http proxy servers for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
		)
		_, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		details, err := returnGetProxiesDetails(cmd, "http", dataFetcher)
		if err != nil {
			return err
		}
		if !watchEnabled {
			// don't display the value if watch was enabled as it was already output
			cmd.Println(details)
		}
		return nil
	},
}

// describeHTTPProxyCmd represents the describe http-proxy command
var describeHTTPProxyCmd = &cobra.Command{
	Use:   "http-server service-name",
	Short: "Describe a http server",
	Long:  `The 'describe http-server' command shows information related to http servers.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			serviceResult  []byte
			proxiesSummary = config.ProxiesSummary{}
			err            error
			dataFetcher    fetcher.Fetcher
			connection     string
		)
		serviceName := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}
		proxyResults, err := dataFetcher.GetProxySummaryJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(proxyResults, &proxiesSummary)
		if err != nil {
			return utils.GetError("unable to decode proxy details", err)
		}

		found := false
		for _, value := range proxiesSummary.Proxies {
			if value.ServiceName == serviceName && value.Protocol == "http" {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unable to find Http Server with service name '%s'", serviceName)
		}

		// we have valid service name so issue queries
		serviceResult, err = dataFetcher.GetSingleServiceDetailsJSON(serviceName)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
			finalResult, err := utils.CombineByteArraysForJSON([][]byte{serviceResult, proxyResults}, []string{"services", "members"})
			if err != nil {
				return err
			}
			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(finalResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
				return nil
			}
			cmd.Println(string(finalResult))
			return nil
		}
		cmd.Print(FormatCurrentCluster(connection))
		cmd.Print("\nHTTP SERVER SERVICE DETAILS\n")
		cmd.Print("-------------------------------\n")
		value, err := FormatJSONForDescribe(serviceResult, true, "Name", "Type")
		if err != nil {
			return err
		}

		cmd.Println(value)

		cmd.Print("HTTP SERVER MEMBER DETAILS\n")
		cmd.Print("--------------------------\n")
		value, err = returnGetProxiesDetails(cmd, "http", dataFetcher)
		if err != nil {
			return err
		}

		cmd.Println(value)

		return nil
	},
}
