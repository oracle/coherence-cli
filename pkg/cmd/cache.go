/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
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
	ignoreSpecialCaches  bool
	validAttributesCache = []string{"expiryDelay", "highUnits", "lowUnits", "batchFactor", "refreshFactor",
		"requeueThreshold"}
	nodeIDCache          string
	tier                 string
	InvalidTierMsg       = "tier must be back or front"
	cannotFindService    = "unable to find service with service name '%s'"
	cannotFindCache      = "no cache named %s exists for service %s"
	cannotFindViewCache  = "no view cache named %s exists for service %s"
	cacheSummary         bool
	partitionSortSize    bool
	partitionSortCount   bool
	partitionSortMaxSize bool
)

const (
	provideCacheMessage     = "you must provide a cache name"
	provideViewCacheMessage = "you must provide a view cache name"
	back                    = "back"
	all                     = "all"
	partitionDisplayType    = "partition"
)

// getCachesCmd represents the get caches command.
var getCachesCmd = &cobra.Command{
	Use:   "caches",
	Short: "display caches for a cluster",
	Long: `The 'get caches' command displays caches for a cluster. If no service
name is specified then all services are queried. You can specify '-o wide' to
display addition information. Use '-I' to ignore internal caches such as those
used by Federation.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
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

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				data, err := dataFetcher.GetCachesSummaryJSONAllServices()
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, data); err != nil {
					return err
				}
			} else {
				// get the services
				servicesResult, err := dataFetcher.GetServiceDetailsJSON()
				if err != nil {
					return err
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

				value, err := formatCachesSummary(serviceList, dataFetcher)
				if err != nil {
					return err
				}

				printWatchHeader(cmd)
				cmd.Println(FormatCurrentCluster(connection))

				cmd.Println(value)
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

// describeCacheCmd represents the describe cache command.
var describeCacheCmd = &cobra.Command{
	Use:   "cache cache-name",
	Short: "describe a cache",
	Long: `The 'describe cache' command displays information related to a specific cache. This
includes cache size, access, storage and index information across all nodes.
You can specify '-o wide' to display addition information.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
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
			cacheStoreDetails   = config.CacheStoreDetails{}
			found               bool
			connection          string
		)

		cacheName := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		if serviceName, err = findServiceForCacheOrTopic(dataFetcher, cacheName, "cache"); err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf(cannotFindService, serviceName)
		}

		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" || len(cacheResult) == 0 {
			return fmt.Errorf(cannotFindCache, cacheName, serviceName)
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
				if vCast["tier"] == back {
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

			value = FormatCacheDetailsSizeAndAccess(cacheDetails.Details)

			sb.WriteString("\nCACHE SIZE AND ACCESS DETAILS\n")
			sb.WriteString("-----------------------------\n")
			sb.WriteString(value)

			value = FormatCacheDetailsStorage(cacheDetails.Details)

			sb.WriteString("\nCACHE STORAGE DETAILS\n")
			sb.WriteString("---------------------\n")
			sb.WriteString(value)

			sb.WriteString("\nINDEX DETAILS\n")
			sb.WriteString("-------------\n")
			sb.WriteString(FormatCacheIndexDetails(cacheDetails.Details))

			if err = json.Unmarshal(cacheResult, &cacheStoreDetails); err != nil {
				return utils.GetError("unable to unmarshall storage result", err)
			}

			if hasCacheStores(cacheStoreDetails.Details) {
				sb.WriteString("\nCACHE STORE DETAILS\n")
				sb.WriteString("-------------------\n")

				// remove any values where tier != "back"
				finalDetails := ensureTierBack(cacheStoreDetails.Details)
				sb.WriteString(FormatCacheStoreDetails(finalDetails, cacheName, serviceName, false))
			}

			cmd.Println(sb.String())
		}

		return nil
	},
}

// describeViewCacheCmd represents the describe view-cache command.
var describeViewCacheCmd = &cobra.Command{
	Use:               "view-cache cache-name",
	Short:             "describe a cache",
	Long:              `The 'describe view-cache' command displays information related to a specific view-cache.`,
	ValidArgsFunction: completionViewCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideViewCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			found       bool
			connection  string
		)

		viewCacheName := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		if serviceName, err = findServiceForCacheOrTopic(dataFetcher, viewCacheName, "cache"); err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf(cannotFindService, serviceName)
		}

		if isJSONPathOrJSON() {
			cachesResult, err1 := dataFetcher.GetViewCachesDetailsJSON(serviceName, viewCacheName)
			if err1 != nil {
				return err1
			}
			if string(cachesResult) == "{}" || len(cachesResult) == 0 {
				return fmt.Errorf(cannotFindViewCache, viewCacheName, serviceName)
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(cachesResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else {
				cmd.Println(string(cachesResult))
			}
		} else {
			allCachesSummary, err := getViewCaches([]string{serviceName}, dataFetcher)
			if err != nil {
				return err
			}

			// only cluster view caches that match the name

			finalViewCachesSummary := make([]config.ViewCacheDetail, 0)

			for _, v := range allCachesSummary {
				if v.ViewName == viewCacheName {
					finalViewCachesSummary = append(finalViewCachesSummary, v)
				}
			}

			if len(finalViewCachesSummary) == 0 {
				return fmt.Errorf(cannotFindViewCache, viewCacheName, serviceName)
			}

			var sb strings.Builder
			sb.WriteString(FormatCurrentCluster(connection))

			sb.WriteString("\nService:    ")
			sb.WriteString(serviceName + "\n")
			sb.WriteString("View Cache: ")
			sb.WriteString(viewCacheName + "\n\n")

			sb.WriteString(FormatViewCacheDetail(finalViewCachesSummary))

			cmd.Println(sb.String())
		}

		return nil
	},
}

// getCacheStoresCmd represents the get cache-stores command.
var getCacheStoresCmd = &cobra.Command{
	Use:   "cache-stores cache-name",
	Short: "display cache stores for a cache and service",
	Long: `The 'get cache-stores' command displays cache store information related to a specific cache.
You can specify '-o wide' to display addition information.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			found       bool
			connection  string
		)

		cacheName := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		if serviceName, err = findServiceForCacheOrTopic(dataFetcher, cacheName, "cache"); err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf(cannotFindService, serviceName)
		}

		for {
			var (
				cacheStoreResult  []byte
				cacheStoreDetails = config.CacheStoreDetails{}
			)

			cacheStoreResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
			if err != nil {
				return err
			}

			if string(cacheStoreResult) == "{}" || len(cacheStoreResult) == 0 {
				return fmt.Errorf(cannotFindCache, cacheName, serviceName)
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(cacheStoreResult, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(cacheStoreResult))
			} else {
				if err = json.Unmarshal(cacheStoreResult, &cacheStoreDetails); err != nil {
					return utils.GetError("unable to unmarshall storage result", err)
				}

				// remove any values where tier != "back"
				finalDetails := ensureTierBack(cacheStoreDetails.Details)

				printWatchHeader(cmd)
				cmd.Println(FormatCurrentCluster(connection))

				if hasCacheStores(finalDetails) {
					cmd.Println(FormatCacheStoreDetails(finalDetails, cacheName, serviceName, true))
				}
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

// getCacheAccessCmd represents the get cache-access command.
var getCacheAccessCmd = &cobra.Command{
	Use:   "cache-access cache-name",
	Short: "display cache access information for a cache and service",
	Long: `The 'get cache-access' command displays cache access information related to a specific cache.
You can specify '-o wide' to display addition information.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return getCacheDetails(cmd, args, "access")
	},
}

// getCacheStorageCmd represents the get cache-storage command.
var getCacheStorageCmd = &cobra.Command{
	Use:   "cache-storage cache-name",
	Short: "display cache storage information for a cache and service",
	Long: `The 'get cache-storage' command displays cache storage information related to a specific cache.
You can specify '-o wide' to display addition information.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return getCacheDetails(cmd, args, "storage")
	},
}

// getCachePartitionsCmd represents the get cache-partitions command.
var getCachePartitionsCmd = &cobra.Command{
	Use:   "cache-partitions cache-name",
	Short: "display cache partition information for a cache and service",
	Long: `The 'get cache-partitions' command displays cache partition information related to a specific cache.
This information includes: partition id, count of entries and size of entries (binary key + value lengths).`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return getCacheDetails(cmd, args, partitionDisplayType)
	},
}

// getCacheIndexesCmd represents the get cache-indexes command.
var getCacheIndexesCmd = &cobra.Command{
	Use:               "cache-indexes cache-name",
	Short:             "display cache index information for a cache and service",
	Long:              `The 'get cache-indexes' command displays cache index information related to a specific cache.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return getCacheDetails(cmd, args, "index")
	},
}

// getViewCachesCmd represents the get view-caches command.
var getViewCachesCmd = &cobra.Command{
	Use:   "view-caches",
	Short: "display view caches for a cluster",
	Long: `The 'get view-caches' command displays view caches for a cluster. If no service
name is specified then all services are queried.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			connection  string
			dataFetcher fetcher.Fetcher
		)

		if OutputFormat == constants.WIDE {
			return errors.New("-o wide option is not available for view-caches")
		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var viewServicesSummary = config.ServicesSummaries{}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				data, err := dataFetcher.GetViewsSummaryJSONAllServices()
				if err != nil {
					return err
				}
				if err = processJSONOutput(cmd, data); err != nil {
					return err
				}
			} else {
				viewServicesResult, err := dataFetcher.GetServiceDetailsJSON()
				if err != nil {
					return err
				}

				err = json.Unmarshal(viewServicesResult, &viewServicesSummary)
				if err != nil {
					return err
				}

				serviceList := GetListOfCacheServices(viewServicesSummary)

				if serviceName != "" {
					if !utils.SliceContains(serviceList, serviceName) {
						return fmt.Errorf("service '%s' was not found", serviceName)
					}

					serviceList = make([]string, 1)
					serviceList[0] = serviceName
				}

				value, err := formatViewCachesSummary(serviceList, dataFetcher)
				if err != nil {
					return err
				}

				printWatchHeader(cmd)
				cmd.Println(FormatCurrentCluster(connection))

				cmd.Println(value)
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

// getCacheDetails displays either access, storage or index details for a cache depending upon
// the value of displayType.
func getCacheDetails(cmd *cobra.Command, args []string, displayType string) error {
	var (
		err         error
		dataFetcher fetcher.Fetcher
		found       bool
		connection  string
	)

	cacheName := args[0]

	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	if serviceName, err = findServiceForCacheOrTopic(dataFetcher, cacheName, "cache"); err != nil {
		return err
	}

	found, err = ServiceExists(dataFetcher, serviceName)
	if !found || err != nil {
		return fmt.Errorf(cannotFindService, serviceName)
	}

	for {
		var (
			cacheResult           []byte
			cacheDetails          = config.CacheDetails{}
			cachePartitionDetails = config.CachePartitionDetails{}
			result                string
		)

		if displayType == partitionDisplayType {
			cacheResult, err = dataFetcher.GetCachePartitions(serviceName, cacheName)
		} else {
			cacheResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
		}
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" || len(cacheResult) == 0 {
			return fmt.Errorf(cannotFindCache, cacheName, serviceName)
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err = utils.GetJSONPathResults(cacheResult, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(cacheResult))
		} else {
			if displayType == partitionDisplayType {
				err = json.Unmarshal(cacheResult, &cachePartitionDetails)
			} else {
				err = json.Unmarshal(cacheResult, &cacheDetails)
			}
			if err != nil {
				return utils.GetError("unable to unmarshall cache result", err)
			}

			printWatchHeader(cmd)
			cmd.Println(FormatCurrentCluster(connection))

			if displayType != partitionDisplayType {
				cmd.Printf("Cache: %s\n\n", args[0])
			}

			if displayType == "access" {
				cmd.Println(FormatCacheDetailsSizeAndAccess(cacheDetails.Details))
			} else if displayType == "index" {
				cmd.Println(FormatCacheIndexDetails(cacheDetails.Details))
			} else if displayType == "storage" {
				cmd.Println(FormatCacheDetailsStorage(cacheDetails.Details))
			} else if displayType == partitionDisplayType {
				cmd.Printf("Cache:            %s\n", args[0])
				cmd.Println(FormatCachePartitions(cachePartitionDetails.Details, cacheSummary))
			}
		}

		// check to see if we should exit if we are not watching
		if !isWatchEnabled() {
			break
		}
		// we are watching so sleep and then repeat until CTRL-C
		time.Sleep(time.Duration(watchDelay) * time.Second)
	}

	return nil
}

// clearCacheCmd represents the clear cache command.
var clearCacheCmd = &cobra.Command{
	Use:               "cache cache-name",
	Short:             "clear a caches contents",
	Long:              `The 'clear cache' command issues a clear against a specific cache.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCacheOperation(cmd, fetcher.ClearCache, args[0])
	},
}

// truncateCacheCmd represents the truncate cache command.
var truncateCacheCmd = &cobra.Command{
	Use:               "cache cache-name",
	Short:             "truncate a caches contents, which does not generate any cache events.",
	Long:              `The 'truncate cache' command issues a truncate against a specific cache. The truncate cache will not generate cache events.`,
	ValidArgsFunction: completionCaches,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideCacheMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeCacheOperation(cmd, fetcher.TruncateCache, args[0])
	},
}

func executeCacheOperation(cmd *cobra.Command, operation, cacheName string) error {
	var (
		err         error
		dataFetcher fetcher.Fetcher
		found       bool
		cacheData   []byte
	)

	_, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	if serviceName, err = findServiceForCacheOrTopic(dataFetcher, cacheName, "cache"); err != nil {
		return err
	}

	// validate that the service exists
	found, err = ServiceExists(dataFetcher, serviceName)
	if !found || err != nil {
		return fmt.Errorf(cannotFindService, serviceName)
	}

	// ensure the cache exists
	cacheData, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
	if err != nil {
		return err
	}

	if string(cacheData) == "{}" || len(cacheData) == 0 {
		return fmt.Errorf(cannotFindCache, cacheName, serviceName)
	}

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to %s cache %s in service %s? (y/n) ",
		operation, cacheName, serviceName)) {
		return nil
	}

	err = dataFetcher.InvokeStorageOperation(serviceName, cacheName, operation)
	if err == nil {
		cmd.Println(OperationCompleted)
	}

	return err
}

// ensureTierBack ensures that only back tier are included.
func ensureTierBack(cacheStoreDetails []config.CacheStoreDetail) []config.CacheStoreDetail {
	finalDetails := make([]config.CacheStoreDetail, 0)
	for _, v := range cacheStoreDetails {
		if v.Tier == back {
			finalDetails = append(finalDetails, v)
		}
	}

	return finalDetails
}

// hasCacheStores returns true of the collected cache store detail contains cache stores
// by checking the QueueSize. A value of -1 means no cache store configured.
func hasCacheStores(cacheStoreDetails []config.CacheStoreDetail) bool {
	for _, v := range cacheStoreDetails {
		if v.QueueSize == -1 {
			return false
		}
	}

	return true
}

// setCacheCmd represents the set cache command.
var setCacheCmd = &cobra.Command{
	Use:   "cache cache-name",
	Short: "set an attribute for a cache across one or more members",
	Long: `The 'set cache' command sets an attribute for a cache across one or member nodes.
The following attribute names are allowed: expiryDelay, highUnits, lowUnits,
batchFactor, refreshFactor or requeueThreshold.`,
	ValidArgsFunction: completionCaches,
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
			nodeIDs        []string
			nodeIDArray    []string
			confirmMessage string
			errorSink      = createErrorSink()
			wg             sync.WaitGroup
			floatValue     float64
			cacheName      = args[0]
			found          bool
			cacheResult    []byte
		)

		if tier != back && tier != "front" {
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

		if serviceName, err = findServiceForCacheOrTopic(dataFetcher, cacheName, "cache"); err != nil {
			return err
		}

		found, err = ServiceExists(dataFetcher, serviceName)
		if !found || err != nil {
			return fmt.Errorf(cannotFindService, serviceName)
		}

		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, cacheName)
		if err != nil {
			return err
		}

		if string(cacheResult) == "{}" {
			return fmt.Errorf(cannotFindCache, cacheName, serviceName)
		}

		// validate the nodes
		nodeIDArray, err = GetClusterNodeIDs(dataFetcher)
		if err != nil {
			return err
		}

		if nodeIDCache == all {
			nodeIDs = append(nodeIDs, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIDs))
		} else {
			if nodeIDs, err = getNodeIDs(nodeIDCache, nodeIDArray); err != nil {
				return err
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIDs))
		}

		cmd.Println(FormatCurrentCluster(connection))

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Selected service/cache: %s/%s\n", serviceName, cacheName)+
			fmt.Sprintf("Are you sure you want to set the value of attribute %s to %s in tier %s for %s? (y/n) ",
				attributeNameCache, attributeValueCache, tier, confirmMessage)) {
			return nil
		}

		nodeCount := len(nodeIDs)
		wg.Add(nodeCount)

		for _, value := range nodeIDs {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.SetCacheAttribute(nodeId, serviceName, cacheName, tier, attributeNameCache, floatValue)
				if err1 != nil {
					if strings.Contains(err1.Error(), "404") { //nolint:all
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

// formatCachesSummary returns the formatted caches for the service list.
func formatCachesSummary(serviceList []string, dataFetcher fetcher.Fetcher) (string, error) {
	allCachesSummary, err := getCaches(serviceList, dataFetcher)
	if err != nil {
		return "", err
	}

	// check for ignoring of special caches including '$'
	if ignoreSpecialCaches {
		finalList := make([]config.CacheSummaryDetail, 0)
		for _, v := range allCachesSummary {
			if !strings.Contains(v.CacheName, "$") {
				finalList = append(finalList, v)
			}
		}
		allCachesSummary = finalList
	}
	value := FormatCacheSummary(allCachesSummary)
	if err != nil {
		return "", err
	}
	return value, err
}

// formatViewCachesSummary returns the formatted view caches for the service list.
func formatViewCachesSummary(serviceList []string, dataFetcher fetcher.Fetcher) (string, error) {
	allCachesSummary, err := getViewCaches(serviceList, dataFetcher)
	if err != nil {
		return "", err
	}

	// build up the summary
	viewCacheSummary := make([]config.ViewCacheSummaryDetail, 0)

	// go through each service and get the view caches and counts of members
	for _, service := range serviceList {

		// get list of view caches for service
		caches := make(map[string]int)

		for _, value := range allCachesSummary {
			viewName := value.ViewName
			svcName := value.ServiceName

			if service == svcName {
				// find the cache in the map
				count := 1
				viewCount, ok := caches[viewName]
				if ok {
					// increment the count
					count = count + viewCount
				}
				caches[viewName] = count
			}
		}

		// now we have map of caches and member counts for the service, build the summary
		for k, v := range caches {
			viewCacheSummary = append(viewCacheSummary, config.ViewCacheSummaryDetail{
				ViewName:    k,
				ServiceName: service,
				MemberCount: int32(v),
			})
		}
	}

	value := FormatViewCacheSummary(viewCacheSummary)
	if err != nil {
		return "", err
	}
	return value, err
}

// getCaches returns a list of caches given a slice of services.
func getCaches(serviceList []string, dataFetcher fetcher.Fetcher) ([]config.CacheSummaryDetail, error) {
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

// getViewCaches returns a list of view caches given a slice of services.
func getViewCaches(serviceList []string, dataFetcher fetcher.Fetcher) ([]config.ViewCacheDetail, error) {
	var (
		wg               sync.WaitGroup
		allCachesSummary = make([]config.ViewCacheDetail, 0)
		errorSink        = createErrorSink()
		numServices      = len(serviceList)
		m                = sync.RWMutex{}
	)

	// loop through the services and build up the view cache list. carry out each service concurrently
	wg.Add(numServices)
	for _, service := range serviceList {
		go func(serviceNameValue string) {
			defer wg.Done()
			cachesResult, err := dataFetcher.GetViewCachesJSON(serviceNameValue)
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					// no view caches for this service, so finish with no error
					return
				}
				// must be another error so log it and finish
				errorSink.AppendError(err)
				return
			}

			cachesSummary := config.ViewCacheDetails{}
			err = json.Unmarshal(cachesResult, &cachesSummary)
			if err != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshal view caches result", err))
				return
			}

			finalCaches := make([]config.ViewCacheDetail, 0)

			for i := range cachesSummary.ViewCaches {
				// WebLogic Server doesn't return service attribute, so we need to fix it
				if cachesSummary.ViewCaches[i].ServiceName == "" {
					cachesSummary.ViewCaches[i].ServiceName = serviceNameValue
				}

				finalCaches = append(finalCaches, cachesSummary.ViewCaches[i])
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

// findServiceForCacheOrTopic attempts to find the service name for a cache or topic and will return
// the service name or an error indicating that the service and cache name is not unique.
func findServiceForCacheOrTopic(dataFetcher fetcher.Fetcher, cacheName, serviceType string) (string, error) {
	// if the serviceName is not blank then return it as the user has specified on command line
	if serviceName != "" {
		return serviceName, nil
	}

	var (
		err     error
		data    []byte
		caches  config.ServiceCaches
		service string
	)

	if serviceType == "cache" {
		data, err = dataFetcher.GetCachesSummaryJSONAllServices()
	} else {
		data, err = dataFetcher.GetTopicsJSON()
	}

	if err != nil {
		return "", err
	}

	if err = json.Unmarshal(data, &caches); err != nil {
		return "", err
	}

	serviceCount := 0

	// look through the details and see if there is only a single cache
	for _, value := range caches.Details {
		if value.Name == cacheName {
			serviceCount++
			service = value.ServiceName
		}
	}

	if serviceCount > 1 {
		return "", fmt.Errorf("there are multiple %ss named %s, please specify the service name", serviceType, cacheName)
	}
	if serviceCount == 0 {
		return "", fmt.Errorf("there are no %ss named %s for any services", serviceType, cacheName)
	}

	return service, nil
}

func init() {
	getCachesCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCachesCmd.Flags().BoolVarP(&ignoreSpecialCaches, "ignore-special", "I", false, "ignore caches with $ in name")

	describeCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	describeViewCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)

	getCacheStoresCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCacheAccessCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCacheStorageCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCacheIndexesCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCachePartitionsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getCachePartitionsCmd.Flags().BoolVarP(&cacheSummary, "summary", "S", false, "show a cache summary")
	getCachePartitionsCmd.Flags().BoolVarP(&partitionSortSize, "sort-size", "", false, "sort by size descendingFlag")
	getCachePartitionsCmd.Flags().BoolVarP(&partitionSortCount, "sort-count", "", false, "sort by count descendingFlag")
	getCachePartitionsCmd.Flags().BoolVarP(&partitionSortMaxSize, "sort-max", "", false, "sort by maximum entry descendingFlag")

	getViewCachesCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)

	clearCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	clearCacheCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	truncateCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	truncateCacheCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	setCacheCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setCacheCmd.Flags().StringVarP(&attributeNameCache, "attribute", "a", "", "attribute name to set")
	_ = setCacheCmd.MarkFlagRequired("attribute")
	setCacheCmd.Flags().StringVarP(&attributeValueCache, "value", "v", "", "attribute value to set")
	_ = setCacheCmd.MarkFlagRequired("value")
	setCacheCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	setCacheCmd.Flags().StringVarP(&nodeIDCache, "node", "n", all, "comma separated node ids to target")
	setCacheCmd.Flags().StringVarP(&tier, "tier", "t", back, "tier to apply to, back or front")
}
