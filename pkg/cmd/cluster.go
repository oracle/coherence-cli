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
	"strconv"
	"strings"
	"sync"
	"time"
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
			response    string
			clusterName = args[0]
		)

		found, connection := GetClusterConnection(clusterName)
		if !found {
			return errors.New(UnableToFindClusterMsg + clusterName)
		}

		processCount := len(connection.ProcessIDs)
		if processCount > 0 {
			return fmt.Errorf("cluster %s has %d processes running. You must stop the cluster before removing it", clusterName, processCount)
		}

		if !automaticallyConfirm {
			cmd.Printf("Are you sure you want to remove the connection to cluster %s? (y/n) ", clusterName)
			_, err := fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
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
		err := WriteConfig()
		if err != nil {
			return err
		}

		cmd.Printf("Removed connection for cluster %s\n", clusterName)

		return nil
	},
}

// getClustersCmd represents the get clusters command
var getClustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "display the list of discovered or manually added clusters",
	Long:  `The 'get clusters' command displays the list of cluster connections.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err          error
			result       []byte
			jsonResult   []byte
			stringResult string
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
			response            string
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

		if !automaticallyConfirm {
			cmd.Printf("Are you sure you want to add the above %d cluster(s)? (y/n) ", withHTTP)
			_, err = fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
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
	clusterVersionParam      string
	serverCountParam         int32
	logLevelParam            int32
	heapMemoryParam          string
	useCommercialParam       bool
	extendClientParam        bool
	skipMavenDepsParam       bool
	validPersistenceModes    = []string{"on-demand", "active", "active-backup", "active-async"}
	persistenceModeParam     string
	startupDelayParam        int32
	additionalArtifactsParam string
)

const defaultCoherenceVersion = "22.06.1"
const startClusterCommand = "start cluster"
const stopClusterCommand = "stop cluster"
const defaultHeap = "512m"

// createClusterCmd represents the create cluster command
var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "create a local Coherence cluster",
	Long: `The 'create cluster' command creates a local cluster, adds to the cohctl.yaml file 
and starts it. You must have the 'mvn' executable and 'java' 11+ executable in your PATH for 
this to work. This cluster is only for development/testing purposes and should not be used, 
and is not supported in a production capacity. Supported versions are: CE 22.06 and above and 
commercial 14.1.1.2206.1 and above.
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
			response       string
			processIDs     []string
			groupID        = getCoherenceGroupID()
			cpEntry        string
			splitArtifacts []string
		)

		// validate the Java and Maven executable are present and in the path
		if err = checkCreateRequirements(); err != nil {
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

		// validate port
		if err = utils.ValidatePort(httpPortParam); err != nil {
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

		if serverCountParam < 1 || serverCountParam > 20 {
			return errors.New("server count must be between 1 and 20")
		}

		// validate ensure
		err = ensureUniqueCluster(clusterName)
		if err != nil {
			return err
		}

		if !automaticallyConfirm {
			cmd.Printf("\nCluster name:         %s\n", clusterName)
			cmd.Printf("Cluster version:      %s\n", clusterVersionParam)
			cmd.Printf("Cluster port:         %d\n", clusterPortParam)
			cmd.Printf("Management port:      %d\n", httpPortParam)
			cmd.Printf("Server count:         %d\n", serverCountParam)
			cmd.Printf("Initial memory:       %s\n", heapMemoryParam)
			cmd.Printf("Persistence mode:     %s\n", persistenceModeParam)
			cmd.Printf("Group ID:             %s\n", groupID)
			cmd.Printf("Additional artifacts: %v\n", additionalArtifactsParam)
			cmd.Printf("Are you sure you want to create the cluster with the above details? (y/n) ")
			_, err = fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
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

		if skipMavenDepsParam {
			cmd.Println("\nSkipping downloading Maven artifcts")
		} else {
			cmd.Printf("\nChecking %d Maven dependencies...\n", len(defaultJars))

			// download the coherence dependencies
			if err = getCoherenceDependencies(cmd); err != nil {
				return fmt.Errorf("unable to get some depdencies: %v", err)
			}
		}

		// generate classpath
		classpath := make([]string, 0)
		for _, entry := range defaultJars {
			cpEntry, err = getMavenClasspath(entry.GroupID, entry.Artifact, entry.Version)
			if err != nil {
				return err
			}
			classpath = append(classpath, cpEntry)
		}

		// generate startup arguments
		arguments := fmt.Sprintf("-Dcoherence.cluster=%s -Dcoherence.clusterport=%d -Dcoherence.ttl=0 -Dcoherence.wka=127.0.0.1 -Djava.net.preferIPv4Stack=true",
			clusterName, clusterPortParam)

		// add the new cluster
		newCluster := ClusterConnection{Name: clusterName, ConnectionType: "http",
			ConnectionURL:   fmt.Sprintf("http://localhost:%d/management/coherence/cluster", httpPortParam),
			ManuallyCreated: true, ClusterVersion: clusterVersionParam, ClusterName: clusterName,
			ClusterType: "Standalone", BaseClasspath: strings.Join(classpath, getClasspathSeparator()),
			Arguments: arguments, ManagementPort: httpPortParam, PersistenceMode: persistenceModeParam}

		cmd.Printf("Starting %d cluster members for cluster %s\n", serverCountParam, clusterName)

		processIDs, err = startCluster(cmd, newCluster, serverCountParam)

		if err != nil {
			return err
		}

		newCluster.ProcessIDs = convertProcessIDs(processIDs)

		Config.Clusters = append(Config.Clusters, newCluster)

		viper.Set(clusterKey, Config.Clusters)
		err = WriteConfig()
		if err != nil {
			return err
		}

		cmd.Printf("Cluster added and started with process ids: %v\n", processIDs)

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
		return runClusterOperation(cmd, args[0], startClusterCommand)
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
	Short: "start a console client against the local Coherence cluster",
	Long:  `The 'start console' command starts a console client against a cluster that was manually created.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, consoleClass)
	},
}

// startCohQL represents the start cohql command
var startCohQLCmd = &cobra.Command{
	Use:   "cohql",
	Short: "start a CohQL client against the local Coherence cluster",
	Long:  `The 'start cohql' command starts a CohQL client against a cluster that was manually created.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, cohQLClass)
	},
}

// getProcsCmd represents the get procs command
var getProcsCmd = &cobra.Command{
	Use:   "procs",
	Short: "display processes started for a cluster",
	Long:  `The 'get procs' command displays processes started for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err           error
			connection    string
			jsonResult    []byte
			membersResult []byte
			dataFetcher   fetcher.Fetcher
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		_, clusterConnection := getConnection(connection)

		if err = checkOperation(clusterConnection, "get procs"); err != nil {
			return err
		}

		for {
			var members = config.Members{}

			// retrieve the members to see if we can match up to running process
			membersResult, _ = dataFetcher.GetMemberDetailsJSON(OutputFormat != constants.TABLE && OutputFormat != constants.WIDE)

			if len(membersResult) > 0 {
				err = json.Unmarshal(membersResult, &members)
				if err != nil {
					return utils.GetError(unableToDecode, err)
				}
			}

			var processes = getProcesses(clusterConnection.ProcessIDs, members)

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				jsonResult, err = json.Marshal(processes)
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(jsonResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(jsonResult))
				}
			} else {
				cmd.Println(FormatCurrentCluster(connection))

				cmd.Println(FormatProcesses(processes.ProcessList))
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

func runStartClientOperation(cmd *cobra.Command, class string) error {
	var (
		err        error
		connection string
	)

	// validate log level
	if err = validateLogLevel(logLevelParam); err != nil {
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
		err        error
		response   string
		proc       *os.Process
		processIDs []string
	)

	// validate the Java and Maven executable are present and in the path
	if err = checkCreateRequirements(); err != nil {
		return err
	}

	found, connection := GetClusterConnection(connectionName)
	if !found {
		return errors.New(UnableToFindClusterMsg + connectionName)
	}

	if err = checkOperation(connection, operation); err != nil {
		return err
	}

	numProcesses := len(connection.ProcessIDs)

	if operation == stopClusterCommand && numProcesses == 0 {
		return fmt.Errorf("the cluster %s does not appear to be started", connection.Name)
	}
	if operation == startClusterCommand && numProcesses > 0 {
		return fmt.Errorf("the cluster %s appears to be already started with process ids: %v", connection.Name, connection.ProcessIDs)
	}

	if !automaticallyConfirm {
		if operation == startClusterCommand {
			cmd.Printf("Are you sure you want to start %d members for cluster %s? (y/n) ", serverCountParam, connection.Name)
		} else {
			cmd.Printf("Are you sure you want to stop %d members for the cluster %s? (y/n) ", numProcesses, connection.Name)
		}

		_, err = fmt.Scanln(&response)
		if response != "y" || err != nil {
			cmd.Println(constants.NoOperation)
			return nil
		}
	}

	if operation == stopClusterCommand {
		count := 0
		for _, v := range connection.ProcessIDs {
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

		// reset the process Ids
		if err = updateConnectionPIDS(connection.Name, make([]int, 0)); err != nil {
			return err
		}

		cmd.Printf("%d processes were stopped for cluster %s\n", count, connection.Name)
	} else {
		processIDs, err = startCluster(cmd, connection, serverCountParam)
		if err != nil {
			return err
		}

		ProcessIDs := convertProcessIDs(processIDs)
		cmd.Printf("Cluster %s and started with process ids: %v\n", connection.Name, ProcessIDs)

		if err = updateConnectionPIDS(connection.Name, ProcessIDs); err != nil {
			return err
		}
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
	createClusterCmd.Flags().StringVarP(&persistenceModeParam, "persistence-mode", "P", "on-demand",
		fmt.Sprintf("persistence mode %v", validPersistenceModes))
	createClusterCmd.Flags().Int32VarP(&httpPortParam, "http-port", "H", 30000, "http management port")
	createClusterCmd.Flags().Int32VarP(&clusterPortParam, "cluster-port", "p", 7574, "cluster port")
	createClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	createClusterCmd.Flags().Int32VarP(&startupDelayParam, startupDelayArg, "D", 1, startupDelayMessage)
	createClusterCmd.Flags().Int32VarP(&serverCountParam, "server-count", "s", 3, serverCountMessage)
	createClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	createClusterCmd.Flags().StringVarP(&additionalArtifactsParam, "additional", "a", "", "additional comma separated Coherence artifacts or others in G:A:V format")
	createClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	createClusterCmd.Flags().BoolVarP(&useCommercialParam, "commercial", "C", false, "use commercial Coherence groupID (default is CE)")
	createClusterCmd.Flags().BoolVarP(&skipMavenDepsParam, "skip-deps", "S", false, "skip pulling Maven artifacts")

	stopClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	startClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	startClusterCmd.Flags().Int32VarP(&serverCountParam, "server-count", "s", 3, serverCountMessage)
	startClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startClusterCmd.Flags().Int32VarP(&startupDelayParam, startupDelayArg, "D", 1, startupDelayMessage)

	startConsoleCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startConsoleCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)

	startCohQLCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startCohQLCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startCohQLCmd.Flags().BoolVarP(&extendClientParam, "extend", "X", false, "start CohQL as Extend client. Only works for default cache config")
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
