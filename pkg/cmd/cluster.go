/*
 * Copyright (c) 2021, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/oracle/coherence-go-client/v2/coherence/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	connectionURL  string
	connectionType string
	verboseOutput  bool
	ignoreErrors   bool
	timeout        int32

	validSetClusterAttributes = []string{
		"bufferPublishSize", "bufferReceiveSize", "loggingLevel", "loggingLimit", "loggingFormat",
		"multicastThreshold", "resendDelay", "sendAckDelay", "trafficJamCount", "trafficJamDelay",
	}

	attributeNameCluster  string
	attributeValueCluster string
)

const (
	ignoreCertsFlag                  = "ignore-certs"
	clusterMessage                   = "A cluster connection already exists with the name %s for %s\n"
	ignoreErrorsMessage              = "ignore errors from NS lookup"
	youMustProviderConnectionMessage = "you must provide a single connection name"
	httpType                         = "http"
)

// addClusterCmd represents the add cluster command
var addClusterCmd = &cobra.Command{
	Use:   "cluster connection-name",
	Short: "add a cluster connection",
	Long: `The 'add cluster' command adds a new connection to a Coherence cluster. You can
specify the full url such as https://<host>:<management-port>/management/coherence/cluster.
You can also specify host:port (for http connections) and the url will be automatically
constructed. Use --ignore-certs only to allow invalid certificates for this
saved HTTPS connection's origin, including the initial probe. Verification is enabled
by default. Prefer a valid matching certificate and COHERENCE_TLS_CERTS_PATH for
private CA trust. Trusting a CA cannot fix expiry or a hostname mismatch.
Downloads, discovery and other-origin destinations remain verified. The exception
warns on stderr before its first use per connection/origin in each execution.`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New(youMustProviderConnectionMessage)
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
			return errors.New("you must provide a connection url")
		}

		ignoreInvalidCerts, err := cmd.Flags().GetBool(ignoreCertsFlag)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed(ignoreCertsFlag) && !fetcher.IsValidHTTPSURL(connectionURL) {
			return errors.New("--ignore-certs requires a valid HTTPS connection URL")
		}
		if err = addCluster(cmd, connection, connectionURL, "manual", "", ignoreInvalidCerts); err != nil {
			return err
		}

		return setContext(cmd, connection)
	},
}

// removeClusterCmd represents the remove cluster command
var removeClusterCmd = &cobra.Command{
	Use:               "cluster connection-name",
	Short:             "remove a cluster connection",
	Long:              `The 'remove cluster' command removes a cluster connection.`,
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
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

		if cluster.ManuallyCreated {
			processCount := len(getRunningProcesses(dataFetcher))

			// only check for running members if the cluster was manually created
			if processCount > 0 {
				return fmt.Errorf("cluster %s has %d processes running. You must stop the cluster before removing it", clusterName, processCount)
			}
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
	RunE: func(cmd *cobra.Command, _ []string) error {
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
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a single cluster connection")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			cluster         = config.Cluster{}
			members         = config.Members{}
			services        = config.ServicesSummaries{}
			proxiesSummary  = config.ProxiesSummary{}
			reporters       = config.Reporters{}
			httpSessions    = config.HTTPSessionSummaries{}
			executors       = config.Executors{}
			healthSummaries = config.HealthSummaries{}
			storageInfo     = config.StorageDetails{}
			dataFetcher     fetcher.Fetcher
			edData          string
			err             error
			cachesData      string
			topicsData      string
		)

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

		clusterSummary, errorList := retrieveClusterSummary(dataFetcher)

		// check if any of the requests returned errors and only fail if all do
		if len(errorList) != 0 {
			// one or more errors.
			err = utils.GetErrors(errorList)
			_, _ = fmt.Fprint(os.Stderr, err.Error())
			return err
		}

		if verboseOutput && len(executors.Executors) > 0 {
			clusterSummary.executorsResult, err = json.Marshal(executors)
			if err != nil {
				return err
			}
		}

		if isJSONPathOrJSON() {
			clusterSummary.cachesResult, err = dataFetcher.GetCachesSummaryJSONAllServices()
			if err != nil {
				return err
			}
			// build the final json data
			jsonDataDest, _ := json.Marshal(clusterSummary.finalSummariesDestinations)
			jsonDataOrigins, _ := json.Marshal(clusterSummary.finalSummariesOrigins)
			finalResult, err := utils.CombineByteArraysForJSON(
				[][]byte{clusterSummary.clusterResult, clusterSummary.machinesData, clusterSummary.membersResult,
					clusterSummary.servicesResult, clusterSummary.cachesResult, clusterSummary.proxyResults,
					clusterSummary.reportersResult, clusterSummary.ramResult, clusterSummary.flashResult,
					clusterSummary.http, clusterSummary.executorsResult, jsonDataDest, jsonDataOrigins, clusterSummary.healthResult},
				[]string{"cluster", "machines", "members", "services", "caches", "proxies", "reporters", constants.RAMJournal,
					constants.FlashJournal, "httpServers", "executors", "federationDestinations", "federationOrigins", "health"})
			if err != nil {
				return err
			}

			return processJSONOutput(cmd, finalResult)
		}

		// format the output for text
		err = json.Unmarshal(clusterSummary.clusterResult, &cluster)
		if err != nil {
			return utils.GetError("unable to decode cluster details", err)
		}

		err = json.Unmarshal(clusterSummary.membersResult, &members)
		if err != nil {
			return utils.GetError("unable to decode members result", err)
		}

		err = json.Unmarshal(clusterSummary.servicesResult, &services)
		if err != nil {
			return utils.GetError("unable to decode services results", err)
		}

		err = json.Unmarshal(clusterSummary.storageData, &storageInfo)
		if err != nil {
			return utils.GetError("unable to decode storageInfo details", err)
		}

		storageMap := utils.GetStorageMap(storageInfo)

		if len(clusterSummary.proxyResults) > 0 {
			err = json.Unmarshal(clusterSummary.proxyResults, &proxiesSummary)
			if err != nil {
				return utils.GetError("unable to decode proxy details", err)
			}
		}

		if len(clusterSummary.reportersResult) > 0 {
			err = json.Unmarshal(clusterSummary.reportersResult, &reporters)
			if err != nil {
				return utils.GetError("unable to unmarshall reporter result", err)
			}
		}

		if len(clusterSummary.http) > 0 {
			err = json.Unmarshal(clusterSummary.http, &httpSessions)
			if err != nil {
				return utils.GetError("unable to decode Coherence*Web details", err)
			}
		}

		if len(clusterSummary.healthResult) > 0 {
			err = json.Unmarshal(clusterSummary.healthResult, &healthSummaries)
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
		sb.WriteString(FormatMachines(clusterSummary.machines))

		sb.WriteString("\nMEMBERS\n")
		sb.WriteString("-------\n")
		sb.WriteString(FormatMembers(members.Members, verboseOutput, storageMap, false, cluster.MembersDepartureCount))

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

		if len(clusterSummary.finalSummariesDestinations) > 0 || len(clusterSummary.finalSummariesOrigins) > 0 {
			sb.WriteString("\nFEDERATION\n")
			sb.WriteString("----------\n")
			if len(clusterSummary.finalSummariesDestinations) > 0 {
				sb.WriteString(FormatFederationSummary(clusterSummary.finalSummariesDestinations, destinations))
			}
			sb.WriteString("\n")
			if len(clusterSummary.finalSummariesOrigins) > 0 {
				sb.WriteString(FormatFederationSummary(clusterSummary.finalSummariesOrigins, origins))
			}
		}

		cachesData, err = formatCachesSummary(clusterSummary.serviceList, dataFetcher)
		if err != nil {
			return err
		}
		cachesData = "\nCACHES\n------\n" + cachesData

		topicsData = "\nTOPICS\n------\n" + FormatTopicsSummary(clusterSummary.topicsDetails.Details)

		sb.WriteString(cachesData + topicsData)

		if len(proxiesSummary.Proxies) > 0 {
			sb.WriteString("\nPROXY SERVERS\n")
			sb.WriteString("-------------\n")
			sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, "tcp"))
		}

		if len(proxiesSummary.Proxies) > 0 {
			sb.WriteString("\nHTTP SERVERS\n")
			sb.WriteString("------------\n")
			sb.WriteString(FormatProxyServers(proxiesSummary.Proxies, httpType))
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

		edData, err = getElasticDataResult(clusterSummary.flashResult, clusterSummary.ramResult)
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
			_ = executeJFROperation(cmd, "", fetcher.GetJFRs, dataFetcher, "", connection)
		}

		return nil
	},
}

// getDiscoveredClusters returns the discovered clusters and the count of clusters without HTTP management enabled.
func getDiscoveredClusters(cmd *cobra.Command, hostPorts []string) ([]discovery.DiscoveredCluster, []int, error) {
	var (
		clustersWithoutHTTP = make([]int, 0)
		numEndpoints        int
		err                 error
		ns                  *discovery.NSLookup
		clusterPorts        []discovery.ClusterNSPort
		finalClusterPorts   = make([]discovery.ClusterNSPort, 0)
		discoveredClusters  = make([]discovery.DiscoveredCluster, 0)
		discoveredCluster   discovery.DiscoveredCluster
	)
	for _, address := range hostPorts {
		ns, err = discovery.Open(address, timeout)
		if err != nil {
			err = logErrorAndCheck(nil, "error connecting to host/port: "+address, err)
			if err != nil {
				return discoveredClusters, clustersWithoutHTTP, err
			}
			// skip to the next address
			closeSilent(ns)
			continue
		}

		// now ask the Name Service for local and remote clusters it knows about
		clusterPorts, err = ns.DiscoverNameServicePorts()
		err = logErrorAndCheck(cmd, "unable to discover clusters on "+address, err)
		if err != nil {
			return discoveredClusters, clustersWithoutHTTP, err
		}

		closeSilent(ns)
		finalClusterPorts = append(finalClusterPorts, clusterPorts...)
	}

	// close the original lookup - possible optimization here
	closeSilent(ns)

	numEndpoints = len(finalClusterPorts)
	if numEndpoints == 0 {
		return discoveredClusters, clustersWithoutHTTP, errors.New("no valid Name Service endpoints found")
	}

	// now query each individual NS host/ port and gather the cluster information
	for i, nsAddress := range finalClusterPorts {
		var (
			nsNew *discovery.NSLookup
		)
		addressPort := fmt.Sprintf("%s:%d", nsAddress.HostName, nsAddress.Port)
		if cmd != nil {
			cmd.Printf("Discovering Management URL for %s on %s ...\n", nsAddress.ClusterName, addressPort)
		}

		nsNew, err = discovery.Open(addressPort, timeout)
		if err != nil {
			err = logErrorAndCheck(cmd, "unable to connect to "+addressPort, err)
			if err != nil {
				return discoveredClusters, clustersWithoutHTTP, err
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
				return discoveredClusters, clustersWithoutHTTP, err
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

	return discoveredClusters, clustersWithoutHTTP, nil
}

// discoverClustersCmd represents the discover clusters command.
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
			count     = len(args)
			err       error
			hostPorts []string
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

		discoveredClusters, clustersWithoutHTTP, err := getDiscoveredClusters(cmd, hostPorts)

		if err != nil {
			return err
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
				cmd.Printf("\nCluster: %s, Name Service address: %s:%d\n", cluster.ClusterName, cluster.Host, cluster.NSPort)

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

		var contextClusterName = ""

		// add the clusters
		for _, cluster := range discoveredClusters {
			if cluster.SelectedURL != "" {
				nsAddress := fmt.Sprintf("%s:%d", cluster.Host, cluster.NSPort)
				err = addCluster(cmd, cluster.ConnectionName, cluster.SelectedURL, "nslookup", nsAddress, false)
				err = logErrorAndCheck(cmd, "unable to discover cluster "+cluster.ConnectionName, err)
				if err != nil {
					return err
				}
				contextClusterName = cluster.ConnectionName
			}
		}

		// set the context only if one cluster was discovered
		if len(discoveredClusters) == 1 {
			if err = setContext(cmd, contextClusterName); err != nil {
				return err
			}
		}

		return nil
	},
}

// addCluster adds a new cluster.
func addCluster(cmd *cobra.Command, connection, connectionURL, discoveryType, nsAddress string, ignoreInvalidCerts bool) error {
	if ignoreInvalidCerts && !fetcher.IsValidHTTPSURL(connectionURL) {
		return errors.New("--ignore-certs requires a valid HTTPS connection URL")
	}
	// check to see if the url is just host:port and then build the full management URL using http as default
	// otherwise let it fall through and get validated
	if !strings.Contains(connectionURL, httpType) {
		split := strings.Split(connectionURL, ":")
		if len(split) == 2 {
			// candidate, second value must be int
			if utils.IsValidInt(split[1]) {
				connectionURL = fmt.Sprintf("http://%s:%s/management/coherence/cluster", split[0], split[1])
			}
		}
	}

	isWebLogic := fetcher.IsWebLogicServer(connectionURL)

	dataFetcher, err := fetcher.GetFetcherForConnection(connectionType, connectionURL, Username, "", connection, ignoreInvalidCerts)
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
	if fetcher.IsValidHTTPSURL(connectionURL) {
		newCluster.IgnoreInvalidCerts = &ignoreInvalidCerts
	}

	connections := append(append([]ClusterConnection(nil), Config.Clusters...), newCluster)
	if err = saveClusterConnections(connections); err != nil {
		return err
	}

	cmd.Printf("Added cluster %s with type %s and URL %s\n", connection, connectionType, connectionURL)
	return nil
}

// Publish live changes only after persistence succeeds; a failed write must not
// leave a connection or exception available to a subsequent command.
func saveClusterConnections(connections []ClusterConnection) error {
	previous := viper.Get(clusterKey)
	viper.Set(clusterKey, connections)
	if err := WriteConfig(); err != nil {
		viper.Set(clusterKey, previous)
		return err
	}
	Config.Clusters = connections
	return nil
}

// variables specifically for create cluster
var (
	httpPortParam            int32
	clusterPortParam         int32
	partitionCountParam      int32
	wkaParam                 string
	clusterVersionParam      string
	replicaCountParam        int32
	metricsStartPortParam    int32
	healthStartPortParam     int32
	jmxRemotePortParam       int32
	jmxRemoteHostParam       string
	logLevelParam            int32
	heapMemoryParam          string
	useCommercialParam       bool
	extendClientParam        bool
	grpcClientParam          bool
	profileFirstParam        bool
	machineParam             string
	rackParam                string
	siteParam                string
	roleParam                string
	skipMavenDepsParam       bool
	backupLogFilesParam      bool
	validPersistenceModes    = []string{"on-demand", "active", "active-backup", "active-async"}
	persistenceModeParam     string
	serverStartClassParam    string
	startupDelayParam        string
	additionalArtifactsParam string
	profileValueParam        string
	fileNameParam            string
	statementParam           string
	logDestinationParam      string
	cacheConfigParam         string
	operationalConfigParam   string
	settingsFileParam        string
	dynamicHTTPParam         bool
)

const (
	defaultCoherenceVersion = "14.1.2-0-5"
	startClusterCommand     = "start cluster"
	restartClusterCommand   = "restart cluster"
	scaleClusterCommand     = "scale cluster"
	stopClusterCommand      = "stop cluster"
	defaultHeap             = "128m"
	localHost               = "127.0.0.1"
	minPartitionCount       = 1
	maxPartitionCount       = 9973
)

// createClusterCmd represents the create cluster command.
var createClusterCmd = &cobra.Command{
	Use:   "cluster cluster-name",
	Short: "create a local Coherence cluster",
	Long: `The 'create cluster' command creates a local cluster, adds to the cohctl.yaml file 
and starts it. You must have the dependency tool and a Coherence-compatible Java executable in your PATH for
this to work. This cluster is only for development/testing purposes and should not be used, 
and is not supported in a production capacity. Supported versions are: CE 22.06 and above and 
commercial 14.1.1.2206.1 and above. Default version is currently CE ` + defaultCoherenceVersion + `.
NOTE: This is an experimental feature and my be altered or removed in the future.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		var (
			clusterName    = sanitizeConnectionName(args[0])
			err            error
			groupID        = getCoherenceGroupID()
			cpEntry        string
			splitArtifacts []string
		)

		// validate persistence mode
		if !utils.SliceContains(validPersistenceModes, persistenceModeParam) {
			return fmt.Errorf("invalid persistence mode %s. Must be one of %v", persistenceModeParam, validPersistenceModes)
		}

		// validate partition count
		if partitionCountParam < minPartitionCount || partitionCountParam > maxPartitionCount {
			return fmt.Errorf("partition count must be between %v and %v", minPartitionCount, maxPartitionCount)
		}

		err = validateOverrideParams()
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

		// validate ensure unique cluster name
		if err = ensureUniqueCluster(clusterName); err != nil {
			return err
		}

		heap := Config.DefaultHeap
		if heap == "" {
			heap = heapMemoryParam
		}

		// validate logging destination
		if logDestinationParam != "" {
			stat, err := os.Stat(logDestinationParam)
			if err != nil || !stat.IsDir() {
				return fmt.Errorf("directory %s does not exist", logDestinationParam)
			}
			if !filepath.IsAbs(logDestinationParam) {
				return errors.New("you must provide an absolute path for log destination")
			}
			logsDirectory = logDestinationParam
		}

		arguments := fmt.Sprintf("-Dcoherence.cluster=%s -Dcoherence.clusterport=%d -Dcoherence.ttl=0 -Dcoherence.wka=%s -Djava.net.preferIPv4Stack=true"+
			" -Dcoherence.distributed.partitioncount=%d -Dcoherence.distributed.partitions=%d%s%s",
			clusterName, clusterPortParam, wkaParam, partitionCountParam, partitionCountParam, getCacheConfigProperty(), getOperationalOverrideConfigProperty())
		startupConnection := ClusterConnection{Name: clusterName, ConnectionType: httpType, ClusterName: clusterName, ManuallyCreated: true,
			Arguments: arguments, ManagementPort: httpPortParam, ClusterVersion: clusterVersionParam,
			ClusterType: "Standalone", PersistenceMode: persistenceModeParam,
			LoggingDestination: logDestinationParam, StartupClass: serverStartClassParam}
		plan, err := preflightManagedStartup(cmd, startupConnection, replicaCountParam, 0, false)
		if err != nil {
			return err
		}
		if err = checkManagementPorts(startupConnection, *plan.connection.Management, false); err != nil {
			return err
		}
		if err = checkDepsRequirements(); err != nil {
			return err
		}
		showManagedSettings(cmd, startupConnection, *plan.connection.Management)

		cmd.Printf("\nCluster name:         %s\n", clusterName)
		cmd.Printf("Cluster version:      %s\n", clusterVersionParam)
		cmd.Printf("Cluster port:         %d\n", clusterPortParam)
		cmd.Printf("Management port:      %d\n", httpPortParam)
		cmd.Printf("Partition count:      %d\n", partitionCountParam)
		cmd.Printf("Replica count:        %d\n", replicaCountParam)
		cmd.Printf("Initial memory:       %s\n", heap)
		cmd.Printf("Persistence mode:     %s\n", persistenceModeParam)
		cmd.Printf("Group ID:             %s\n", groupID)
		cmd.Printf("Additional artifacts: %v\n", additionalArtifactsParam)
		cmd.Printf("Startup Profile:      %v\n", profileValueParam)
		cmd.Printf("Log destination root: %v\n", logDestinationParam)
		cmd.Printf("Cache Config:         %v\n", cacheConfigParam)
		cmd.Printf("Operational Override: %v\n", operationalConfigParam)
		cmd.Printf("Startup Class:        %v\n", serverStartClassParam)
		cmd.Printf("Dependency tool:      %v\n", getExecType())

		// confirm the operation
		if err = cmd.Context().Err(); err != nil {
			return err
		}
		if !confirmOperation(cmd, "Are you sure you want to create the cluster with the above details? (y/n) ") {
			return cmd.Context().Err()
		}
		if err = cmd.Context().Err(); err != nil {
			return err
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
					// bring in many dependencies due to me not yet figuring out how to not bring in optional deps
					cpEntry, err = getTransitiveClasspath(entry.GroupID, entry.Artifact, entry.Version)

					if err != nil {
						return err
					}
					classpath = append(classpath, cpEntry)
				}
			}
		}

		plan.connection.BaseClasspath = strings.Join(classpath, getClasspathSeparator())

		cmd.Printf("Starting %d cluster members for cluster %s\n", replicaCountParam, clusterName)

		// No children exist during validation, dependency resolution or confirmation;
		// retain normal OS signal behavior there, including blocking terminal input.
		restore := managedOperationSignals(cmd)
		defer restore()
		if _, err = runPreparedStartup(cmd, plan, managedCreate); err != nil {
			return err
		}
		cmd.Printf("Saved management settings and URL %s\n", plan.connection.ConnectionURL)

		cmd.Println("Cluster added and started")

		return nil
	},
}

// getClusterConfigCmd represents the get cluster-config command.
var getClusterConfigCmd = &cobra.Command{
	Use:   "cluster-config",
	Short: "display the cluster operational config",
	Long: `The 'get cluster-config' displays the cluster operational config for a
cluster using the current context or a cluster specified by using '-c'. Only available
in most recent Coherence versions`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			connection  string
			dataFetcher fetcher.Fetcher
			data        []byte
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		cmd.Println(FormatCurrentCluster(connection))

		data, err = dataFetcher.GetClusterConfig()
		if err != nil {
			return err
		}

		cmd.Println(string(data))

		return nil
	},
}

// getClusterDescription represents the get cluster-description command.
var getClusterDescription = &cobra.Command{
	Use:   "cluster-description",
	Short: "display cluster description",
	Long: `The 'get cluster-description' command displays information regarding a cluster and it's members.
Only available in most recent Coherence versions.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			dataFetcher fetcher.Fetcher
			connection  string
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var (
				descriptionData []byte
				description     config.Description
			)

			descriptionData, err = dataFetcher.GetClusterDescriptionJSON()
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

// startClusterCmd represents the start cluster command.
var startClusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "start a local Coherence cluster",
	Long:              `The 'start cluster' command starts a cluster that was manually created.`,
	ValidArgsFunction: completionAllManualClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], startClusterCommand)
	},
}

// setClusterCmd represents the set cluster command.
var setClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "set attributes for all members in cluster",
	Long: `The 'set cluster' command sets an attribute for all members across a cluster.
The following attribute names are allowed: bufferPublishSize, bufferReceiveSize, loggingFormat,
loggingLevel, loggingLimit, multicastThreshold, resendDelay, sendAckDelay, trafficJamCount or trafficJamDelay. 
Note: Many of these are advanced cluster configuration values and setting them should be done 
carefully and in consultation with Oracle Support.`,
	ValidArgsFunction: completionAllManualClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			connectionName             = args[0]
			valueToSet     interface{} = attributeValueCluster
			dataFetcher    fetcher.Fetcher
			err            error
		)

		if !utils.SliceContains(validSetClusterAttributes, attributeNameCluster) {
			return fmt.Errorf("invalid attribute %s valid values are: %v", attributeNameCluster, validSetClusterAttributes)
		}

		// validate anything other than loggingFormat as an int
		if attributeNameCluster != "loggingFormat" {
			v, err := strconv.Atoi(attributeValueCluster)
			if err != nil {
				return fmt.Errorf("invalid value for %s attribute: %s", attributeNameCluster, attributeValueCluster)
			}
			valueToSet = v
		}

		found, _ := GetClusterConnection(connectionName)
		if !found {
			return errors.New(UnableToFindClusterMsg + connectionName)
		}

		dataFetcher, err = GetDataFetcher(connectionName)
		if err != nil {
			return err
		}

		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the attribute %s to value %v on all members of cluster %s? (y/n) ", attributeNameCluster, valueToSet, connectionName)) {
			return nil
		}

		_, err = dataFetcher.SetClusterAttribute(attributeNameCluster, valueToSet)
		if err != nil {
			return err
		}
		cmd.Println(OperationCompleted)

		return nil
	},
}

// scaleClusterCmd represents the start cluster command.
var scaleClusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "scales a local Coherence cluster",
	Long:              `The 'scale cluster' command scales a cluster that was manually created.`,
	ValidArgsFunction: completionAllManualClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], scaleClusterCommand)
	},
}

// stopClusterCmd represents the stop cluster command.
var stopClusterCmd = &cobra.Command{
	Use:               "cluster",
	Short:             "stop a local Coherence cluster",
	Long:              `The 'stop cluster' command stops a cluster that was manually created or started.`,
	ValidArgsFunction: completionAllManualClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], stopClusterCommand)
	},
}

// restartClusterCmd represents the stop cluster command.
var restartClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "restart a local Coherence cluster",
	Long: `The 'restart cluster' command stops and restarts a manually created cluster.
It requires a usable member snapshot and a matching positive membership count from
the saved management URL before stopping any JVMs. Use 'start cluster' for an actually stopped cluster.`,
	ValidArgsFunction: completionAllManualClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClusterOperation(cmd, args[0], restartClusterCommand)
	},
}

// startConsoleCmd represents the start console command.
var startConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "start a console client against a cluster that was manually created",
	Long: `The 'start console' command starts a console client which connects to a
cluster using the current context or a cluster specified by using '-c'.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runStartClientOperation(cmd, consoleClass)
	},
}

// startCohQL represents the start cohql command.
var startCohQLCmd = &cobra.Command{
	Use:   "cohql",
	Short: "start a CohQL client against a cluster that was manually created",
	Long: `The 'start cohql' command starts a CohQL client which connects to a
cluster using the current context or a cluster specified by using '-c'..`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runStartClientOperation(cmd, cohQLClass)
	},
}

// startClassCmd represents the start class command.
var startClassCmd = &cobra.Command{
	Use:   "class",
	Short: "start a specific Java class against a cluster that was manually created",
	Long: `The 'start class' command starts a specific Java class which connects to a
cluster using the current context or a cluster specified by using '-c'.
The class name must include the full package and class name and must be included in
an artefact included in the initial cluster creation. You may pass additional arguments
which will be passed to the called class.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			displayErrorAndExit(cmd, "you must provide a class to run")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStartClientOperation(cmd, args[0], args[1:]...)
	},
}

func runStartClientOperation(cmd *cobra.Command, class string, extraArgs ...string) error {
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

	return startClient(cmd, clusterConn, class, extraArgs...)
}

func runClusterOperation(cmd *cobra.Command, connectionName, operation string) error {
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	found, connection := GetClusterConnection(connectionName)
	if !found {
		return errors.New(UnableToFindClusterMsg + connectionName)
	}
	if err := checkOperation(connection, operation); err != nil {
		return err
	}
	var plan managedStartPlan
	var err error
	// Stop deliberately bypasses all JVM, profile and startup-option checks.
	if operation == startClusterCommand || operation == restartClusterCommand {
		plan, err = preflightManagedStartup(cmd, connection, replicaCountParam, 0, false)
		if err != nil {
			return err
		}
		showManagedSettings(cmd, connection, *plan.connection.Management)
	}
	dataFetcher, err := GetDataFetcher(connectionName) // always the old saved URL
	if err != nil {
		return err
	}
	var processIDs []int
	if operation == restartClusterCommand {
		processIDs, err = discoverLifecycleMembers(dataFetcher)
		if err != nil {
			return fmt.Errorf("cannot restart cluster %s: old-member snapshot unavailable or unusable; no stop or replacement attempted; use start cluster for an actually stopped cluster: %w", connection.Name, err)
		}
	} else {
		processIDs = getRunningProcesses(dataFetcher)
	}
	numProcesses := int32(len(processIDs))
	if numProcesses > 0 {
		result, err := dataFetcher.GetClusterDetailsJSON()
		if err != nil {
			return err
		}
		var cluster config.Cluster
		if err = json.Unmarshal(result, &cluster); err != nil {
			return err
		}
		if cluster.ClusterName != connection.ClusterName {
			return fmt.Errorf("a different cluster %s is running at the saved management URL; refusing %s", cluster.ClusterName, operation)
		}
		// Missing/null clusterSize decodes to zero. Restart needs a positive
		// count matching every validated distinct PID before acquiring observers
		// or requesting any stops. REST views can still change concurrently.
		if operation == restartClusterCommand && (cluster.ClusterSize <= 0 || cluster.ClusterSize != len(processIDs)) {
			return fmt.Errorf("cannot restart cluster %s: old-member snapshot has unusable or inconsistent clusterSize %d for %d distinct member PIDs; require a positive matching count; no stop or replacement attempted", connection.Name, cluster.ClusterSize, len(processIDs))
		}
	}
	if (operation == stopClusterCommand || operation == scaleClusterCommand) && numProcesses == 0 {
		return fmt.Errorf("the cluster %s does not appear to be started", connection.Name)
	}
	if operation == startClusterCommand && numProcesses > 0 {
		return fmt.Errorf("the cluster %s appears to be already started with process ids: %v", connection.Name, processIDs)
	}
	count := replicaCountParam
	if operation == startClusterCommand || operation == restartClusterCommand {
		if err = checkManagementPorts(connection, *plan.connection.Management, operation == restartClusterCommand && numProcesses > 0); err != nil {
			return err
		}
	}
	if operation == scaleClusterCommand {
		if count <= numProcesses {
			return fmt.Errorf("replicas must exceed the current %d members; scaling down is not supported", numProcesses)
		}
		count -= numProcesses
		plan, err = preflightManagedStartup(cmd, connection, count, numProcesses, true)
		if err != nil {
			return err
		}
		cmd.Printf("Scaling the cluster %s up by %d member(s) to %d members\n", connection.Name, count, replicaCountParam)
	}
	stopping := operation == stopClusterCommand || operation == restartClusterCommand && numProcesses > 0
	if err = cmd.Context().Err(); err != nil {
		return err
	}
	if stopping && !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to stop %d members for the cluster %s? (y/n) ", numProcesses, connection.Name)) {
		return cmd.Context().Err()
	}
	// Register only after all terminal input and preflight, before any stop or
	// launch. Unrelated commands retain their existing signal behavior.
	restore := managedOperationSignals(cmd)
	defer restore()
	if err = cmd.Context().Err(); err != nil {
		return err
	}
	if stopping {
		if operation == restartClusterCommand {
			if err = stopClusterForRestart(cmd, connection, *plan.connection.Management, processIDs); err != nil {
				return err
			}
		} else {
			if err = stopClusterProcesses(cmd, connection.Name, processIDs); err != nil {
				return fmt.Errorf("unable to stop cluster %s: stop incomplete: %w", connection.Name, err)
			}
			return cmd.Context().Err()
		}
	}
	if connection.LoggingDestination != "" {
		logsDirectory = connection.LoggingDestination
	}
	save := managedUpdate
	if operation == scaleClusterCommand {
		save = managedNoSave
	}
	_, err = runPreparedStartup(cmd, plan, save)
	if err == nil && save != managedNoSave {
		cmd.Printf("Saved management settings and URL %s\n", plan.connection.ConnectionURL)
	}
	if err != nil {
		if operation == restartClusterCommand && numProcesses > 0 {
			return fmt.Errorf("replacement failed after stop requests for the old cluster; previous configuration retained; stopped members cannot be restored automatically: %w", err)
		}
		return err
	}
	cmd.Printf("Cluster %s %s completed\n", connection.Name, operation)
	return nil
}

func init() {
	addClusterCmd.Flags().StringVarP(&connectionURL, "url", "u", "", "connection URL")
	_ = addClusterCmd.MarkFlagRequired("url")
	addClusterCmd.Flags().StringVarP(&connectionType, "type", "t", httpType, "connection type, http")
	addClusterCmd.Flags().Bool(ignoreCertsFlag, false, "allow invalid certificates only for this saved HTTPS connection")

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
	createClusterCmd.Flags().Int32VarP(&partitionCountParam, "partition-count", "T", 257, "partition count")
	createClusterCmd.Flags().StringVarP(&wkaParam, "wka", "K", localHost, "well known address")
	createClusterCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	createClusterCmd.Flags().StringVarP(&startupDelayParam, startupDelayArg, "D", "0ms", startupDelayMessage)
	createClusterCmd.Flags().Int32VarP(&replicaCountParam, "replicas", "r", 3, serverCountMessage)
	createClusterCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	createClusterCmd.Flags().StringVarP(&additionalArtifactsParam, "additional", "a", "", "additional comma separated Coherence artifacts or others in G:A:V format")
	createClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	createClusterCmd.Flags().BoolVarP(&useCommercialParam, "commercial", "C", false, "use commercial Coherence groupID (default CE)")
	createClusterCmd.Flags().BoolVarP(&skipMavenDepsParam, "skip-deps", "I", false, "skip pulling artifacts")
	createClusterCmd.Flags().Int32VarP(&metricsStartPortParam, metricsPortArg, "t", 0, metricsPortMessage)
	createClusterCmd.Flags().Int32VarP(&healthStartPortParam, healthPortArg, "e", 0, healthPortMessage)
	createClusterCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)
	createClusterCmd.Flags().StringVarP(&serverStartClassParam, startClassArg, "S", "", startClassMessage)
	createClusterCmd.Flags().StringVarP(&logDestinationParam, logDestinationArg, "L", "", logDestinationMessage)
	createClusterCmd.Flags().StringVarP(&cacheConfigParam, cacheConfigArg, "", "", cacheConfigMessage)
	createClusterCmd.Flags().StringVarP(&operationalConfigParam, operationalConfigArg, "", "", operationalConfigMessage)
	createClusterCmd.Flags().Int32VarP(&jmxRemotePortParam, jmxPortArg, "J", 0, jmxPortMessage)
	createClusterCmd.Flags().StringVarP(&jmxRemoteHostParam, jmxHostArg, "j", "", jmxHostMessage)
	managementFlags(createClusterCmd)
	createClusterCmd.Flags().BoolVarP(&profileFirstParam, profileFirstArg, "F", false, profileFirstMessage)
	createClusterCmd.Flags().StringVarP(&machineParam, machineArg, "", "", machineMessage)
	createClusterCmd.Flags().StringVarP(&rackParam, rackArg, "", "", rackMessage)
	createClusterCmd.Flags().StringVarP(&siteParam, siteArg, "", "", siteMessage)
	createClusterCmd.Flags().StringVarP(&roleParam, roleArg, "", "", roleMessage)
	createClusterCmd.Flags().StringVarP(&settingsFileParam, "maven-settings", "", "", "full path to Maven settings file")
	createClusterCmd.Flags().BoolVarP(&dynamicHTTPParam, "dynamic-http", "N", false, dynamicHTTPMessage)

	stopClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	startConsoleCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startConsoleCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startConsoleCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	startCohQLCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startCohQLCmd.Flags().StringVarP(&fileNameParam, "file", "f", "", "file name to read CohQL commands from")
	startCohQLCmd.Flags().StringVarP(&statementParam, "statement", "S", "", "statement to execute enclosed in double quotes")
	startCohQLCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startCohQLCmd.Flags().BoolVarP(&extendClientParam, "extend", "X", false, "start CohQL as Extend client. Only works for default cache config")
	startCohQLCmd.Flags().BoolVarP(&grpcClientParam, "grpc", "G", false, "start CohQL as gRPC client. Only works for default cache config")
	startCohQLCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	startClassCmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	startClassCmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	startClassCmd.Flags().BoolVarP(&extendClientParam, "extend", "X", false, "start a class as Extend client. Only works for default cache config")
	startClassCmd.Flags().BoolVarP(&grpcClientParam, "grpc", "G", false, "start a class as gRPC client. Only works for default cache config")
	startClassCmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)

	applyStartParams(scaleClusterCmd)
	applyStartParams(restartClusterCmd)
	applyStartParams(startClusterCmd)

	restartClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	setClusterCmd.Flags().StringVarP(&attributeNameCluster, "attribute", "a", "", attrNameToSet)
	_ = setClusterCmd.MarkFlagRequired("attribute")
	setClusterCmd.Flags().StringVarP(&attributeValueCluster, "value", "v", "", attrValueToSet)
	_ = setClusterCmd.MarkFlagRequired("value")
	setClusterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}

func applyStartParams(cmd *cobra.Command) {
	cmd.Flags().Int32VarP(&replicaCountParam, "replicas", "r", 3, serverCountMessage)
	cmd.Flags().Int32VarP(&metricsStartPortParam, metricsPortArg, "t", 0, metricsPortMessage)
	cmd.Flags().Int32VarP(&healthStartPortParam, healthPortArg, "e", 0, healthPortMessage)
	cmd.Flags().StringVarP(&heapMemoryParam, heapMemoryArg, "M", defaultHeap, heapMemoryMessage)
	cmd.Flags().StringVarP(&profileValueParam, profileArg, "P", "", profileMessage)
	cmd.Flags().Int32VarP(&logLevelParam, logLevelArg, "l", 5, logLevelMessage)
	cmd.Flags().StringVarP(&startupDelayParam, startupDelayArg, "D", "0ms", startupDelayMessage)
	cmd.Flags().StringVarP(&serverStartClassParam, startClassArg, "S", "", startClassMessage)
	cmd.Flags().Int32VarP(&jmxRemotePortParam, jmxPortArg, "J", 0, jmxPortMessage)
	cmd.Flags().StringVarP(&jmxRemoteHostParam, jmxHostArg, "j", "", jmxHostMessage)
	managementFlags(cmd)
	cmd.Flags().BoolVarP(&profileFirstParam, profileFirstArg, "F", false, profileFirstMessage)
	cmd.Flags().BoolVarP(&backupLogFilesParam, backupLogFilesArg, "B", false, backupLogFilesMessage)
	cmd.Flags().StringVarP(&machineParam, machineArg, "", "", machineMessage)
	cmd.Flags().StringVarP(&rackParam, rackArg, "", "", rackMessage)
	cmd.Flags().StringVarP(&siteParam, siteArg, "", "", siteMessage)
	cmd.Flags().StringVarP(&roleParam, roleArg, "", "", roleMessage)
	cmd.Flags().BoolVarP(&dynamicHTTPParam, "dynamic-http", "N", false, dynamicHTTPMessage)
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
func acceptIntegerValue(cmd *cobra.Command, message string, minValue, maxValue int) (int, error) {
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
		if err != nil || value < minValue || value > maxValue {
			cmd.Printf("Please enter a value between %d and %d\n", minValue, maxValue)
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
	if cmd != nil {
		cmd.Println(actualError)
	}

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
	return validateProfileValue(getProfileValue(profileValueParam))
}

func validateProfileValue(startupProfile string) error {
	if profileValueParam != "" && startupProfile == "" {
		return fmt.Errorf("a profile with the name %s does not exist", profileValueParam)
	}

	if profileFirstParam && profileValueParam == "" {
		return errors.New("if you specified -F option you must also specify a profile")
	}

	return nil
}
