/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
)

const httpString = "http"

// getHTTPProxiesCmd represents the get http-servers command.
var getHTTPProxiesCmd = &cobra.Command{
	Use:   "http-servers",
	Short: "display http proxy services for a cluster",
	Long:  `The 'get http-servers' command displays the list of http proxy servers for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			connection  string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		details, err := returnGetProxiesDetails(cmd, httpString, dataFetcher, connection)
		if err != nil {
			return err
		}
		if !isWatchEnabled() {
			// don't display the value if watch was enabled as it was already output
			cmd.Println(details)
		}
		return nil
	},
}

// describeHTTPProxyCmd represents the describe http-proxy command.
var describeHTTPProxyCmd = &cobra.Command{
	Use:               "http-server service-name",
	Short:             "describe a http server",
	Long:              `The 'describe http-server' command shows information related to http servers.`,
	ValidArgsFunction: completionHTTPServers,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
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
			if value.ServiceName == serviceName && value.Protocol == httpString {
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

		err = displayProxyDetails(cmd, dataFetcher, connection, httpString, serviceResult, proxyResults)
		if err != nil {
			return err
		}

		return nil
	},
}
