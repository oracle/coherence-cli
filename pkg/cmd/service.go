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
	serviceType       string
	statusHATimeout   int32
	statusHAType      string
	validServiceTypes = []string{constants.DistributedService, constants.FederatedService, constants.PagedTopic,
		"Invocation", "Proxy", "RemoteCache", "ReplicatedCache", "OptimisticCache", "LocalCache", "RemoteGrpcCache", all}
	validStatusHA = []string{"NODE-SAFE", "MACHINE-SAFE", "RACK-SAFE", "SITE-SAFE"}
	allStatusHA   = []string{"ENDANGERED", "NODE-SAFE", "MACHINE-SAFE", "RACK-SAFE", "SITE-SAFE"}

	attributeNameService   string
	attributeValueService  string
	validAttributesService = []string{"threadCount", "threadCountMin", "threadCountMax",
		"taskHungThresholdMillis", "requestTimeoutMillis"}
	nodeIDService          string
	nodeIDServiceOperation string
	excludeStorageDisabled bool
	includeBacklogOnly     bool
)

const (
	serviceUse          = "service service-name"
	unableToFindService = "unable to find service with service name '%s'"
	noDistributionsData = "No distributions data is available"
	serviceUnmarshall   = "unable to unmarshall members result"
)

// getServicesCmd represents the get services command.
var getServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "display services for a cluster",
	Long: `The 'get services' command displays services for a cluster using various options. 
You may specify the service type as well a status-ha value to wait for. You
can also specify '-o wide' to display addition information.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err            error
			statusHAValues []string
			dataFetcher    fetcher.Fetcher
			connection     string
		)

		if statusHAType != "none" {
			if !isWatchEnabled() {
				return errors.New("if you have specified a status-ha value then you must enable watch option")
			}
			if !utils.SliceContains(validStatusHA, statusHAType) {
				return fmt.Errorf("the list of Status-HA values must be one of %v", validStatusHA)
			}
		}

		// check service type
		if serviceType != "" && !utils.SliceContains(validServiceTypes, serviceType) {
			return fmt.Errorf("invalid service type of '%s' specified. \nValid types are %v", serviceType, validServiceTypes)
		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		startTime := time.Now()

		for {
			var servicesSummary = config.ServicesSummaries{}

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
				err = json.Unmarshal(servicesResult, &servicesSummary)
				if err != nil {
					return utils.GetError("unable to unmarshall service result", err)
				}

				deDuplicatedServices := DeduplicateServices(servicesSummary, serviceType)

				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))
				cmd.Println(FormatServices(deDuplicatedServices))

				// collect all the statusHA values
				statusHAValues = make([]string, 0)
				for _, value := range deDuplicatedServices {
					// ignore n/a
					if value.StatusHA != "n/a" && !utils.SliceContains(statusHAValues, value.StatusHA) {
						statusHAValues = append(statusHAValues, value.StatusHA)
					}
				}
			}

			// check to see if we should exit if we are not watching
			if !isWatchEnabled() {
				break
			}

			// if we have specified a statusHA value to wait for then process this
			if statusHAType != "none" {
				elapsedSeconds := int32(time.Since(startTime).Seconds())
				if len(statusHAValues) == 1 && isStatusHASaferThan(statusHAValues[0], statusHAType) {
					cmd.Printf("Status HA value of %s or better reached in %d seconds for service types of '%s'\n",
						statusHAType, elapsedSeconds, serviceType)
					return nil
				}

				if elapsedSeconds > statusHATimeout {
					return fmt.Errorf("status HA value of %s or better NOT reached in %d seconds for service types of '%s'",
						statusHAType, elapsedSeconds, serviceType)
				}

				cmd.Printf("Waiting for Status HA value %s or better for service type '%s' within %d seconds",
					statusHAType, serviceType, statusHATimeout)
			}

			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// getServiceStorageCmd represents the get service-storage command.
var getServiceStorageCmd = &cobra.Command{
	Use:   "service-storage",
	Short: "display partitioned services storage information for a cluster",
	Long: `The 'get service-storage' command displays partitioned services storage for a cluster including
information regarding partition sizes.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			connection  string
			jsonData    []byte
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			storageSummary, err := getServiceStorageDetails(dataFetcher)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				// serialize the data to JSON
				jsonData, err = json.Marshal(storageSummary)
				if err != nil {
					return err
				}

				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(jsonData, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(jsonData))
				}
			} else {
				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))
				cmd.Println(FormatServicesStorage(storageSummary))
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

func getServiceStorageDetails(dataFetcher fetcher.Fetcher) ([]config.ServiceStorageSummary, error) {
	var (
		servicesSummary = config.ServicesSummaries{}
		serviceList     = make([]string, 0)
		errorSink       = createErrorSink()
		wg              sync.WaitGroup
		m               = sync.RWMutex{}
	)

	// get the list of services
	servicesResult, err := dataFetcher.GetServiceDetailsJSON()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(servicesResult, &servicesSummary)
	if err != nil {
		return nil, err
	}

	// get the list of partitioned services
	for _, v := range servicesSummary.Services {
		if utils.IsDistributedCache(v.ServiceType) {
			if !utils.SliceContains(serviceList, v.ServiceName) {
				serviceList = append(serviceList, v.ServiceName)
			}
		}
	}

	if len(serviceList) == 0 {
		return nil, errors.New("unable to find any partitioned services")
	}

	// now we have partitioned services, retrieve the data for each service
	wg.Add(len(serviceList))

	storageSummary := make([]config.ServiceStorageSummary, 0)

	for _, service := range serviceList {
		go func(serviceName string) {
			var data = config.ServiceStorageSummary{}
			defer wg.Done()
			partitionsData, err1 := dataFetcher.GetServicePartitionsJSON(serviceName)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
			err1 = json.Unmarshal(partitionsData, &data)
			if err1 != nil {
				errorSink.AppendError(utils.GetError("unable to unmarshall storage data", err1))
				return
			}

			// protect the slice for update
			m.Lock()
			defer m.Unlock()
			storageSummary = append(storageSummary, data)
		}(service)
	}

	wg.Wait()

	errorList := errorSink.GetErrors()
	if len(errorList) != 0 {
		return nil, utils.GetErrors(errorList)
	}

	return storageSummary, nil
}

// getServiceDistributionsCmd represents the get service-distributions command.
var getServiceDistributionsCmd = &cobra.Command{
	Use:   "service-distributions service-name",
	Short: "display partition distributions information for a service",
	Long:  `The 'get service-distributions' command displays partition distributions for a service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	ValidArgsFunction: completionDistributedService,
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

		servicesResult, err := GetDistributedServices(dataFetcher)
		if err != nil {
			return err
		}

		if !utils.SliceContains(servicesResult, args[0]) {
			return fmt.Errorf(unableToFindService, args[0])
		}

		for {
			var (
				distributionsData []byte
				distributions     config.Distributions
			)
			distributionsData, err = dataFetcher.GetScheduledDistributionsJSON(args[0])
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(distributionsData, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(distributionsData))
				}
			} else {
				printWatchHeader(cmd)

				if len(distributionsData) != 0 {
					err = json.Unmarshal(distributionsData, &distributions)
					if err != nil {
						return err
					}
				} else {
					distributions.ScheduledDistributions = noDistributionsData
				}

				cmd.Println(FormatCurrentCluster(connection))
				cmd.Println(distributions.ScheduledDistributions)
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

// getServiceOwnershipCmd represents the get service-ownership command.
var getServiceOwnershipCmd = &cobra.Command{
	Use:   "service-ownership service-name",
	Short: "display partition ownership information for a service",
	Long:  `The 'get service-ownership' command displays partition ownership for a service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	ValidArgsFunction: completionDistributedService,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err            error
			dataFetcher    fetcher.Fetcher
			connection     string
			membersResult  []byte
			membersDetails = config.ServiceMemberDetails{}
			memberNodeID   string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		servicesResult, err := GetDistributedServices(dataFetcher)
		if err != nil {
			return err
		}

		if !utils.SliceContains(servicesResult, args[0]) {
			return fmt.Errorf(unableToFindService, args[0])
		}

		// find storage member node
		membersResult, err = dataFetcher.GetServiceMembersDetailsJSON(args[0])
		if err != nil {
			return err
		}

		err = json.Unmarshal(membersResult, &membersDetails)
		if err != nil {
			return utils.GetError(serviceUnmarshall, err)
		}

		// find the first node
		for _, v := range membersDetails.Services {
			memberNodeID = v.NodeID
			break
		}

		if memberNodeID == "" {
			return fmt.Errorf("cannot find a node for service %s", args[0])
		}

		for {
			var ownershipData []byte

			ownershipData, err = dataFetcher.GetServiceOwnershipJSON(args[0], memberNodeID)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(ownershipData, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(ownershipData))
				}
			} else {
				printWatchHeader(cmd)

				result, err := getOwnershipData(dataFetcher, ownershipData)
				if err != nil {
					return err
				}
				cmd.Println(FormatCurrentCluster(connection))
				cmd.Println(FormatPartitionOwnership(result))
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

func getOwnershipData(dataFetcher fetcher.Fetcher, ownershipData []byte) (map[int]*config.PartitionOwnership, error) {
	var ownership config.Ownership

	if len(ownershipData) != 0 {
		err := json.Unmarshal(ownershipData, &ownership)
		if err != nil {
			return nil, err
		}
	} else {
		ownership.Details = ""
	}

	results, err := utils.ParsePartitionOwnership(ownership.Details)
	if err != nil {
		return nil, err
	}

	if OutputFormat == constants.WIDE {
		var (
			members       = config.Members{}
			membersResult []byte
		)

		membersResult, err = dataFetcher.GetMemberDetailsJSON(OutputFormat != constants.TABLE && OutputFormat != constants.WIDE)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(membersResult, &members)
		if err != nil {
			return nil, utils.GetError(unableToDecode, err)
		}

		// retrieve the machine, rack and site for display in wide mode
		for _, v := range results {
			v.Machine, v.Rack, v.Site = getMachineRackSite(fmt.Sprintf("%v", v.MemberID), members.Members)
		}
	}

	return results, nil
}

func getMachineRackSite(nodeID string, members []config.Member) (string, string, string) {
	for _, v := range members {
		if v.NodeID == nodeID {
			return v.MachineName, v.RackName, v.SiteName
		}
	}
	return "", "", ""
}

// getServiceDescriptionCmd represents the get service-description command.
var getServiceDescriptionCmd = &cobra.Command{
	Use:   "service-description service-name",
	Short: "display description for a service",
	Long: `The 'get service-description' command displays information regarding a service and it's members.
Only available in most recent Coherence versions.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	ValidArgsFunction: completionService,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err             error
			dataFetcher     fetcher.Fetcher
			connection      string
			servicesSummary = config.ServicesSummaries{}
			serviceResult   []byte
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		serviceResult, err = dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(serviceResult, &servicesSummary)
		if err != nil {
			return err
		}

		if found := serviceExists(args[0], servicesSummary); !found {
			return fmt.Errorf(unableToFindService, args[0])
		}

		for {
			var (
				descriptionData []byte
				description     config.Description
			)

			descriptionData, err = dataFetcher.GetServiceDescriptionJSON(args[0])
			if err != nil {
				return err
			}
			if len(descriptionData) != 0 {
				err = json.Unmarshal(descriptionData, &description)
				if err != nil {
					return err
				}
			} else {
				return nil
			}

			printWatchHeader(cmd)

			cmd.Println(FormatCurrentCluster(connection))
			cmd.Println(description.Description)

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

// getServiceMembersCmd represents the get service-members command.
var getServiceMembersCmd = &cobra.Command{
	Use:               "service-members service-name",
	Short:             "display service members for a cluster",
	Long:              `The 'get service-members' command displays service members for a cluster.`,
	ValidArgsFunction: completionService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err             error
			dataFetcher     fetcher.Fetcher
			serviceResult   []byte
			connection      string
			servicesSummary = config.ServicesSummaries{}
		)

		serviceName := args[0]

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		serviceResult, err = dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(serviceResult, &servicesSummary)
		if err != nil {
			return err
		}

		found := serviceExists(serviceName, servicesSummary)

		if !found {
			return fmt.Errorf(unableToFindService, serviceName)
		}
		for {
			var (
				membersResult  []byte
				membersDetails = config.ServiceMemberDetails{}
			)
			membersResult, err = dataFetcher.GetServiceMembersDetailsJSON(serviceName)
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(membersResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
					return nil
				}
				cmd.Println(string(membersResult))
			} else {
				printWatchHeader(cmd)
				cmd.Println(FormatCurrentCluster(connection))

				err = json.Unmarshal(membersResult, &membersDetails)
				if err != nil {
					return utils.GetError(serviceUnmarshall, err)
				}

				var finalDetails []config.ServiceMemberDetail

				if excludeStorageDisabled {
					// remove any entries where idle threads == -1 which indicates client
					var newMemberDetails = make([]config.ServiceMemberDetail, 0)
					for _, v := range membersDetails.Services {
						if v.OwnedPartitionsPrimary > 0 {
							newMemberDetails = append(newMemberDetails, v)
						}
					}
					finalDetails = newMemberDetails
				} else if includeBacklogOnly {
					// Only include members with backlog > 0
					var newMemberDetails = make([]config.ServiceMemberDetail, 0)
					for _, v := range membersDetails.Services {
						if v.TaskBacklog > 0 {
							newMemberDetails = append(newMemberDetails, v)
						}
					}
					finalDetails = newMemberDetails
				} else {
					finalDetails = membersDetails.Services
				}

				cmd.Println("Service: " + serviceName + "\n")
				cmd.Println(FormatServiceMembers(finalDetails))
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

// describeService represents the describe service command.
var describeServiceCmd = &cobra.Command{
	Use:   serviceUse,
	Short: "describe a service",
	Long: `The 'describe service' command shows information related to services. This
includes information about each service member as well as Persistence information if the
service is a cache service.`,
	ValidArgsFunction: completionService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			servicesSummary    = config.ServicesSummaries{}
			membersDetails     = config.ServiceMemberDetails{}
			persistenceDetails = config.ServicesSummaries{}
			proxiesSummary     = config.ProxiesSummary{}
			coordinator        = config.PersistenceCoordinator{}
			serviceResult      []byte
			membersResult      []byte
			proxyResults       []byte
			coordData          []byte
			distributionsData  []byte
			partitionsData     []byte
			err                error
			dataFetcher        fetcher.Fetcher
			connection         string
			errorSink          = createErrorSink()
			wg                 sync.WaitGroup
		)

		serviceName := args[0]

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		serviceResult, err = dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(serviceResult, &servicesSummary)
		if err != nil {
			return err
		}

		if found := serviceExists(serviceName, servicesSummary); !found {
			return fmt.Errorf(unableToFindService, serviceName)
		}

		// we have valid service name so issue queries in parallel
		wg.Add(5)
		go func() {
			defer wg.Done()
			var err1 error
			serviceResult, err1 = dataFetcher.GetSingleServiceDetailsJSON(serviceName)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			membersResult, err1 = dataFetcher.GetServiceMembersDetailsJSON(serviceName)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			proxyResults, err1 = dataFetcher.GetProxySummaryJSON()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			partitionsData, err = dataFetcher.GetServicePartitionsJSON(serviceName)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			distributionsData, err1 = dataFetcher.GetScheduledDistributionsJSON(serviceName)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		// wait for all data fetchers requests to complete
		wg.Wait()

		errorList := errorSink.GetErrors()

		// check if any of the requests returned errors
		if len(errorList) > 0 {
			return utils.GetErrors(errorList)
		}

		if isJSONPathOrJSON() {
			finalResult, err := utils.CombineByteArraysForJSON([][]byte{serviceResult, proxyResults,
				membersResult, proxyResults, distributionsData},
				[]string{"services", "proxies", "members", "partitions", "distribution"})
			if err != nil {
				return err
			}
			if err = processJSONOutput(cmd, finalResult); err != nil {
				return err
			}
		} else {
			var sb strings.Builder
			sb.WriteString(FormatCurrentCluster(connection))
			sb.WriteString("\nSERVICE DETAILS\n")
			sb.WriteString("---------------\n")
			value, err := FormatJSONForDescribe(serviceResult, true, "Name", "Type")
			if err != nil {
				return err
			}
			sb.WriteString(value)

			sb.WriteString("\nSERVICE MEMBERS\n")
			sb.WriteString("---------------\n")

			err = json.Unmarshal(membersResult, &membersDetails)
			if err != nil {
				return utils.GetError(serviceUnmarshall, err)
			}

			err = json.Unmarshal(membersResult, &persistenceDetails)
			if err != nil {
				return utils.GetError("unable to unmarshall members persistence result", err)
			}

			value = FormatServiceMembers(membersDetails.Services)

			sb.WriteString(value)

			if len(proxyResults) != 0 {
				err = json.Unmarshal(proxyResults, &proxiesSummary)
				if err != nil {
					return utils.GetError("unable to unmarshall proxy result", err)
				}
			}

			// filter out any proxy servers that are not for this service
			hasProxies := false
			hasHTTPServers := false
			filteredProxies := make([]config.ProxySummary, 0)
			for _, value := range proxiesSummary.Proxies {
				if value.ServiceName == serviceName {
					if value.Protocol == "tcp" {
						hasProxies = true
					}
					if value.Protocol == "http" {
						hasHTTPServers = true
					}
					filteredProxies = append(filteredProxies, value)
				}
			}

			if !hasProxies && !hasHTTPServers {
				sb.WriteString("\nSERVICE CACHES\n")
				sb.WriteString("--------------\n")

				serviceList := make([]string, 1)
				serviceList[0] = serviceName

				value, err = formatCachesSummary(serviceList, dataFetcher)
				if err != nil {
					return err
				}
				sb.WriteString(value)

				sb.WriteString("\nPERSISTENCE FOR SERVICE\n")
				sb.WriteString("-----------------------\n")

				value = FormatPersistenceServices(persistenceDetails.Services, false)
				sb.WriteString(value)

				coordData, err = dataFetcher.GetPersistenceCoordinator(serviceName)
				if err != nil {
					return err
				}

				if len(coordData) > 0 {
					err = json.Unmarshal(coordData, &coordinator)
					if err != nil {
						return err
					}
				}

				value, err = FormatJSONForDescribe(coordData, false,
					"Coordinator Id", "Idle", "Operation Status", "Snapshots")
				if err != nil {
					return err
				}
				sb.WriteString("\nPERSISTENCE COORDINATOR\n")
				sb.WriteString("-----------------------\n")
				sb.WriteString(value)

				if len(distributionsData) != 0 {
					value, err = FormatJSONForDescribe(distributionsData, true)
					if err != nil {
						return err
					}
				} else {
					value = noDistributionsData
				}

				if value != "" {
					sb.WriteString("\nDISTRIBUTION INFORMATION\n")
					sb.WriteString("------------------------\n")
					sb.WriteString(value)
				}

				if string(partitionsData) != "" {
					value, err = FormatJSONForDescribe(partitionsData, true,
						"Service", "Strategy Name")
					if err != nil {
						return err
					}
				} else {
					value = ""
				}

				if value != "" {
					sb.WriteString("\nPARTITION INFORMATION\n")
					sb.WriteString("---------------------\n")
					sb.WriteString(value)
				}
			}

			if hasProxies {
				sb.WriteString("\nSERVICE PROXY SERVERS\n")
				sb.WriteString("---------------------\n")
				sb.WriteString(FormatProxyServers(filteredProxies, "tcp"))
			}

			if hasHTTPServers {
				sb.WriteString("\nSERVICE HTTP SERVERS\n")
				sb.WriteString("--------------------\n")
				sb.WriteString(FormatProxyServers(filteredProxies, "http"))
			}

			cmd.Println(sb.String())
		}

		return nil
	},
}

// setServiceCmd represents the set service command.
var setServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "set a service attribute across one or more members",
	Long: `The 'set service' command sets an attribute for a service across one or member nodes.
The following attribute names are allowed: threadCount, threadCountMin, threadCountMax or
taskHungThresholdMillis or requestTimeoutMillis.`,
	ValidArgsFunction: completionService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			servicesSummary = config.ServicesSummaries{}
			serviceResult   []byte
			err             error
			dataFetcher     fetcher.Fetcher
			connection      string
			nodeIDArray     []string
			nodeIDs         []string
			confirmMessage  string
			intValue        int
			errorSink       = createErrorSink()
			wg              sync.WaitGroup
		)

		serviceName := args[0]

		// validate the attribute name
		if !utils.SliceContains(validAttributesService, attributeNameService) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of\n%v",
				attributeNameService, validAttributesService)
		}

		// validate the attribute value
		intValue, err = strconv.Atoi(attributeValueService)
		if err != nil {
			return fmt.Errorf("invalid integer value of %s for attribute %s", attributeValueService, attributeNameService)
		}

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		serviceResult, err = dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(serviceResult, &servicesSummary)
		if err != nil {
			return err
		}

		found := false
		for _, v := range servicesSummary.Services {
			if v.ServiceName == serviceName {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf(unableToFindService, serviceName)
		}

		// validate the nodes
		nodeIDArray, err = GetClusterNodeIDs(dataFetcher)
		if err != nil {
			return err
		}

		if nodeIDService == all {
			nodeIDs = append(nodeIDs, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIDs))
		} else {
			if nodeIDs, err = getNodeIDs(nodeIDService, nodeIDArray); err != nil {
				return err
			}
			confirmMessage = fmt.Sprintf("%d node(s)", len(nodeIDs))
		}

		cmd.Println(FormatCurrentCluster(connection))

		cmd.Printf("Selected service: %s\n", serviceName)
		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
			attributeNameService, attributeValueService, confirmMessage)) {
			return nil
		}

		wg.Add(len(nodeIDs))

		for _, value := range nodeIDs {
			go func(nodeId string) {
				var err1 error
				defer wg.Done()
				_, err1 = dataFetcher.SetServiceAttribute(nodeId, serviceName, attributeNameService, intValue)
				if err1 != nil {
					errorSink.AppendError(err1)
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

// suspendServiceCmd represents the suspend service command.
var suspendServiceCmd = &cobra.Command{
	Use:               serviceUse,
	Short:             "suspend a service",
	Long:              `The 'suspend service' command suspends a specific service in all the members of a cluster.`,
	ValidArgsFunction: completionPersistenceService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceCommand(cmd, args[0], fetcher.SuspendService)
	},
}

// resumeServiceCmd represents the resume service command.
var resumeServiceCmd = &cobra.Command{
	Use:               serviceUse,
	Short:             "resume a service",
	Long:              `The 'resume service' command resumes a specific service in all the members of a cluster.`,
	ValidArgsFunction: completionPersistenceService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceCommand(cmd, args[0], fetcher.ResumeService)
	},
}

// stopServiceCmd represents the stop service command.
var stopServiceCmd = &cobra.Command{
	Use:   serviceUse,
	Short: "stop a service",
	Long: `The 'stop service' command forces a specific service to stop on a cluster member.
Use the shutdown service command for normal service termination.`,
	ValidArgsFunction: completionPersistenceService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.StopService)
	},
}

// startServiceCmd represents the start service command.
var startServiceCmd = &cobra.Command{
	Use:               serviceUse,
	Short:             "start a service",
	Long:              `The 'start service' command starts a specific service on a cluster member.`,
	ValidArgsFunction: completionPersistenceService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.StartService)
	},
}

// shutdownServiceCmd represents the shutdown service command.
var shutdownServiceCmd = &cobra.Command{
	Use:   serviceUse,
	Short: "shutdown a service",
	Long: `The 'shutdown service' command performs a controlled shut-down of a specific service
on a cluster member. Shutting down a service is preferred over stopping a service.`,
	ValidArgsFunction: completionPersistenceService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.ShutdownService)
	},
}

// issueServiceNodeCommand issues an operation against a service member.
func issueServiceNodeCommand(cmd *cobra.Command, serviceName, operation string) error {
	var (
		dataFetcher     fetcher.Fetcher
		connection      string
		err             error
		serviceResult   []byte
		servicesSummary = config.ServicesSummaries{}
		nodeIDArray     []string
	)
	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	cmd.Println(FormatCurrentCluster(connection))

	serviceResult, err = dataFetcher.GetServiceDetailsJSON()
	if err != nil {
		return err
	}

	err = json.Unmarshal(serviceResult, &servicesSummary)
	if err != nil {
		return err
	}

	found := serviceExists(serviceName, servicesSummary)

	if !found {
		return fmt.Errorf(unableToFindService, serviceName)
	}

	// validate the nodes
	nodeIDArray, err = GetClusterNodeIDs(dataFetcher)
	if err != nil {
		return err
	}

	// check the node id exists
	if !utils.IsValidInt(nodeIDServiceOperation) {
		return fmt.Errorf("invalid value for node id of %s", nodeIDServiceOperation)
	}

	if !utils.SliceContains(nodeIDArray, nodeIDServiceOperation) {
		return fmt.Errorf("no node with node id %s exists in this cluster", nodeIDServiceOperation)
	}

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to perform %s for service %s on node %s? (y/n) ",
		operation, serviceName, nodeIDServiceOperation)) {
		return nil
	}

	_, err = dataFetcher.InvokeServiceMemberOperation(serviceName, nodeIDServiceOperation, operation)
	if err != nil {
		return err
	}
	cmd.Println(OperationCompleted)

	return nil
}

// issueServiceCommand issues an operation against a service such as suspend or resume.
func issueServiceCommand(cmd *cobra.Command, serviceName, operation string) error {
	var (
		dataFetcher    fetcher.Fetcher
		connection     string
		servicesResult []string
		err            error
	)
	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	// get the services
	servicesResult, err = GetPersistenceServices(dataFetcher)
	if err != nil {
		return err
	}

	cmd.Println(FormatCurrentCluster(connection))

	// if a service was specified then validate
	if !utils.SliceContains(servicesResult, serviceName) {
		return fmt.Errorf("cannot find persistence service named %s", serviceName)
	}

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to perform %s for service %s? (y/n) ", operation, serviceName)) {
		return nil
	}

	_, err = dataFetcher.InvokeServiceOperation(serviceName, operation)
	if err != nil {
		return err
	}
	cmd.Println(OperationCompleted)

	return nil
}

// isStatusHASaferThan returns true if the statusHaValue is safer that the safestStatusHAValue.
func isStatusHASaferThan(statusHAValue, safestStatusHAValue string) bool {
	thisIndex := utils.GetSliceIndex(allStatusHA, statusHAValue)
	safestIndex := utils.GetSliceIndex(allStatusHA, safestStatusHAValue)
	return thisIndex >= safestIndex
}

// DeduplicateServices removes duplicated service details.
func DeduplicateServices(servicesSummary config.ServicesSummaries, serviceType string) []config.ServiceSummary {
	// the current results include 1 entry for each service and member, so we need to remove duplicates
	var finalServices = make([]config.ServiceSummary, 0)

	for _, value := range servicesSummary.Services {
		if serviceType != all && value.ServiceType != serviceType {
			continue
		}
		// check to see if this service and member already exists in the finalServices
		if len(finalServices) == 0 {
			// no entries so add it anyway
			finalServices = append(finalServices, value)
		} else {
			var found = false
			for _, v := range finalServices {
				if v.ServiceName == value.ServiceName {
					found = true
					break
				}
			}
			if !found {
				finalServices = append(finalServices, value)
			}
		}
	}
	return finalServices
}

// serviceExists returns true if the service exists in the services summary.
func serviceExists(serviceName string, servicesSummary config.ServicesSummaries) bool {
	for _, v := range servicesSummary.Services {
		if v.ServiceName == serviceName {
			return true
		}
	}

	return false
}

func init() {
	getServicesCmd.Flags().StringVarP(&serviceType, "type", "t", all,
		`service types to show. E.g. DistributedCache, FederatedCache, PagedTopic,
Invocation, Proxy, RemoteCache or ReplicatedCache`)
	getServicesCmd.Flags().StringVarP(&statusHAType, "status-ha", "a", "none",
		"statusHA to wait for. Used in conjunction with -T option")
	getServicesCmd.Flags().Int32VarP(&statusHATimeout, "timeout", "T", 60, "timeout to wait for StatusHA value of all services")

	setServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setServiceCmd.Flags().StringVarP(&attributeNameService, "attribute", "a", "", "attribute name to set")
	_ = setServiceCmd.MarkFlagRequired("attribute")
	setServiceCmd.Flags().StringVarP(&attributeValueService, "value", "v", "", "attribute value to set")
	_ = setServiceCmd.MarkFlagRequired("value")
	setServiceCmd.Flags().StringVarP(&nodeIDService, "node", "n", all, commaSeparatedIDMessage)

	suspendServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	resumeServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	stopServiceCmd.Flags().StringVarP(&nodeIDServiceOperation, "node", "n", "", "node id to target")
	_ = stopServiceCmd.MarkFlagRequired("node")
	stopServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	startServiceCmd.Flags().StringVarP(&nodeIDServiceOperation, "node", "n", "", "node id to target")
	_ = startServiceCmd.MarkFlagRequired("node")
	startServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	shutdownServiceCmd.Flags().StringVarP(&nodeIDServiceOperation, "node", "n", "", "node id to target")
	_ = shutdownServiceCmd.MarkFlagRequired("node")
	shutdownServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	getServiceMembersCmd.Flags().BoolVarP(&excludeStorageDisabled, "exclude", "x", false, "exclude storage-disabled clients")
	getServiceMembersCmd.Flags().BoolVarP(&includeBacklogOnly, "include", "B", false, "include members with backlog only")
}
