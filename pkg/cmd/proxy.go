/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
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
	"time"
)

// var getProxiesCmd = &cobra.Command{ represents the getProxies command
var getProxiesCmd = &cobra.Command{
	Use:   "proxies",
	Short: "display Coherence*Extend proxy services for a cluster",
	Long: `The 'get proxies' command displays the list of Coherence*Extend proxy
servers for a cluster. You can specify '-o wide' to display addition information.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
		)

		_, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		details, err := returnGetProxiesDetails(cmd, "tcp", dataFetcher)
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

// describeProxyCmd represents the describe proxy command
var describeProxyCmd = &cobra.Command{
	Use:   "proxy service-name",
	Short: "describe a proxy server",
	Long: `The 'describe proxy' command shows information related to proxy servers including
all nodes running the proxy service as well as detailed connection information.`,
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

		// retrieve the current context or the value from "-c"
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
			return err
		}

		// get a list of node Id's while we search for the service name
		nodeIds := make([]string, 0)
		proxyCount := 0
		for _, value := range proxiesSummary.Proxies {
			if value.ServiceName == serviceName && value.Protocol == "tcp" {
				proxyCount++
				nodeIds = append(nodeIds, value.NodeID)
			}
		}
		if proxyCount == 0 {
			return fmt.Errorf("unable to find Proxy service with service name '%s'", serviceName)
		}

		// we have valid service name so issue queries
		serviceResult, err = dataFetcher.GetSingleServiceDetailsJSON(serviceName)
		if err != nil {
			return err
		}

		// retrieve all connection details from JSON
		connectionDetails := make([]config.GenericDetails, len(nodeIds))

		for i := range nodeIds {
			connectionDetails[i] = config.GenericDetails{}
			data, err := dataFetcher.GetProxyConnectionsJSON(serviceName, nodeIds[i])
			if err != nil {
				return err
			}
			err = json.Unmarshal(data, &connectionDetails[i])
			if err != nil {
				return err
			}
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
		cmd.Print("\nPROXY SERVICE DETAILS\n")
		cmd.Print("---------------------\n")
		value, err := FormatJSONForDescribe(serviceResult, true, "Name", "Type")
		if err != nil {
			return err
		}

		cmd.Println(value)

		cmd.Print("PROXY MEMBER DETAILS\n")
		cmd.Print("--------------------\n")
		value, err = returnGetProxiesDetails(cmd, "tcp", dataFetcher)
		if err != nil {
			return err
		}

		cmd.Println(value)

		cmd.Print("PROXY CONNECTIONS\n")
		cmd.Print("-----------------\n")

		for _, value := range connectionDetails {
			for _, detail := range value.Details {
				// deserialize the map into json for the format function
				jsonData, err := json.Marshal(detail)
				if err != nil {
					return err
				}

				value, err := FormatJSONForDescribe(jsonData, true, "Node Id", "Remote Address", "Remote Port")
				if err != nil {
					return err
				}
				cmd.Println(value)
			}
		}

		return nil
	},
}

func returnGetProxiesDetails(cmd *cobra.Command, protocol string, dataFetcher fetcher.Fetcher) (string, error) {
	var proxiesSummary = config.ProxiesSummary{}
	var sb strings.Builder

	for {
		if watchEnabled {
			sb.WriteString("\n" + time.Now().String() + "\n")
		}

		proxyResults, err := dataFetcher.GetProxySummaryJSON()
		if err != nil {
			return "", err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err := utils.GetJSONPathResults(proxyResults, OutputFormat)
			if err != nil {
				return "", err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			sb.WriteString(string(proxyResults))
		} else {
			err = json.Unmarshal(proxyResults, &proxiesSummary)
			if err != nil {
				return "", utils.GetError("unable to unmarshall proxy result", err)
			}
			sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, protocol))
		}

		// check to see if we should exit if we are not watching
		if !watchEnabled {
			break
		}

		// if we are watching then output the details directly
		cmd.Println(sb.String())

		// we are watching so sleep and then repeat until CTRL-C
		time.Sleep(time.Duration(watchDelay) * time.Second)
	}

	return sb.String(), nil
}
