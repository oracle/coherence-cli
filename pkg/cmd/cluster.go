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
	"github.com/oracle/coherence-cli/pkg/discovery"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	connectionURL  string
	connectionType string
	verboseOutput  bool
	ignoreErrors   bool
	timeout        int32
)

const clusterMessage = "A cluster connection already exists with the name %s for %s\n"
const ignoreErrorsMessage = "ignore errors from NS lookup"
const youMustProviderClusterMessage = "you must provide a cluster name"

// addClusterCmd represents the add cluster command
var addClusterCmd = &cobra.Command{
	Use:   "cluster connection-name",
	Short: "add a cluster connection",
	Long: `The 'add cluster' command adds a new connection to a Coherence cluster. You can
specify the full url such as https://<host>:<management-port>/management/coherence/cluster.
You can also specify host:port (for http connections) and the url will be automatically
populated constructed.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a connection name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		connection := sanitizeConnectionName(args[0])

		err := ensureUniqueCluster(connection)
		if err != nil {
			return err
		}

		if connectionURL == "" {
			return errors.New("you must supply a connection url")
		}

		return addCluster(cmd, connection, connectionURL, "manual", "")
	},
}

// removeClusterCmd represents the remove cluster command
var removeClusterCmd = &cobra.Command{
	Use:   "cluster connection-name",
	Short: "remove a cluster connection",
	Long:  `The 'remove cluster' command removes a cluster connection.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a connection name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusterName = args[0]
			dataFetcher fetcher.Fetcher
			err         error
		)

		found, cluster := GetClusterConnection(clusterName)
		if !found {
			return errors.New(UnableToFindClusterMsg + clusterName)
		}

		dataFetcher, err = GetDataFetcher(clusterName)
		if err != nil {
			return err
		}

		processCount := len(getRunningProcesses(dataFetcher))

		// only check for running members if the cluster was manually created
		if processCount > 0 && cluster.ManuallyCreated {
			return fmt.Errorf("cluster %s has %d processes running. You must stop the cluster before removing it", clusterName, processCount)
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to remove the connection to cluster %s? (y/n) ", clusterName)) {
			return nil
		}

		newConnection := make([]ClusterConnection, 0)
		for _, value := range Config.Clusters {
			if value.Name != clusterName {
				newConnection = append(newConnection, value)
			}
		}

		// replace the config with the new one
		Config.Clusters = newConnection

		viper.Set("clusters", Config.Clusters)
		err = WriteConfig()
		if err != nil {
			return err
		}

		cmd.Printf("Removed connection for cluster %s\n", clusterName)

		// if the cluster that was removed was in the current context, then reset it
		if Config.CurrentContext == clusterName {
			if err = clearContext(cmd); err != nil {
				return err
			}
		}

		return nil
	},
}

// getClustersCmd represents the get clusters command
var getClustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "display the list of discovered, manually added or created clusters",
	Long: `The 'get clusters' command displays the list of cluster connections.
The 'LOCAL' column is set to 'true' if the cluster has been created using the
'cohctl create cluster' command. You can also use the '-o wide' option to see if the
cluster is running.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err           error
			result        []byte
			jsonResult    []byte
			clusterResult []byte
			stringResult  string
			dataFetcher   fetcher.Fetcher
		)
		outputFormat, _ := cmd.Flags().GetString("output")

		err = checkOutputFormat()
		if err != nil {
			return err
		}

		var clusters = Config.Clusters
		if strings.Contains(outputFormat, constants.JSONPATH) {
			jsonResult, err = json.Marshal(clusters)
			if err != nil {
				return err
			}
			stringResult, err = utils.GetJSONPathResults(jsonResult, outputFormat)
			if err != nil {
				return err
			}
			cmd.Println(stringResult)
		} else if outputFormat == constants.JSON {
			result, err = json.Marshal(clusters)
			if err != nil {
				return utils.GetError("unable to unmarshall clusters", err)
			}
			cmd.Println(string(result))
		} else {
			if outputFormat == constants.WIDE {
				// go through each of the connections and see if the management URL responds with
				// a http 200, which would at least indicating the management node is up.
				// it is not a true test of if the cluster is actually fully functional, but just an indicator
				// Manually set request timeout for this operation, so we don't wait for too long if endpoints cannot be reached
				if Config.RequestTimeout > 5 {
					fetcher.RequestTimeout = 5
				}

				for i, v := range Config.Clusters {
					// see if we can contact the management node
					var running = false
					dataFetcher, err = GetDataFetcher(v.Name)
					if err == nil {
						var cluster = config.Cluster{}
						// must be a valid connection
						clusterResult, err = dataFetcher.GetClusterDetailsJSON()
						if err == nil && len(clusterResult) > 0 {
							// unmarshall and only set true if the cluster names match
							err = json.Unmarshal(clusterResult, &cluster)
							running = err == nil && cluster.ClusterName == v.ClusterName
						}
					}

					Config.Clusters[i].ManagementAvailable = running
				}
			}
			cmd.Println(FormatClusterConnections(clusters))
		}
		return nil
	},
}

// describeClusterCmd represents the describe cluster command
var describeClusterCmd = &cobra.Command{
	Use:   "cluster cluster-name",
	Short: "describe a cluster",
	Long: `The 'describe cluster' command shows cluster information related to a specific 
cluster connection, including: cluster overview, members, machines, services, caches, 
reporters, proxy servers and Http servers. You can specify '-o wide' to display 
addition information as well as '-v' to displayed additional information.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a cluster connection")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			cluster                    = config.Cluster{}
			members                    = config.Members{}
			services                   = config.ServicesSummaries{}
			proxiesSummary             = config.ProxiesSummary{}
			reporters                  = config.Reporters{}
			httpSessions               = config.HTTPSessionSummaries{}
			executors                  = config.Executors{}
			healthSummaries            = config.HealthSummaries{}
			machines                   []config.Machine
			finalSummariesDestinations []config.FederationSummary
			finalSummariesOrigins      []config.FederationSummary
			storage                    = config.StorageDetails{}
			dataFetcher                fetcher.Fetcher
			federatedServices          []string
			edData                     string
			wg                         sync.WaitGroup
			err                        error
			clusterResult              []byte
			membersResult              []byte
			servicesResult             []byte
			proxyResults               []byte
			reportersResult            []byte
			ramResult                  []byte
			flashResult                []byte
			cachesResult               []byte
			http                       []byte
			executorsResult            []byte
			machinesData               []byte
			storageData                []byte
			healthResult               []byte
			errorSink                  = createErrorSink()
			cachesData                 string
			topicsData                 string
			jsonPathOrJSON             = strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON
		)

		const waitGroupCount = 12

		connection := args[0]

		// do validation for OutputFormat
		err = checkOutputFormat()
		if err != nil {
			return err
		}

		dataFetcher, err = GetDataFetcher(connection)
		if err != nil {
			return err
		}

		// retrieve cluster details first so if we are connected
		// to WLS or need authentication, this can be done first
		clusterResult, err = dataFetcher.GetClusterDetailsJSON()
		if err != nil {
			return err
		}

		// retrieve the rest of the details for the cluster in parallel
		wg.Add(waitGroupCount)

		go func() {
			defer wg.Done()
			var err1 error
			membersResult, err1 = dataFetcher.GetMemberDetailsJSON(false)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			machinesMap, err1 := GetMachineList(dataFetcher)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}
			machines, err1 = getMachines(machinesMap, dataFetcher)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}

			if jsonPathOrJSON {
				machinesData, err = getOSJson(machinesMap, dataFetcher)
				if err1 != nil {
					errorSink.AppendError(err1)
				}
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			servicesResult, err1 = dataFetcher.GetServiceDetailsJSON()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			healthResult, err1 = dataFetcher.GetMembersHealth()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			storageData, err1 = dataFetcher.GetStorageDetailsJSON()
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
			if !verboseOutput {
				return
			}
			reportersResult, err1 = dataFetcher.GetReportersJSON()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			if !verboseOutput {
				return
			}
			reportersResult, err1 = dataFetcher.GetReportersJSON()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			federatedServices, err1 = GetFederatedServices(dataFetcher)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}
			finalSummariesDestinations, err1 = getFederationSummaries(federatedServices, outgoing, dataFetcher)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}
			finalSummariesOrigins, err = getFederationSummaries(federatedServices, incoming, dataFetcher)
			if err1 != nil {
				errorSink.AppendError(err1)
				return
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			flashResult, err1 = dataFetcher.GetElasticDataDetails("flash")
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			http, err1 = dataFetcher.GetHTTPSessionDetailsJSON()
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		go func() {
			defer wg.Done()
			var err1 error
			executors, err1 = getExecutorDetails(dataFetcher, true)
			if err1 != nil {
				errorSink.AppendError(err1)
			}
		}()

		// wait for all data fetchers requests to complete
		wg.Wait()

		errorList := errorSink.GetErrors()

		// check if any of the requests returned errors and only fail if all do
		errorCount := len(errorList)
		if errorCount == waitGroupCount {
			return utils.GetErrors(errorList)
		} else if errorCount != 0 {
			// one or more errors.
			err = utils.GetErrors(errorList)
			_, _ = fmt.Fprint(os.Stderr, err.Error())
		}

		if verboseOutput && len(executors.Executors) > 0 {
			executorsResult, err = json.Marshal(executors)
			if err != nil {
				return err
			}
		}

		if jsonPathOrJSON {
			cachesResult, err = dataFetcher.GetCachesSummaryJSONAllServices()
			if err != nil {
				return err
			}
			// build the final json data
			jsonDataDest, _ := json.Marshal(finalSummariesDestinations)
			jsonDataOrigins, _ := json.Marshal(finalSummariesOrigins)
			finalResult, err := utils.CombineByteArraysForJSON(
				[][]byte{clusterResult, machinesData, membersResult, servicesResult, cachesResult,
					proxyResults, reportersResult, ramResult, flashResult, http, executorsResult,
					jsonDataDest, jsonDataOrigins, healthResult},
				[]string{"cluster", "machines", "members", "services", "caches", "proxies", "reporters", constants.RAMJournal,
					constants.FlashJournal, "httpServers", "executors", "federationDestinations", "federationOrigins", "health"})
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
			// format the output for text
			err = json.Unmarshal(clusterResult, &cluster)
			if err != nil {
				return utils.GetError("unable to decode cluster details", err)
			}

			err = json.Unmarshal(membersResult, &members)
			if err != nil {
				return utils.GetError("unable to decode members result", err)
			}

			err = json.Unmarshal(servicesResult, &services)
			if err != nil {
				return utils.GetError("unable to decode services results", err)
			}

			err = json.Unmarshal(storageData, &storage)
			if err != nil {
				return utils.GetError("unable to decode storage details", err)
			}

			storageMap := utils.GetStorageMap(storage)

			if len(proxyResults) > 0 {
				err = json.Unmarshal(proxyResults, &proxiesSummary)
				if err != nil {
					return utils.GetError("unable to decode proxy details", err)
				}
			}

			if len(reportersResult) > 0 {
				err = json.Unmarshal(reportersResult, &reporters)
				if err != nil {
					return utils.GetError("unable to unmarshall reporter result", err)
				}
			}

			if len(http) > 0 {
				err = json.Unmarshal(http, &httpSessions)
				if err != nil {
					return utils.GetError("unable to decode Coherence*Web details", err)
				}
			}

			if len(healthResult) > 0 {
				err = json.Unmarshal(healthResult, &healthSummaries)
				if err != nil {
					return err
				}
			}

			var sb strings.Builder

			sb.WriteString("CLUSTER\n")
			sb.WriteString("-------\n")
			sb.WriteString(FormatCluster(cluster))

			sb.WriteString("\nMACHINES\n")
			sb.WriteString("--------\n")
			sb.WriteString(FormatMachines(machines))

			sb.WriteString("\nMEMBERS\n")
			sb.WriteString("-------\n")
			sb.WriteString(FormatMembers(members.Members, verboseOutput, storageMap))

			sb.WriteString("\nSERVICES\n")
			sb.WriteString("--------\n")
			sb.WriteString(FormatServices(DeduplicateServices(services, "all")))

			sb.WriteString("\nPERSISTENCE\n")
			sb.WriteString("-----------\n")
			deDuplicatedServices := DeduplicatePersistenceServices(services)

			err = processPersistenceServices(deDuplicatedServices, dataFetcher)
			if err != nil {
				return err
			}
			sb.WriteString(FormatPersistenceServices(deDuplicatedServices, true))

			if len(finalSummariesDestinations) > 0 || len(finalSummariesOrigins) > 0 {
				sb.WriteString("\nFEDERATION\n")
				sb.WriteString("----------\n")
				if len(finalSummariesDestinations) > 0 {
					sb.WriteString(FormatFederationSummary(finalSummariesDestinations, destinations))
				}
				if len(finalSummariesOrigins) > 0 {
					sb.WriteString(FormatFederationSummary(finalSummariesOrigins, origins))
				}
			}

			cacheServices := GetListOfCacheServices(services)

			// reset the error sink
			errorSink = createErrorSink()

			// carry out the caches and topics requests concurrently
			wg.Add(2)
			go func() {
				defer wg.Done()
				var err1 error
				cachesData, err1 = formatCachesSummary(cacheServices, dataFetcher)
				if err1 != nil {
					errorSink.AppendError(err1)
				}
				cachesData = "\nCACHES\n------\n" + cachesData
			}()

			go func() {
				defer wg.Done()
				var err1 error
				topicsData, err1 = formatTopicsSummary(cacheServices, dataFetcher)
				if err1 != nil {
					errorSink.AppendError(err1)
				}
				topicsData = "\nTOPICS\n------\n" + topicsData
			}()

			wg.Wait()
			errorList = errorSink.GetErrors()
			if len(errorList) > 0 {
				return utils.GetErrors(errorList)
			}

			sb.WriteString(cachesData + topicsData)

			if len(proxiesSummary.Proxies) > 0 {
				sb.WriteString("\nPROXY SERVERS\n")
				sb.WriteString("-------------\n")
				sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, "tcp"))
			}

			if len(proxiesSummary.Proxies) > 0 {
				sb.WriteString("\nHTTP SERVERS\n")
				sb.WriteString("------------\n")
				sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, "http"))
			}

			if verboseOutput {
				sb.WriteString("\nREPORTERS\n")
				sb.WriteString("---------\n")
				sb.WriteString(FormatReporters(reporters.Reporters))

				if len(executors.Executors) > 0 {
					sb.WriteString("\nEXECUTORS\n")
					sb.WriteString("---------\n")
					sb.WriteString(FormatExecutors(executors.Executors, true))
				}

				if len(healthSummaries.Summaries) > 0 {
					sb.WriteString("\nHEALTH\n")
					sb.WriteString("------\n")
					sb.WriteString(FormatMemberHealth(healthSummaries.Summaries))
				}
			}

			edData, err = getElasticDataResult(flashResult, ramResult)
			if err != nil {
				return err
			}
			if edData != "" {
				sb.WriteString("\nELASTIC DATA\n")
				sb.WriteString("------------\n")
				sb.WriteString(edData)
			}

			if len(httpSessions.HTTPSessions) > 0 {
				sb.WriteString("\nHTTP SESSION DETAILS\n")
				sb.WriteString("--------------------\n")
				sb.WriteString(FormatHTTPSessions(DeduplicateSessions(httpSessions), true))
			}

			cmd.Println(sb.String())

			if verboseOutput {
				cmd.Println("\nFLIGHT RECORDINGS")
				cmd.Println("-----------------")
				_ = executeJFROperation(cmd, "", fetcher.GetJFRs, dataFetcher, "")
			}
		}

		return nil
	},
}

// discoverClustersCmd represents the discover clusters command
var discoverClustersCmd = &cobra.Command{
	Use:   "clusters [host[:port]...]",
	Short: "discover clusters using the Coherence Name Service",
	Long: `The 'discover clusters' command discovers Coherence clusters using the Name Service.
You can specify a list of either host:port pairs or if you specify a host name the default cluster
port of 7574 will be used.
You will be presented with a list of clusters that have Management over REST configured and
you can confirm if you wish to add the discovered clusters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			count               = len(args)
			clustersWithoutHTTP = make([]int, 0)
			numEndpoints        int
			hostPorts           []string
			err                 error
			ns                  *discovery.NSLookup
			clusterPorts        []discovery.ClusterNSPort
			finalClusterPorts   = make([]discovery.ClusterNSPort, 0)
			discoveredClusters  = make([]discovery.DiscoveredCluster, 0)
			discoveredCluster   discovery.DiscoveredCluster
		)

		err = validateTimeout(timeout)
		if err != nil {
			return err
		}

		if count == 0 {
			hostPorts = []string{"localhost"}
		} else {
			hostPorts = args
		}

		cmd.Printf("Attempting to discover clusters using the following NameService addresses: %v\n", hostPorts)

		for _, address := range hostPorts {
			ns, err = discovery.Open(address, timeout)
			if err != nil {
				err = logErrorAndCheck(cmd, "unable to connect to "+address, err)
				if err != nil {
					return err
				}
				// skip to the next address
				closeSilent(ns)
				continue
			}

			// now ask the Name Service for local and remote clusters it knows about
			clusterPorts, err = ns.DiscoverNameServicePorts()
			err = logErrorAndCheck(cmd, "unable to discover clusters on "+address, err)
			if err != nil {
				return err
			}

			closeSilent(ns)
			finalClusterPorts = append(finalClusterPorts, clusterPorts...)
		}

		// close the original lookup - possible optimization here
		closeSilent(ns)

		numEndpoints = len(finalClusterPorts)
		if numEndpoints == 0 {
			return errors.New("no valid Name Service endpoints found")
		}

		// now query each individual NS host/ port and gather the cluster information
		for i, nsAddress := range finalClusterPorts {
			var (
				nsNew *discovery.NSLookup
			)
			addressPort := fmt.Sprintf("%s:%d", nsAddress.HostName, nsAddress.Port)
			cmd.Printf("Discovering Management URL for %s on %s ...\n", nsAddress.ClusterName, addressPort)
			nsNew, err = discovery.Open(addressPort, timeout)
			if err != nil {
				err = logErrorAndCheck(cmd, "unable to connect to "+addressPort, err)
				if err != nil {
					return err
				}
				// skip to the next address

				closeSilent(nsNew)
				continue
			}

			// discover the cluster information
			discoveredCluster, err = nsNew.DiscoverClusterInfo()
			if err != nil {
				err = logErrorAndCheck(cmd, "unable to get cluster information for to "+addressPort, err)
				if err != nil {
					return err
				}
				// skip to the next address
				closeSilent(nsNew)
				continue
			}

			// we discovered ok, so add to the list
			discoveredClusters = append(discoveredClusters, discoveredCluster)
			closeSilent(nsNew)
			if len(discoveredCluster.ManagementURLs) == 0 {
				clustersWithoutHTTP = append(clustersWithoutHTTP, i)
			}
		}

		var (
			totalClusters = len(discoveredClusters)
			withoutHTTP   = len(clustersWithoutHTTP)
			withHTTP      = totalClusters - withoutHTTP
			newConnection string
		)

		cmd.Printf("\nClusters found:    %d\nWithout Http Mgmt: %d\nWith Http Mgmt:    %d\n", totalClusters,
			withoutHTTP, withHTTP)

		if len(clustersWithoutHTTP) > 0 {
			cmd.Println("\nThe following clusters do not have Management over REST enabled and cannot be added:")
			// display the clusters without http
			for _, index := range clustersWithoutHTTP {
				cmd.Print("  " + formatCluster(discoveredClusters[index]))
			}
		}

		for i, cluster := range discoveredClusters {
			var (
				urls     = cluster.ManagementURLs
				urlsLen  = len(urls)
				selected int
			)
			if urlsLen == 0 {
				continue
			}

			if urlsLen == 1 {
				discoveredClusters[i].SelectedURL = urls[0]
			} else {
				cmd.Printf("\nCluster: %s, Name Service address: %s%d\n", cluster.ClusterName, cluster.Host, cluster.NSPort)

				header := "Urls: "
				for i, url := range urls {
					cmd.Printf("%s %3d - %s\n", header, i, url)
					if i == 0 {
						header = "      "
					}
				}
				selected, err = acceptIntegerValue(cmd, "Please enter the URL index to add: ", 0, urlsLen)
				if err != nil {
					return err
				}
				discoveredClusters[i].SelectedURL = urls[selected]
			}

			safeConnectionName := sanitizeConnectionName(cluster.ClusterName)

			// validate that the cluster connect name does not already exist as we will
			// try to add the cluster with the sane connection name as the cluster name
			found, conn := GetClusterConnection(safeConnectionName)
			if !found {
				discoveredClusters[i].ConnectionName = safeConnectionName
			} else {
				// cluster connection was found
				cmd.Printf(clusterMessage, safeConnectionName, conn.ConnectionURL)
				newConnection, err = acceptConnection(cmd, "Please enter a cluster name: ")
				newConnection = sanitizeConnectionName(newConnection)
				if len(newConnection) == 0 {
					return errors.New("invalid connection name")
				}
				if err != nil {
					return err
				}
				discoveredClusters[i].ConnectionName = newConnection
			}
		}

		cmd.Println()
		cmd.Println(FormatDiscoveredClusters(discoveredClusters))

		if withHTTP == 0 {
			return errors.New("no clusters have Management over REST enabled")
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to add the above %d cluster(s)? (y/n) ", withHTTP)) {
			return nil
		}

		// add the clusters
		for _, cluster := range discoveredClusters {
			if cluster.SelectedURL != "" {
				nsAddress := fmt.Sprintf("%s:%d", cluster.Host, cluster.NSPort)
				err = addCluster(cmd, cluster.ConnectionName, cluster.SelectedURL, "nslookup", nsAddress)
				err = logErrorAndCheck(cmd, "unable to discover cluster "+cluster.ConnectionName, err)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}

// addCluster adds a new cluster
func addCluster(cmd *cobra.Command, connection, connectionURL, discoveryType, nsAddress string) error {
	// check to see if the url is just host:port and then build the full management URL using http as default
	// otherwise let it fall through and get validated
	if !strings.Contains(connectionURL, "http") {
		split := strings.Split(connectionURL, ":")
		if len(split) == 2 {
			// candidate, second value must be int
			if utils.IsValidInt(split[1]) {
				connectionURL = fmt.Sprintf("http://%s:%s/management/coherence/cluster", split[0], split[1])
			}
		}
	}

	isWebLogic := fetcher.IsWebLogicServer(connectionURL)

	dataFetcher, err := fetcher.GetFetcherOrError(connectionType, connectionURL, Username, "")
	if err != nil {
		return err
	}
	clusterResult, err := dataFetcher.GetClusterDetailsJSON()
	if err != nil {
		var sb strings.Builder
		sb.WriteString("Unable to query cluster connection. " + err.Error() + "\n")
		sb.WriteString("Urls must be in the following format\n")
		sb.WriteString(" - Standalone: http[s]://<host>:<management-port>/management/coherence/cluster\n")
		sb.WriteString(" - WebLogic: http[s]://<admin-host>:<admin-port>/management/coherence/latest/clusters\n")
		return utils.GetError(sb.String(), err)
	}
	cluster := config.Cluster{}
	err = json.Unmarshal(clusterResult, &cluster)
	if err != nil {
		return utils.GetError("unable to decode cluster details", err)
	}

	clusterType := "Standalone"
	if isWebLogic {
		clusterType = "WebLogic"
	}

	// add the new cluster
	newCluster := ClusterConnection{Name: connection, ConnectionType: connectionType, ConnectionURL: connectionURL,
		DiscoveryType: discoveryType, ClusterVersion: cluster.Version, ClusterName: cluster.ClusterName,
		ClusterType: clusterType, NameServiceDiscovery: nsAddress}

	Config.Clusters = append(Config.Clusters, newCluster)

	viper.Set(clusterKey, Config.Clusters)
	err = WriteConfig()
	if err != nil {
		return err
	}

	cmd.Printf("Added cluster %s with type %s and URL %s\n", connection, connectionType, connectionURL)
	return nil
}

// variables specifically for create cluster
var (
	httpPortParam            int32
	clusterPortParam         int32
	wkaParam                 string
	clusterVersionParam      string
	replicaCountParam        int32
	metricsStartPortParam    int32
	logLevelParam            int32
	heapMemoryParam          string
	useCommercialParam       bool
	extendClientParam        bool
	skipMavenDepsParam       bool
	validPersistenceModes    = []string{"on-demand", "active", "active-backup", "active-async"}
	persistenceModeParam     string
	serverStartClassParam    string
	startupDelayParam        string
	additionalArtifactsParam string
	profileValueParam        string
	fileNameParam            string
	statementParam           string
)

const defaultCoherenceVersion = "22.06.2"
const startClusterCommand = "start cluster"
const scaleClusterCommand = "scale cluster"
const stopClusterCommand = "stop cluster"
const defaultHeap = "128m"
const localHost = "127.0.0.1"

// createClusterCmd represents the create cluster command
var createClusterCmd = &cobra.Command{
	Use:   "cluster cluster-name",
	Short: "create a local Coherence cluster",
	Long: `The 'create cluster' command creates a local cluster, adds to the cohctl.yaml file 
and starts it. You must have the 'mvn' executable and 'java' 17+ executable in your PATH for 
this to work. This cluster is only for development/testing purposes and should not be used, 
and is not supported in a production capacity. Supported versions are: CE 22.06 and above and 
commercial 14.1.1.2206.1 and above. Default version is currently CE 22.09.
NOTE: This is an experimental feature and my be altered or removed in the future.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusterName    = sanitizeConnectionName(args[0])
			err            error
			groupID        = getCoherenceGroupID()
			cpEntry        string
			splitArtifacts []string
		)

		// validate the Java and Maven/Gradle executable are present and in the path
		if err = checkRuntimeRequirements(); err != nil {
			return err
		}

		if err = checkDepsRequirements(); err != nil {
			return err
		}

		// validate persistence mode
		if !utils.SliceContains(validPersistenceModes, persistenceModeParam) {
			return fmt.Errorf("invalid persistence mode %s. Must be one of %v", persistenceModeParam, validPersistenceModes)
		}

		// validate log level
		if err = validateLogLevel(logLevelParam); err != nil {
			return err
		}

		// validate http port
		if err = utils.ValidatePort(httpPortParam); err != nil {
			return err
		}

		// validate cluster port
		if err = utils.ValidatePort(clusterPortParam); err != nil {
			return err
		}

		// validate metrics port
		if metricsStartPortParam > 0 {
			if err = utils.ValidatePort(metricsStartPortParam); err != nil {
				return err
			}
		}

		// validate the server start class
		if err = utils.ValidateStartClass(serverStartClassParam); err != nil {
			return err
		}

		// ensure the http port is not already used by a running cluster
		if isPortUsed(httpPortParam) {
			return fmt.Errorf("the management port %d is already used, please choose another", httpPortParam)
		}

		// validate startup delay
		_, err = utils.GetStartupDelayInMillis(startupDelayParam)
		if err != nil {
			return err
		}

		// validate any additional artifacts
		if additionalArtifactsParam != "" {
			splitArtifacts = strings.Split(additionalArtifactsParam, ",")
			for _, v := range splitArtifacts {
				if strings.Contains(v, ":") {
					// could be G:A:V format so validate
					if len(strings.Split(v, ":")) != 3 {
						return fmt.Errorf("invalid G:A:V format for other artifact specified: %s", v)
					}
				} else if !utils.SliceContains(validCoherenceArtifacts, v) {
					return fmt.Errorf("invalid additional artifact specified: %s.\nValid values are: %v", v, validCoherenceArtifacts)
				}
			}
		}

		if replicaCountParam < 1 {
			return errors.New("replica count must be 1 or more")
		}

		// validate ensure unique cluster name
		if err = ensureUniqueCluster(clusterName); err != nil {
			return err
		}

		heap := Config.DefaultHeap
		if heap == "" {
			heap = heapMemoryParam
		}

		// validate profile
		if err = validateProfile(); err != nil {
			return err
		}

		cmd.Printf("\nCluster name:         %s\n", clusterName)
		cmd.Printf("Cluster version:      %s\n", clusterVersionParam)
		cmd.Printf("Cluster port:         %d\n", clusterPortParam)
		cmd.Printf("Management port:      %d\n", httpPortParam)
		cmd.Printf("Replica count:        %d\n", replicaCountParam)
		cmd.Printf("Initial memory:       %s\n", heap)
		cmd.Printf("Persistence mode:     %s\n", persistenceModeParam)
		cmd.Printf("Group ID:             %s\n", groupID)
		cmd.Printf("Additional artifacts: %v\n", additionalArtifactsParam)
		cmd.Printf("Startup Profile:      %v\n", profileValueParam)
		cmd.Printf("Dependency Tool:      %v\n", getExecType())

		// confirm the operation
		if !confirmOperation(cmd, "Are you sure you want to create the cluster with the above details? (y/n) ") {
			return nil
		}

		// update default jars based up coherence group and version
		updateDefaultJars()

		// update default jars with additional artifacts
		if len(splitArtifacts) > 0 {
			for _, v := range splitArtifacts {
				if strings.Contains(v, ":") {
					// G:A:V format
					gav := strings.Split(v, ":")
					defaultJars = append(defaultJars, &config.DefaultDependency{GroupID: gav[0], Artifact: gav[1], Version: gav[2], IsCoherence: false})
				} else {
					defaultJars = append(defaultJars, &config.DefaultDependency{GroupID: groupID, Artifact: v, IsCoherence: true, Version: clusterVersionParam})
				}
			}
		}

		// sort the defaultJars dependencies
		sort.Slice(defaultJars, func(p, q int) bool {
			if defaultJars[p].GroupID == defaultJars[q].GroupID {
				return strings.Compare(defaultJars[p].Artifact, defaultJars[q].Artifact) < 0
			}
			return strings.Compare(defaultJars[p].GroupID, defaultJars[q].GroupID) < 0
		})

		classpath := make([]string, 0)

		if Config.UseGradle {
			cmd.Printf("\nUsing gradle to generate classpath for %d dependencies...\n", len(defaultJars))
			for _, entry := range defaultJars {
				cmd.Printf("- %s:%s:%s\n", entry.GroupID, entry.Artifact, entry.Version)
			}
			classpath, err = buildGradleClasspath()
			if err != nil {
				return err
			}
		} else {
			// use maven dependencies
			if skipMavenDepsParam {
				cmd.Println("\nSkipping downloading Maven artifacts")
			} else {
				cmd.Printf("\nChecking %d Maven dependencies...\n", len(defaultJars))

				// download the coherence dependencies
				if err = getCoherenceMavenDependencies(cmd); err != nil {
					return fmt.Errorf("unable to get some depdencies: %v", err)
				}
			}

			// generate classpath
			for _, entry := range defaultJars {
				// get the maven repository classpath for the jar
				cpEntry, err = getMavenClasspath(entry.GroupID, entry.Artifact, entry.Version, fileTypeJar)

				if err != nil {
					return err
				}
				classpath = append(classpath, cpEntry)

				// get transitive deps
				if entry.Artifact != "jline" && entry.Artifact != "coherence" {
					// if we have specified to get transitive dependencies, then we need to use the downloaded pom
					// file for the dependency and get the classpath. Ignore coherence and jline as this will
					// bring in many dependencies due to me not uet figuring out how to not bring in optional deps
					cpEntry, err = getTransitiveClasspath(entry.GroupID, entry.Artifact, entry.Version)

					if err != nil {
						return err
					}
					classpath = append(classpath, cpEntry)
				}
			}
		}

		// generate startup arguments
		arguments := fmt.Sprintf("-Dcoherence.cluster=%s -Dcoherence.clusterport=%d -Dcoherence.ttl=0 -Dcoherence.wka=%s -Djava.net.preferIPv4Stack=true",
			clusterName, clusterPortParam, wkaParam)

		// add the new cluster
		newCluster := ClusterConnection{Name: clusterName, ConnectionType: "http",
			ConnectionURL:   fmt.Sprintf("http://localhost:%d/management/coherence/cluster", httpPortParam),
			ManuallyCreated: true, ClusterVersion: clusterVersionParam, ClusterName: clusterName,
			ClusterType: "Standalone", BaseClasspath: strings.Join(classpath, getClasspathSeparator()),
			Arguments: arguments, ManagementPort: httpPortParam, PersistenceMode: persistenceModeParam}

		cmd.Printf("Starting %d cluster members for cluster %s\n", replicaCountParam, clusterName)

		err = startCluster(cmd, newCluster, replicaCountParam, 0)

		if err != nil {
			return err
		}

		Config.Clusters = append(Config.Clusters, newCluster)

		viper.Set(clusterKey, Config.Clusters)
		if err = WriteConfig(); err != nil {
			return err
		}

		if err = setContext(cmd, clusterName); err != nil {
			return err
		}

		cmd.Println("Cluster added and started")

		return nil
	},
}

// startClusterCmd represents the start cluster command
var startClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "start a local Coherence cluster",
	Long:  `The 'start cluster' command starts a cluster that was manually created.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := utils.ValidateStartClass(serverStartClassParam); err != nil {
			return err
		}
		return runClusterOperation(cmd, args[0], startClusterCommand)
	},
}

// scaleClusterCmd represents the start cluster command
var scaleClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "scales a local Coherence cluster",
	Long:  `The 'scale cluster' command scales a cluster that was manually created.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], scaleClusterCommand)
	},
}

// stopClusterCmd represents the stop cluster command
var stopClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "stop a local Coherence cluster",
	Long:  `The 'stop cluster' command stops a cluster that was manually created or started.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], stopClusterCommand)
	},
}

// startConsoleCmd represents the start console command
var startConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "start a console client and connect to a local Coherence cluster",
	Long: `The 'start console' command starts a console client which connects to a
cluster using the current context or a cluster specified by using '-c'.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, consoleClass)
	},
}

// startCohQL represents the start cohql command
var startCohQLCmd = &cobra.Command{
	Use:   "cohql",
	Short: "start a CohQL client and connect to a local Coherence cluster",
	Long: `The 'start cohql' command starts a CohQL client which connects to a
cluster using the current context or a cluster specified by using '-c'..`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, cohQLClass)
	},
}

// startClassCmd represents the start class command
var startClassCmd = &cobra.Command{
	Use:   "class",
	Short: "start a specific Java class and connect to a local Coherence cluster",
	Long: `The 'start class' command starts a specific Java class which connects to a
cluster using the current context or a cluster specified by using '-c'.
The class name must include the full package and class name and must be included in
an artefact included in the initial cluster creation.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a class to run")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, args[0])
	},
}

func runStartClientOperation(cmd *cobra.Command, class string) error {
	var (
		err        error
		connection string
	)

	// validate log level
	if err = validateLogLevel(logLevelParam); err != nil {
		return err
	}

	// validate profile
	if err = validateProfile(); err != nil {
		return err
	}

	connection, _, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	_, clusterConn := getConnection(connection)

	if err = checkOperation(clusterConn, class); err != nil {
		return err
	}

	cmd.Println(FormatCurrentCluster(connection))

	return startClient(cmd, clusterConn, class)
}

func runClusterOperation(cmd *cobra.Command, connectionName, operation string) error {
	var (
		err            error
		proc           *os.Process
		processIDs     []int
		serverDelta    int32
		scaleType      string
		dataFetcher    fetcher.Fetcher
		confirmMessage string
		cluster        = config.Cluster{}
		clusterResult  []byte
	)

	// validate the Java executable are present and in the path
	if err = checkRuntimeRequirements(); err != nil {
		return err
	}

	// validate metrics port
	if metricsStartPortParam > 0 {
		if err = utils.ValidatePort(metricsStartPortParam); err != nil {
			return err
		}
	}

	// validate profile
	if err = validateProfile(); err != nil {
		return err
	}

	found, connection := GetClusterConnection(connectionName)
	if !found {
		return errors.New(UnableToFindClusterMsg + connectionName)
	}

	if err = checkOperation(connection, operation); err != nil {
		return err
	}

	dataFetcher, err = GetDataFetcher(connectionName)
	if err != nil {
		return err
	}

	// retrieve the slice of running PIDS
	processIDs = getRunningProcesses(dataFetcher)

	numProcesses := int32(len(processIDs))

	if (operation == stopClusterCommand || operation == scaleClusterCommand) && numProcesses == 0 {
		return fmt.Errorf("the cluster %s does not appear to be started", connection.Name)
	}
	if operation == startClusterCommand && numProcesses > 0 {
		// Retrieve the cluster details to see if the cluster name matches as
		// a different cluster could already be running on this http management port
		clusterResult, err = dataFetcher.GetClusterDetailsJSON()
		if err != nil {
			return err
		}

		err = json.Unmarshal(clusterResult, &cluster)
		if err != nil {
			return utils.GetError("unable to decode cluster details", err)
		}

		// We now have the cluster which is running based upon the management port
		if connection.ClusterName != cluster.ClusterName {
			return fmt.Errorf("A different cluster %s is running on this management port with process id: %v, please stop this cluster first", cluster.ClusterName, processIDs)
		}

		return fmt.Errorf("the cluster %s appears to be already started with process ids: %v", connection.Name, processIDs)
	}

	if operation == scaleClusterCommand {
		if replicaCountParam < 1 {
			return errors.New("replicas must be a positive value")
		} else if replicaCountParam == numProcesses {
			return fmt.Errorf("the cluster already running %d members. Please privde a different replica value", numProcesses)
		}
		if replicaCountParam <= numProcesses {
			return errors.New("scaling down a cluster is not yet supported")
		}
		serverDelta = replicaCountParam - numProcesses
		if serverDelta > 0 {
			scaleType = "up"
		} else {
			scaleType = "down"
		}

		confirmMessage = ""
		cmd.Printf("Scaling the cluster %s %s by %d member(s) to %d members\n", connection.Name, scaleType, serverDelta, replicaCountParam)
		replicaCountParam = serverDelta
	} else if operation == startClusterCommand {
		if replicaCountParam < 1 {
			return errors.New("replica count must be 1 or more")
		}

		confirmMessage = ""
	} else {
		confirmMessage = fmt.Sprintf("Are you sure you want to stop %d members for the cluster %s? (y/n) ", numProcesses, connection.Name)
	}

	// confirm the operation
	if confirmMessage != "" && !confirmOperation(cmd, confirmMessage) {
		return nil
	}

	if operation == stopClusterCommand {
		count := 0
		for _, v := range processIDs {
			proc, err = os.FindProcess(v)
			if err != nil {
				// silently ignore as it may have gone already
			} else {
				err = proc.Kill()
				if err != nil {
					// ignore as process may have exited
					cmd.Printf("unable to kill process %v\n", proc.Pid)
				} else {
					count++
					cmd.Printf("killed process %d\n", proc.Pid)
				}
			}
		}

		cmd.Printf("%d processes were stopped for cluster %s\n", count, connection.Name)
	} else {
		var message = "started"
		if operation == scaleClusterCommand {
			message = "scaled"
		}

		err = startCluster(cmd, connection, replicaCountParam, numProcesses)
		if err != nil {
			return err
		}

		cmd.Printf("Cluster %s %s\n", connection.Name, message)
	}

	return nil
}

func init() {
	addClusterCmd.Flags().StringVarP(&connectionURL, "url", "u", "", "connection URL")
	_ = addClusterCmd.MarkFlagRequired("url")
	addClusterCmd.Flags().StringVarP(&connectionType, "type", "t", "http", "connection type, http")

	describeClusterCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false,
		"include verbose output including individual members, reporters and executor details")

	discoverClustersCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignore", "I", false, ignoreErrorsMessage)
	discoverClustersCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	discoverClustersCmd.Flags().Int32VarP(&timeout, "timeout", "t", 30, timeoutMessage)

	removeClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	createClusterCmd.Flags().StringVarP(&clusterVersionParam, "version", "v", defaultCoherenceVersion, "cluster version")
	createClusterCmd.Flags().StringVarP(&persistenceModeParam, "persistence-mode", "s", "on-demand",
		fmt.Sprintf("persistence mode %v", validPersistenceModes))
	createClusterCmd.Flags().Int32VarP(&httpPortParam, "http-port", "H", 30000, "http management port")
	createClusterCmd.Flags().Int32VarP(&clusterPortParam, "cluster-port", "p", 7574, "cluster port")
	createClusterCmd.Flags().StringVarP(&wkaParam, "wka", "W", localHost, "well known address")
	createClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	createClusterCmd.Flags().StringVarP(&startupDelayParam, startupDelayArg, "D", "0ms", startupDelayMessage)
	createClusterCmd.Flags().Int32VarP(&replicaCountParam, "replicas", "r", 3, serverCountMessage)
	createClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	createClusterCmd.Flags().StringVarP(&additionalArtifactsParam, "additional", "a", "", "additional comma separated Coherence artifacts or others in G:A:V format")
	createClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	createClusterCmd.Flags().BoolVarP(&useCommercialParam, "commercial", "C", false, "use commercial Coherence groupID (default CE)")
	createClusterCmd.Flags().BoolVarP(&skipMavenDepsParam, "skip-deps", "K", false, "skip pulling artifacts")
	createClusterCmd.Flags().Int32VarP(&metricsStartPortParam, metricsPortArg, "t", 0, metricsPortMessage)
	createClusterCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)
	createClusterCmd.Flags().StringVarP(&serverStartClassParam, startClassArg, "S", "", startClassMessage)

	stopClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	startClusterCmd.Flags().Int32VarP(&replicaCountParam, "replicas", "r", 3, serverCountMessage)
	startClusterCmd.Flags().Int32VarP(&metricsStartPortParam, metricsPortArg, "t", 0, metricsPortMessage)
	startClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startClusterCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)
	startClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startClusterCmd.Flags().StringVarP(&startupDelayParam, startupDelayArg, "D", "0ms", startupDelayMessage)
	startClusterCmd.Flags().StringVarP(&serverStartClassParam, startClassArg, "S", "", startClassMessage)

	startConsoleCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startConsoleCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startConsoleCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	startCohQLCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startCohQLCmd.Flags().StringVarP(&fileNameParam, "file", "f", "", "file name to read CohQL commands from")
	startCohQLCmd.Flags().StringVarP(&statementParam, "statement", "S", "", "statement to execute enclosed in double quotes")
	startCohQLCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startCohQLCmd.Flags().BoolVarP(&extendClientParam, "extend", "X", false, "start CohQL as Extend client. Only works for default cache config")
	startCohQLCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	startClassCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startClassCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startClassCmd.Flags().BoolVarP(&extendClientParam, "extend", "X", false, "start a class as Extend client. Only works for default cache config")
	startClassCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	scaleClusterCmd.Flags().Int32VarP(&replicaCountParam, "replicas", "r", 3, serverCountMessage)
	scaleClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	scaleClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	scaleClusterCmd.Flags().StringVarP(&startupDelayParam, startupDelayArg, "D", "0ms", startupDelayMessage)
	scaleClusterCmd.Flags().Int32VarP(&metricsStartPortParam, metricsPortArg, "t", 0, metricsPortMessage)
	scaleClusterCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)
	scaleClusterCmd.Flags().StringVarP(&serverStartClassParam, startClassArg, "S", "", startClassMessage)
}

// sanitizeConnectionName sanitizes a cluster connection
func sanitizeConnectionName(connectionName string) string {
	return replaceAll(connectionName, "$", ",", " ", "'", "\"", "(", ")", "[", "]", "\\", "*",
		"%", "^", "&", "#", "/", "@", ";", "!")
}

// replaceAll replaces all the specified values with ""
func replaceAll(connectionName string, values ...string) string {
	result := connectionName
	for _, v := range values {
		result = strings.ReplaceAll(result, v, "")
	}
	return result
}

// closeSilent closes a NsLookup connection and ignores if it is nil
func closeSilent(ns *discovery.NSLookup) {
	if ns != nil {
		_ = ns.Close()
	}
}

// formatCluster formats a cluster
func formatCluster(cluster discovery.DiscoveredCluster) string {
	return fmt.Sprintf("Cluster: %s, Name Service address: %s:%d\n", cluster.ClusterName, cluster.Host, cluster.NSPort)
}

// acceptConnection accepts a connection name
func acceptConnection(cmd *cobra.Command, message string) (string, error) {
	var (
		response string
		err      error
	)
	for {
		cmd.Print(message)
		_, err = fmt.Scanln(&response)
		if err != nil || response == "" {
			cmd.Println("Please enter a connection name")
		} else {
			found, conn := GetClusterConnection(response)
			if found {
				cmd.Printf(clusterMessage, conn.ClusterName, conn.ConnectionURL)
			} else {
				return response, nil
			}
		}
	}
}

// acceptIntegerValue accepts and integer value in the range specified
func acceptIntegerValue(cmd *cobra.Command, message string, min, max int) (int, error) {
	var (
		response string
		err      error
		value    int
	)
	for {
		cmd.Print(message)
		_, err = fmt.Scanln(&response)
		if err != nil {
			return 0, err
		}
		value, err = strconv.Atoi(response)
		if err != nil || value < min || value > max {
			cmd.Printf("Please enter a value between %d and %d\n", min, max)
		} else {
			return value, nil
		}
	}
}

// logErrorAndCheck will log the error to the log file and check if
// ignore errors is selected and will return nil which means continue
func logErrorAndCheck(cmd *cobra.Command, message string, err error) error {
	if err == nil {
		return nil
	}
	actualError := utils.GetError(message, err)
	if !ignoreErrors {
		return actualError
	}
	// log and continue
	cmd.Println(actualError)
	return nil
}

// validateTimeout validates the NS Lookup timeout
func validateTimeout(timeout int32) error {
	if timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	return nil
}

// ensureUniqueCluster ensures the connection string is unique
func ensureUniqueCluster(connection string) error {
	found, clusterConn := GetClusterConnection(connection)
	if found {
		return fmt.Errorf("A connection for cluster named %s already exists with url=%s and type=%s",
			connection, clusterConn.ConnectionURL, clusterConn.ConnectionType)
	}

	return nil
}

// validateProfile validates the given profile param
func validateProfile() error {
	startupProfile := getProfileValue(profileValueParam)
	if profileValueParam != "" && startupProfile == "" {
		return fmt.Errorf("a profile with the name %s does not exist", profileValueParam)
	}

	return nil
}
