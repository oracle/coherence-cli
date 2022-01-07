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
	serviceType       string
	statusHATimeout   int32
	statusHAType      string
	validServiceTypes = []string{"DistributedCache", "FederatedCache", "Invocation", "Proxy", "RemoteCache", "ReplicatedCache", "all"}
	validStatusHA     = []string{"NODE-SAFE", "MACHINE-SAFE", "RACK-SAFE", "SITE-SAFE"}
	allStatusHA       = []string{"ENDANGERED", "NODE-SAFE", "MACHINE-SAFE", "RACK-SAFE", "SITE-SAFE"}

	attributeNameService   string
	attributeValueService  string
	validAttributesService = []string{"threadCount", "threadCountMin", "threadCountMax",
		"taskHungThresholdMillis", "requestTimeoutMillis"}
	nodeIDService          string
	nodeIDServiceOperation string
)

// getServicesCmd represents the get services command
var getServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "display services for a cluster",
	Long: `The 'get services' command displays services for a cluster using various options. 
You may specify the service type as well a status-ha value to wait for. You
can also specify '-o wide' to display addition information.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			servicesSummary = config.ServicesSummaries{}
			err             error
			statusHAValues  []string
			dataFetcher     fetcher.Fetcher
			connection      string
		)

		if statusHAType != "none" {
			if !watchEnabled {
				return errors.New("if you have specified a status-ha value then you must enable watch option")
			}
			if !utils.SliceContains(validStatusHA, statusHAType) {
				return fmt.Errorf("the list of Status-HA values must be one of %v", validStatusHA)
			}
		}

		// check service type
		if serviceType != "" && !utils.SliceContains(validServiceTypes, serviceType) {
			return fmt.Errorf("invalid service type of '%s' specified", serviceType)
		}

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		startTime := time.Now()

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

				deDuplicatedServices := DeduplicateServices(servicesSummary, serviceType)
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
			if !watchEnabled {
				break
			}

			// if we have specified a statusHA value to wait for then process this
			if statusHAType != "none" {
				elapsedSeconds := int32(time.Since(startTime).Seconds())
				if len(statusHAValues) == 1 && isStatusHASaferThan(statusHAValues[0], statusHAType) {
					cmd.Printf("Status HA value of %s or better reached in %d seconds for all service types of %s\n",
						statusHAType, elapsedSeconds, serviceType)
					return nil
				}

				if elapsedSeconds > statusHATimeout {
					return fmt.Errorf("status HA value of %s or better NOT reached in %d seconds for all service types od %s",
						statusHAType, elapsedSeconds, serviceType)
				}

				cmd.Printf("Waiting for Status HA value %s or better for all service type %s within %d seconds",
					statusHAType, serviceType, statusHATimeout)
			}

			// we are watching so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// describeService represents the describe service command
var describeServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "describe a service",
	Long: `The 'describe service' command shows information related to services. This
includes information about each service member as well as Persistence information if the
service is a cache service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
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

		found := serviceExists(serviceName, servicesSummary)

		if !found {
			return fmt.Errorf("unable to find service with service name '%s'", serviceName)
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

		if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
			finalResult, err := utils.CombineByteArraysForJSON([][]byte{serviceResult, proxyResults,
				membersResult, proxyResults, distributionsData},
				[]string{"services", "proxies", "members", "partitions", "distribution"})
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
				return utils.GetError("unable to unmarshall members result", err)
			}

			err = json.Unmarshal(membersResult, &persistenceDetails)
			if err != nil {
				return utils.GetError("unable to unmarshall members persistence result", err)
			}

			value = FormatServiceMembers(membersDetails.Services)

			sb.WriteString(value)

			err = json.Unmarshal(proxyResults, &proxiesSummary)
			if err != nil {
				return utils.GetError("unable to unmarshall proxy result", err)
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

				err = json.Unmarshal(coordData, &coordinator)
				if err != nil {
					return err
				}

				value, err = FormatJSONForDescribe(coordData, false,
					"Coordinator Id", "Idle", "Operation Status", "Snapshots")
				if err != nil {
					return err
				}
				sb.WriteString("\nPERSISTENCE COORDINATOR\n")
				sb.WriteString("-----------------------\n")
				sb.WriteString(value)

				value, err = FormatJSONForDescribe(distributionsData, true)
				if err != nil {
					return err
				}

				if value != "" {
					sb.WriteString("\nDISTRIBUTION INFORMATION\n")
					sb.WriteString("------------------------\n")
					sb.WriteString(value)
				}

				value, err = FormatJSONForDescribe(partitionsData, true,
					"Service", "Strategy Name")
				if err != nil {
					return err
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

// setServiceCmd represents the set service command
var setServiceCmd = &cobra.Command{
	Use:   "service <service-name>",
	Short: "set a service attribute across one or more members",
	Long: `The 'set service' command sets an attribute for a service across one or member nodes.
The following attribute names are allowed: threadCount, threadCountMin, threadCountMax or
taskHungThresholdMillis or requestTimeoutMillis.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
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
			nodeIds         []string
			confirmMessage  string
			response        string
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
			return fmt.Errorf("unable to find service with service name '%s'", serviceName)
		}

		// validate the nodes
		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		if nodeIDService == "all" {
			nodeIds = append(nodeIds, nodeIDArray...)
			confirmMessage = fmt.Sprintf("all %d nodes", len(nodeIds))
		} else {
			nodeIds = strings.Split(nodeIDService, ",")
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
			cmd.Printf("Selected service: %s\n", serviceName)
			cmd.Printf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
				attributeNameService, attributeValueService, confirmMessage)
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
		cmd.Println("operation completed")

		return nil
	},
}

// suspendServiceCmd represents the suspend service command
var suspendServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "suspend a service",
	Long:  `The 'suspend service' command suspends a specific service in all the members of a cluster.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceCommand(cmd, args[0], fetcher.SuspendService)
	},
}

// resumeServiceCmd represents the resume service command
var resumeServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "resume a service",
	Long:  `The 'resume service' command resumes a specific service in all the members of a cluster.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceCommand(cmd, args[0], fetcher.ResumeService)
	},
}

// stopServiceCmd represents the stop service command
var stopServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "stop a service",
	Long: `The 'stop service' command forces a specific service to stop on a cluster member.
Use the shutdown service command for normal service termination.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.StopService)
	},
}

// startServiceCmd represents the start service command
var startServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "start a service",
	Long:  `The 'start service' command starts a specific service on a cluster member.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.StartService)
	},
}

// shutdownServiceCmd represents the shutdown service command
var shutdownServiceCmd = &cobra.Command{
	Use:   "service service-name",
	Short: "shutdown a service",
	Long: `The 'shutdown service' command performs a controlled shut-down of a specific service
on a cluster member. Shutting down a service is preferred over stopping a service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return issueServiceNodeCommand(cmd, args[0], fetcher.ShutdownService)
	},
}

// issueServiceNodeCommand issues an operation against a service member
func issueServiceNodeCommand(cmd *cobra.Command, serviceName, operation string) error {
	var (
		dataFetcher     fetcher.Fetcher
		connection      string
		err             error
		response        string
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
		return fmt.Errorf("unable to find service with service name '%s'", serviceName)
	}

	// validate the nodes
	nodeIDArray, err = GetNodeIds(dataFetcher)
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

	// confirmation
	if !automaticallyConfirm {
		cmd.Printf("Are you sure you want to perform %s for service %s on node %s? (y/n) ",
			operation, serviceName, nodeIDServiceOperation)
		_, err = fmt.Scanln(&response)
		if response != "y" || err != nil {
			cmd.Println(constants.NoOperation)
			return nil
		}
	}

	_, err = dataFetcher.InvokeServiceMemberOperation(serviceName, nodeIDServiceOperation, operation)
	if err != nil {
		return err
	}
	cmd.Println("operation completed")

	return nil
}

// issueServiceCommand issues an operation against a service such as suspend or resume
func issueServiceCommand(cmd *cobra.Command, serviceName, operation string) error {
	var (
		dataFetcher    fetcher.Fetcher
		connection     string
		servicesResult []string
		err            error
		response       string
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

	// confirmation
	if !automaticallyConfirm {
		cmd.Printf("Are you sure you want to perform %s for service %s? (y/n) ", operation, serviceName)
		_, err = fmt.Scanln(&response)
		if response != "y" || err != nil {
			cmd.Println(constants.NoOperation)
			return nil
		}
	}

	_, err = dataFetcher.InvokeServiceOperation(serviceName, operation)
	if err != nil {
		return err
	}
	cmd.Println("operation completed")

	return nil
}

// isStatusHASaferThan returns true if the statusHaValue is safer that the safestStatusHAValue
func isStatusHASaferThan(statusHAValue, safestStatusHAValue string) bool {
	thisIndex := utils.GetSliceIndex(allStatusHA, statusHAValue)
	safestIndex := utils.GetSliceIndex(allStatusHA, safestStatusHAValue)
	return thisIndex >= safestIndex
}

// DeduplicateServices removes duplicated service details
func DeduplicateServices(servicesSummary config.ServicesSummaries, serviceType string) []config.ServiceSummary {
	// the current results include 1 entry for each service and member, so we need to remove duplicates
	var finalServices = make([]config.ServiceSummary, 0)

	for _, value := range servicesSummary.Services {
		if serviceType != "all" && value.ServiceType != serviceType {
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

// serviceExists returns true if the service exists in the services summary
func serviceExists(serviceName string, servicesSummary config.ServicesSummaries) bool {
	found := false
	for _, v := range servicesSummary.Services {
		if v.ServiceName == serviceName {
			found = true
			break
		}
	}

	return found
}

func init() {
	getServicesCmd.Flags().StringVarP(&serviceType, "type", "t", "all",
		`service types to show. E.g. DistributedCache, FederatedCache,
Invocation, Proxy, RemoteCache or ReplicatedCache`)
	getServicesCmd.Flags().StringVarP(&statusHAType, "status-ha", "a", "none",
		"statusHA to wait for. Used in conjunction with -T option")
	getServicesCmd.Flags().Int32VarP(&statusHATimeout, "timeout", "T", 60, "timeout to wait for StatusHA value of all services")

	setServiceCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setServiceCmd.Flags().StringVarP(&attributeNameService, "attribute", "a", "", "attribute name to set")
	_ = setServiceCmd.MarkFlagRequired("attribute")
	setServiceCmd.Flags().StringVarP(&attributeValueService, "value", "v", "", "attribute value to set")
	_ = setServiceCmd.MarkFlagRequired("value")
	setServiceCmd.Flags().StringVarP(&nodeIDService, "node", "n", "all", "comma separated node ids to target")

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

}
