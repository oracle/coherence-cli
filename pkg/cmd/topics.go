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

// getTopicsCmd represents the get topics command
var getTopicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "display topics for a cluster",
	Long: `The 'get topics' command displays topics for a cluster. If 
no service name is specified then all services are queried.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err             error
			servicesSummary = config.ServicesSummaries{}
			connection      string
			dataFetcher     fetcher.Fetcher
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// get the services
		servicesResult, err := dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			data, err := dataFetcher.GetCachesSummaryJSONAllServices()
			if err != nil {
				return err
			}
			result, err := utils.GetJSONPathResults(data, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			data, err := dataFetcher.GetCachesSummaryJSONAllServices()
			if err != nil {
				return err
			}
			cmd.Println(string(data))
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			for {
				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				err = json.Unmarshal(servicesResult, &servicesSummary)
				if err != nil {
					return err
				}

				serviceList := GetListOfCacheServices(servicesSummary)

				if serviceName != "" {
					if !utils.SliceContains(serviceList, serviceName) {
						return fmt.Errorf("service '%s' was not found", serviceName)
					}

					// overwrite the list of services with the selected one
					serviceList = make([]string, 1)
					serviceList[0] = serviceName
				}

				value, err := formatTopicsSummary(serviceList, dataFetcher)
				if err != nil {
					return err
				}
				cmd.Println(value)

				// check to see if we should exit if we are not watching
				if !watchEnabled {
					break
				}
				// we are watching so sleep and then repeat until CTRL-C
				time.Sleep(time.Duration(watchDelay) * time.Second)
			}
		}

		return nil
	},
}

// formatTopicsSummary returns the formatted topics for the service list
func formatTopicsSummary(serviceList []string, dataFetcher fetcher.Fetcher) (string, error) {
	allCachesSummary, err := getCaches(serviceList, dataFetcher, true)
	if err != nil {
		return "", err
	}
	value := FormatTopicsSummary(allCachesSummary)
	if err != nil {
		return "", err
	}
	return value, err
}

func init() {
	getTopicsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
}
