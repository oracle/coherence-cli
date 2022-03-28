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

// addClusterCmd represents the add cluster command
var addClusterCmd = &cobra.Command{
	Use:   "cluster connection-name",
	Short: "add a cluster connection",
	Long: `The 'add cluster' command adds a new connection to a Coherence cluster. You can
specify the full url such as https://<host>:<management-port>/management/coherence/cluster.
You can also specify host and port (for http connections) and the url will be automatically 
populated constructed.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a connection name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		connection := args[0]

		found, clusterConnection := GetClusterConnection(connection)
		if found {
			return fmt.Errorf("A connection for cluster named %s already exists with url=%s and type=%s",
				connection, clusterConnection.ConnectionURL, clusterConnection.ConnectionType)
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
		clusterName := args[0]

		found, _ := GetClusterConnection(clusterName)
		if !found {
			return errors.New(UnableToFindClusterMsg + clusterName)
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
			stringResult string
		)
		outputFormat, _ := cmd.Flags().GetString("output")

		err = checkOutputFormat()
		if err != nil {
			return err
		}

		var clusters = Config.Clusters
		if strings.Contains(outputFormat, constants.JSONPATH) {
			var jsonResult, err = json.Marshal(clusters)
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
addition information.`,
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
			errorSink                  = createErrorSink()
			cachesData                 string
			topicsData                 string
			jsonPathOrJSON             = strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON
		)

		const waitGroupCount = 11

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
				machinesData, err1 = getOSJson(machinesMap, dataFetcher)
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
			finalSummariesOrigins, err1 = getFederationSummaries(federatedServices, incoming, dataFetcher)
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

		// check if any of the requests returned errors and only fail if any do
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
					jsonDataDest, jsonDataOrigins},
				[]string{"cluster", "machines", "members", "services", "caches", "proxies", "reporters", constants.RAMJournal,
					constants.FlashJournal, "httpServers", "executors", "federationDestinations", "federationOrigins"})
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
				sb.WriteString("-------------\n")
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
	Use:   "clusters",
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
				if err != nil {
					return err
				}
				discoveredClusters[i].ConnectionName = sanitizeConnectionName(newConnection)
			}
		}

		cmd.Println()
		cmd.Println(FormatDiscoveredClusters(discoveredClusters))

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

func init() {
	addClusterCmd.Flags().StringVarP(&connectionURL, "url", "u", "", "connection URL")
	_ = addClusterCmd.MarkFlagRequired("url")
	addClusterCmd.Flags().StringVarP(&connectionType, "type", "t", "http", "connection type, http")

	describeClusterCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false,
		"include verbose output including individual members, reporters and executor details")

	discoverClustersCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignore", "I", false, ignoreErrorsMessage)
	discoverClustersCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	discoverClustersCmd.Flags().Int32VarP(&timeout, "timeout", "t", 30, timeoutMessage)
}

// sanitizeConnectionName sanitizes a cluster connection
func sanitizeConnectionName(connectionName string) string {
	return replaceAll(connectionName, "$", ",", " ", "'", "\"", "(", ")", "[", "]", "\\")
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
