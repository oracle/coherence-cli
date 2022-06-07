/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	attributeNameCache   string
	attributeValueCache  string
	validAttributesCache = []string{"expiryDelay", "highUnits", "lowUnits", "batchFactor", "refreshFactor",
		"requeueThreshold"}
	nodeIDCache    string
	tier           string
	InvalidTierMsg = "tier must be back or front"
)

// getCachesCmd represents the get caches command
var getCachesCmd = &cobra.Command{
	Use:   "caches",
	Short: "display caches for a cluster",
	Long: `The 'get caches' command displays caches for a cluster. If 
no service name is specified then all services are queried. You
can specify '-o wide' to display addition information.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			connection  string
			dataFetcher fetcher.Fetcher
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var servicesSummary = config.ServicesSummaries{}

			// get the services
			servicesResult, err := dataFetcher.GetServiceDetailsJSON()
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				data, err := dataFetcher.GetCachesSummaryJSONAllServices()
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(data, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(data))
				}
			} else {
				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				cmd.Println(FormatCurrentCluster(connection))

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

				value, err := formatCachesSummary(serviceList, dataFetcher)
				if err != nil {
					return err
				}
				cmd.Println(value)

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

// describeCacheCmd represents the describe cache command
var describeCacheCmd = &cobra.Command{
	Use:   "cache cache-name",
	Short: "describe a cache",
	Long: `The 'describe cache' command displays information related to a specific cache. This
includes cache size, access, storage and index information across all nodes. You
can specify '-o wide' to display addition information.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a cache name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			cacheResult         []byte
			err                 error
			dataFetcher         fetcher.Fetcher
			jsonData            []byte
			cacheDetails        = config.CacheDetails{}
			cacheDetailsGeneric = config.GenericDetails{}
			found               bool
			connection          string
		)

		cacheName := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf("unable to find service with service name '%s'", serviceName)
		}

		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" || len(cacheResult) == 0 {
			return fmt.Errorf("no cache named %s exists for service %s", cacheName, serviceName)
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err := utils.GetJSONPathResults(cacheResult, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(cacheResult))
		} else {
			var sb strings.Builder
			sb.WriteString(FormatCurrentCluster(connection))

			// retrieve some header information
			err := json.Unmarshal(cacheResult, &cacheDetailsGeneric)
			if err != nil {
				return err
			}

			if len(cacheDetailsGeneric.Details) == 0 {
				return fmt.Errorf("no members found for cache %s and service %s", cacheName, serviceName)
			}

			// retrieve a storage enabled back tier to retrieve header details from
			for _, v := range cacheDetailsGeneric.Details {
				vCast := v.(map[string]interface{})
				if vCast["tier"] == "back" {
					jsonData, err = json.Marshal(vCast)
					if err != nil {
						return utils.GetError("unable tun unmarshall back tier", err)
					}
				}
			}

			sb.WriteString("\nCACHE DETAILS\n")
			sb.WriteString("-------------\n")
			value, err := FormatJSONForDescribe(jsonData, false,
				"Service", "Name", "Type", "Description", "Cache Store Type")
			if err != nil {
				return err
			}

			sb.WriteString(value)

			err = json.Unmarshal(cacheResult, &cacheDetails)
			if err != nil {
				return utils.GetError("unable to unmarshall cache result", err)
			}

			value, err = FormatCacheDetailsSizeAndAccess(cacheDetails.Details)
			if err != nil {
				return err
			}

			sb.WriteString("\nCACHE SIZE AND ACCESS DETAILS\n")
			sb.WriteString("-----------------------------\n")
			sb.WriteString(value)

			value, err = FormatCacheDetailsStorage(cacheDetails.Details)
			if err != nil {
				return err
			}

			sb.WriteString("\nCACHE STORAGE DETAILS\n")
			sb.WriteString("---------------------\n")
			sb.WriteString(value)

			sb.WriteString("\nINDEX DETAILS\n")
			sb.WriteString("-------------\n")

			sb.WriteString(FormatCacheIndexDetails(cacheDetails.Details))

			cmd.Println(sb.String())
		}

		return nil
	},
}

// setCacheCmd represents the set cache command
var setCacheCmd = &cobra.Command{
	Use:   "cache cache-name",
	Short: "set an attribute for a cache across one or more members",
	Long: `The 'set cache' command sets an attribute for a cache across one or member nodes.
The following attribute names are allowed: expiryDelay, highUnits, lowUnits,
batchFactor, refreshFactor or requeueThreshold.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a cache name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher    fetcher.Fetcher
			connection     string
			err            error
			nodeIds        []string
			nodeIDArray    []string
			confirmMessage string
			response       string
			errorSink      = createErrorSink()
			wg             sync.WaitGroup
			floatValue     float64
			cacheName      = args[0]
			found          bool
			cacheResult    []byte
		)

		if tier != "back" && tier != "front" {
			return errors.New(InvalidTierMsg)
		}

		if !utils.SliceContains(validAttributesCache, attributeNameCache) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of\n%v",
				attributeNameCache, validAttributesCache)
		}

		// validate the attribute value
		floatValue, err = strconv.ParseFloat(attributeValueCache, 64)
		if err != nil {
			return fmt.Errorf("invalid float value of %s for attribute %s", attributeValue, attributeNameCache)
		}

		// carry out some basic validation
		if floatValue < 0 {
			return fmt.Errorf("value for attribute %s must be greater or equal to zero", attributeNameCache)
		}

		// validate the cache and service
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf("unable to find service with service name '%s'", serviceName)
		}

		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" {
			return fmt.Errorf("no cache named %s exists for service %s", cacheName, serviceName)
		}

		// validate the nodes
		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if nodeIDCache == "all" {
			nodeIds = append(nodeIds, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIds))
		} else {
			nodeIds = strings.Split(nodeIDCache, ",")
			for _, value := range nodeIds {
				if !utils.IsValidInt(value) {
					return fmt.Errorf("invalid value for node id of %s", value)
				}

				if !utils.SliceContains(nodeIDArray, value) {
					return fmt.Errorf("no node with node id %s exists in this cluster", value)
				}
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIds))
		}

		cmd.Println(FormatCurrentCluster(connection))

		if !automaticallyConfirm {
			cmd.Printf("Selected service/cache: %s/%s\n", serviceName, cacheName)
			cmd.Printf("Are you sure you want to set the value of attribute %s to %s in tier %s for %s? (y/n) ",
				attributeNameCache, attributeValueCache, tier, confirmMessage)
			_, err = fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
		}

		nodeCount := len(nodeIds)
		wg.Add(nodeCount)

		for _, value := range nodeIds {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.SetCacheAttribute(nodeId, serviceName, cacheName, tier, attributeNameCache, floatValue)
				if err1 != nil {
					if strings.Contains(err1.Error(), "404") {
						// ignore as this is likely trying to set a value for a back tier where the member is a near cache
					} else {
						errorSink.AppendError(err1)
					}
				}
			}(value)
		}

		wg.Wait()
		errorList := errorSink.GetErrors()

		if len(errorList) > 0 {
			return utils.GetErrors(errorList)
		}
		cmd.Println(OperationCompleted)

		return nil
	},
}

// formatCachesSummary returns the formatted caches for the service list
func formatCachesSummary(serviceList []string, dataFetcher fetcher.Fetcher) (string, error) {
	allCachesSummary, err := getCaches(serviceList, dataFetcher, false)
	if err != nil {
		return "", err
	}
	value := FormatCacheSummary(allCachesSummary)
	if err != nil {
		return "", err
	}
	return value, err
}

// getCaches returns a list of caches given a slice of services
func getCaches(serviceList []string, dataFetcher fetcher.Fetcher, topicsOnly bool) ([]config.CacheSummaryDetail, error) {
	var (
		wg               sync.WaitGroup
		allCachesSummary = make([]config.CacheSummaryDetail, 0)
		errorSink        = createErrorSink()
		numServices      = len(serviceList)
		m                = sync.RWMutex{}
	)

	// loop through the services and build up the cache list. carry out each service concurrently
	wg.Add(numServices)
	for _, service := range serviceList {
		go func(serviceNameValue string) {
			defer wg.Done()
			cachesResult, err := dataFetcher.GetCachesSummaryJSON(serviceNameValue)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					// no caches for this service, so finish with no error
					return
				}
				// must be another error so log it and finish
				errorSink.AppendError(err)
				return
			}

			cachesSummary := config.CacheSummaries{}
			err = json.Unmarshal(cachesResult, &cachesSummary)
			if err != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshal caches result", err))
				return
			}

			finalCaches := make([]config.CacheSummaryDetail, 0)

			for i := range cachesSummary.Caches {
				if topicsOnly && !strings.Contains(cachesSummary.Caches[i].CacheName, "$topic$") {
					continue
				}

				// WebLogic Server doesn't return service attribute, so we need to fix it
				if cachesSummary.Caches[i].ServiceName == "" {
					cachesSummary.Caches[i].ServiceName = serviceNameValue
				}

				finalCaches = append(finalCaches, cachesSummary.Caches[i])
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			allCachesSummary = append(allCachesSummary, finalCaches...)
		}(service)
	}

	// wait for the results
	wg.Wait()

	errorList := errorSink.GetErrors()

	if len(errorList) > 0 {
		return nil, utils.GetErrors(errorList)
	}

	return allCachesSummary, nil
}

func init() {
	getCachesCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)

	describeCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = describeCacheCmd.MarkFlagRequired(serviceNameOption)

	setCacheCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setCacheCmd.Flags().StringVarP(&attributeNameCache, "attribute", "a", "", "attribute name to set")
	_ = setCacheCmd.MarkFlagRequired("attribute")
	setCacheCmd.Flags().StringVarP(&attributeValueCache, "value", "v", "", "attribute value to set")
	_ = setCacheCmd.MarkFlagRequired("value")
	setCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = setCacheCmd.MarkFlagRequired(serviceNameOption)
	setCacheCmd.Flags().StringVarP(&nodeIDCache, "node", "n", "all", "comma separated node ids to target")
	setCacheCmd.Flags().StringVarP(&tier, "tier", "t", "back", "tier to apply to, back or front")
}
