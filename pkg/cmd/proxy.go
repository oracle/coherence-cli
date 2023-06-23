/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
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
	"sync"
	"time"
)

const (
	provideProxyService = "you must provide a proxy service name"
	proxyErrorMsg       = "unable to find proxy service with service name"
	tcpString           = "tcp"
)

// var getProxiesCmd = &cobra.Command{ represents the getProxies command.
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
			connection  string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		details, err := returnGetProxiesDetails(cmd, tcpString, dataFetcher, connection)
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

// describeProxyCmd represents the describe proxy command.
var describeProxyCmd = &cobra.Command{
	Use:   "proxy service-name",
	Short: "describe a proxy server",
	Long: `The 'describe proxy' command shows information related to proxy servers including
all nodes running the proxy service as well as detailed connection information.`,
	ValidArgsFunction: completionProxies,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideProxyService)
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

		if len(proxyResults) == 0 {
			return fmt.Errorf("%s '%s'", proxyErrorMsg, serviceName)
		}

		err = json.Unmarshal(proxyResults, &proxiesSummary)
		if err != nil {
			return err
		}

		// get a list of node Id's while we search for the service name
		nodeIds := getProxyNodeIDs(serviceName, proxiesSummary)

		if len(nodeIds) == 0 {
			return fmt.Errorf("%s '%s'", proxyErrorMsg, serviceName)
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

		err = displayProxyDetails(cmd, dataFetcher, connection, "tcp", serviceResult, proxyResults)
		if err != nil {
			return err
		}

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

// getProxyConnectionsCmd represents the get proxy-connections command.
var getProxyConnectionsCmd = &cobra.Command{
	Use:               "proxy-connections service-name",
	Short:             "display proxy server connections for a specific proxy server",
	Long:              `The 'get proxy-connections' command displays proxy server connections for a specific proxy server.`,
	ValidArgsFunction: completionProxies,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideProxyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			connection  string
		)
		serviceName := args[0]

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var (
				proxiesSummary         = config.ProxiesSummary{}
				connectionDetailsFinal = make([]config.ProxyConnection, 0)
				connectionsResult      []byte
				proxyResults           []byte
				wg                     sync.WaitGroup
				errorSink              = createErrorSink()
				m                      = sync.RWMutex{}
			)

			proxyResults, err = dataFetcher.GetProxySummaryJSON()
			if err != nil {
				return err
			}

			if len(proxyResults) == 0 {
				return fmt.Errorf("%s '%s'", proxyErrorMsg, serviceName)
			}

			err = json.Unmarshal(proxyResults, &proxiesSummary)
			if err != nil {
				return err
			}

			nodeIds := getProxyNodeIDs(serviceName, proxiesSummary)
			nodeIdsLen := len(nodeIds)

			if nodeIdsLen == 0 {
				return fmt.Errorf("%s '%s'", proxyErrorMsg, serviceName)
			}

			wg.Add(nodeIdsLen)

			// retrieve all connection details from JSON
			for i := range nodeIds {
				go func(nodeID string) {
					defer wg.Done()
					connectionDetails := config.ProxyConnections{}
					data, err1 := dataFetcher.GetProxyConnectionsJSON(serviceName, nodeID)
					if err1 != nil {
						errorSink.AppendError(err1)
						return
					}
					err1 = json.Unmarshal(data, &connectionDetails)
					if err1 != nil {
						errorSink.AppendError(err1)
						return
					}
					// protect the slice for update
					m.Lock()
					defer m.Unlock()
					connectionDetailsFinal = append(connectionDetailsFinal, connectionDetails.Proxies...)
				}(nodeIds[i])
			}

			// wait for the results
			wg.Wait()
			errorList := errorSink.GetErrors()
			if len(errorList) > 0 {
				return utils.GetErrors(errorList)
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				connectionsResult, err = json.Marshal(connectionDetailsFinal)
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(connectionsResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else if OutputFormat == constants.JSON {
					cmd.Println(string(connectionsResult))
				}
			} else {
				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))
				cmd.Println(FormatProxyConnections(connectionDetailsFinal))
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}
			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

func getProxyNodeIDs(selectedService string, proxiesSummary config.ProxiesSummary) []string {
	// get a list of node Id's while we search for the service name
	nodeIDs := make([]string, 0)
	for _, value := range proxiesSummary.Proxies {
		if value.ServiceName == selectedService && value.Protocol == "tcp" {
			nodeIDs = append(nodeIDs, value.NodeID)
		}
	}
	return nodeIDs
}

func displayProxyDetails(cmd *cobra.Command, dataFetcher fetcher.Fetcher, connection, protocol string, serviceResult, proxyResults []byte) error {
	var (
		err         error
		finalResult []byte
		result      string
		details     = "PROXY SERVICE DETAILS"
		member      = "PROXY MEMBER DETAILS"
		header      = ""
	)

	if protocol == "http" {
		details = "HTTP SERVER SERVICE DETAILS"
		member = "HTTP SERVER MEMBER DETAILS"
		header = "------"
	}

	if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
		finalResult, err = utils.CombineByteArraysForJSON([][]byte{serviceResult, proxyResults}, []string{"services", "members"})
		if err != nil {
			return err
		}
		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err = utils.GetJSONPathResults(finalResult, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
			return nil
		}
		cmd.Println(string(finalResult))
		return nil
	}

	cmd.Print("\n" + details + "\n")
	cmd.Print("---------------------" + header + "\n")
	value, err := FormatJSONForDescribe(serviceResult, true, "Name", "Type")
	if err != nil {
		return err
	}

	cmd.Println(value)

	cmd.Print(member + "\n")
	cmd.Print("--------------------" + header + "\n")
	value, err = returnGetProxiesDetails(cmd, protocol, dataFetcher, connection)
	if err != nil {
		return err
	}

	cmd.Println(value)

	return nil
}

func returnGetProxiesDetails(cmd *cobra.Command, protocol string, dataFetcher fetcher.Fetcher, connection string) (string, error) {
	var sb strings.Builder
	for {
		var proxiesSummary = config.ProxiesSummary{}
		sb = strings.Builder{}

		proxyResults, err := dataFetcher.GetProxySummaryJSON()
		if err != nil {
			return "", err
		}

		if len(proxyResults) == 0 {
			return "", nil
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
			printWatchHeader(cmd)

			sb.WriteString(FormatCurrentCluster(connection) + "\n")

			err = json.Unmarshal(proxyResults, &proxiesSummary)
			if err != nil {
				return "", utils.GetError("unable to unmarshall proxy result", err)
			}
			sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, protocol))
		}

		// check to see if we should exit if we are not watching
		if !isWatchEnabled() {
			break
		}

		// if we are watching then output the details directly
		cmd.Println(sb.String())

		// we are watching so sleep and then repeat until CTRL-C
		time.Sleep(time.Duration(watchDelay) * time.Second)
	}

	return sb.String(), nil
}
