/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
	"sync"
	"time"
)

// getPersistenceCmd represents the get persistence command
var getPersistenceCmd = &cobra.Command{
	Use:   "persistence",
	Short: "Display persistence details for a cluster",
	Long:  `The 'get persistence' command displays persistence information for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			servicesSummary = config.ServicesSummaries{}
			err             error
			dataFetcher     fetcher.Fetcher
			connection      string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			if watchEnabled {
				cmd.Println("\n" + time.Now().String())
			}
			servicesResult, err := dataFetcher.GetServiceDetailsJSON()

			if err != nil {
				return err
			}
			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(servicesResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(servicesResult))
			} else {
				cmd.Println(FormatCurrentCluster(connection))
				err = json.Unmarshal(servicesResult, &servicesSummary)
				if err != nil {
					return utils.GetError("unable to unmarshall service result", err)
				}

				deDuplicatedServices := DeduplicatePersistenceServices(servicesSummary)

				err = processPersistenceServices(deDuplicatedServices, dataFetcher)
				if err != nil {
					return err
				}

				cmd.Println(FormatPersistenceServices(deDuplicatedServices, true))
			}

			// check to see if we should exit if we are not watching
			if !watchEnabled {
				break
			}

			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

func processPersistenceServices(deDuplicatedServices []config.ServiceSummary, dataFetcher fetcher.Fetcher) error {
	var (
		wg        sync.WaitGroup
		errorSink = createErrorSink()
		m         = sync.RWMutex{}
	)

	wg.Add(len(deDuplicatedServices))

	// get the persistence coordinator details for each service
	for i, value := range deDuplicatedServices {
		go func(service string, index int) {
			defer wg.Done()
			var (
				data        []byte
				err1        error
				coordinator = config.PersistenceCoordinator{}
			)
			data, err1 = dataFetcher.GetPersistenceCoordinator(service)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}

			err1 = json.Unmarshal(data, &coordinator)
			if err1 != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshall persistence coordinator", err1))
				return
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			deDuplicatedServices[index].Idle = coordinator.Idle
			deDuplicatedServices[index].Snapshots = coordinator.Snapshots
			deDuplicatedServices[index].OperationStatus = coordinator.OperationStatus
		}(value.ServiceName, i)
	}

	// wait for the results
	wg.Wait()
	errorList := errorSink.GetErrors()
	if len(errorList) == 0 {
		return nil
	}
	return utils.GetErrors(errorList)
}

// DeduplicatePersistenceServices removes duplicated persistence details
func DeduplicatePersistenceServices(servicesSummary config.ServicesSummaries) []config.ServiceSummary {
	// the current results include 1 entry for each service and member, so we need to remove duplicates
	var finalServices = make([]config.ServiceSummary, 0)

	for _, value := range servicesSummary.Services {
		// only check distributed
		if !utils.IsDistributedCache(value.ServiceType) || !value.StorageEnabled {
			continue
		}
		// check to see if this service and member already exists in the finalServices
		if len(finalServices) == 0 {
			// no entries so add it anyway
			finalServices = append(finalServices, value)
		} else {
			var foundIndex = -1
			for i, v := range finalServices {
				if v.ServiceName == value.ServiceName {
					foundIndex = i
					break
				}
			}

			if foundIndex >= 0 {
				// update the existing service
				finalServices[foundIndex].PersistenceActiveSpaceUsed += value.PersistenceActiveSpaceUsed
				finalServices[foundIndex].PersistenceLatencyAverageTotal += value.PersistenceLatencyAverage
				if value.PersistenceLatencyMax > finalServices[foundIndex].PersistenceLatencyMax {
					finalServices[foundIndex].PersistenceLatencyMax = value.PersistenceLatencyMax
				}
			} else {
				// new service
				finalServices = append(finalServices, value)
			}
		}
	}
	return finalServices
}
