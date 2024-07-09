/*
 * Copyright (c) 2022, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-go-client/coherence/discovery"
	"github.com/spf13/cobra"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	healthSubType   string
	healthName      string
	healthSummary   bool
	nslookupAddress string
	healthEndpoints string
	getNodeID       bool
	ignoreNSErrors  bool
	healthTimeout   int32
)

// getHealthCmd represents the get health command.
var getHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "display health information for a cluster",
	Long:  `The 'get health' command displays the health for members of a cluster.`,
	Args:  cobra.ExactArgs(0),
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
				healthData      []byte
				healthSummaries = config.HealthSummaries{}
			)

			healthData, err = dataFetcher.GetMembersHealth()
			if err != nil {
				return err
			}

			if isJSONPathOrJSON() {
				if err = processJSONOutput(cmd, healthData); err != nil {
					return err
				}
			} else {
				if len(healthData) > 0 {
					err = json.Unmarshal(healthData, &healthSummaries)
					if err != nil {
						return err
					}
				}

				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))

				var (
					count    = len(healthSummaries.Summaries)
					filtered = filterHealth(healthSummaries)
				)

				if count > 0 && len(filtered) == 0 {
					return fmt.Errorf("filter on sub-type=%s and name=%s returned no entries", healthSubType, healthName)
				}

				if healthSummary {
					// summarise the health data across nodes
					healthShort := summariseHealth(filtered)
					cmd.Println(FormatHealthSummary(healthShort))
				} else {
					cmd.Println(FormatMemberHealth(filtered))
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

func filterHealth(health config.HealthSummaries) []config.HealthSummary {
	var (
		filtered = make([]config.HealthSummary, 0)
	)

	for _, value := range health.Summaries {
		if healthSubType != all && value.SubType != healthSubType {
			continue
		}
		if healthName != all && value.Name != healthName {
			continue
		}

		filtered = append(filtered, value)
	}

	return filtered
}

func summariseHealth(health []config.HealthSummary) []config.HealthSummaryShort {
	var (
		healthShort = make([]config.HealthSummaryShort, 0)
	)

	for _, value := range health {
		i := findIndex(healthShort, value.Name, value.SubType)
		var entry config.HealthSummaryShort
		if i == -1 {
			// not found so append a new one
			entry = config.HealthSummaryShort{Name: value.Name, SubType: value.SubType}
			healthShort = append(healthShort, entry)
			i = int32(len(healthShort) - 1)
		} else {
			// use existing
			entry = healthShort[i]
		}

		// update the entry values
		entry.TotalCount++
		if value.Started {
			entry.StartedCount++
		}

		if value.Ready {
			entry.ReadyCount++
		}

		if value.Live {
			entry.LiveCount++
		}

		if value.Safe {
			entry.SafeCount++
		}

		healthShort[i] = entry
	}

	return healthShort
}

// findIndex finds the index of the entry for name and subType, -1 means no entry found.
func findIndex(health []config.HealthSummaryShort, name, subType string) int32 {
	for i, value := range health {
		if value.Name == name && value.SubType == subType {
			return int32(i)
		}
	}

	return -1 // not found
}

// monitorHealthCmd represents the monitor health command.
var monitorHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "monitors health information for a cluster or set of health endpoints",
	Long: `The 'get monitor' command monitors the health of nodes for a cluster or set of health endpoints.
Specify -n and a host:port to lookup or -e and a list of http endpoints without the path.
You may also specify -T option to wait until all health endpoints are safe.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			err         error
			endpoints   []string
			ns          *discovery.NSLookup
			result      string
			clusterName string
			dataFetcher fetcher.Fetcher
		)
		if healthEndpoints == "" && nslookupAddress == "" {
			return errors.New("you must specify one of -n or -e")
		} else if healthEndpoints != "" && nslookupAddress != "" {
			return errors.New("you cannot specify both -n and -e")
		}

		if getNodeID {
			// get the data fetcher if we are wanting the node ID
			_, dataFetcher, err = GetConnectionAndDataFetcher()
			if err != nil {
				return err
			}

			// retrieve cluster details first so if we are connected
			// to WLS or need authentication, this can be done first
			_, err1 := dataFetcher.GetClusterDetailsJSON()
			if err1 != nil {
				return err1
			}
		}

		// retrieve the list of endpoints using either of the 3 methods
		if healthEndpoints != "" {
			endpoints, err = parseHealthEndpoints(healthEndpoints)
			if err != nil {
				return err
			}
		}

		if healthTimeout < 0 {
			return errors.New("you must provide a positive value in seconds for health timeout")
		}
		if healthTimeout > 0 && !isWatchEnabled() {
			return errors.New("if you have specified a health timeout you must enable watch option")
		}
		startTime := time.Now()

		for {
			var emptyData = false
			if nslookupAddress != "" {
				// use nslookup to look up the health endpoint of the cluster
				ns, err = discovery.Open(nslookupAddress, timeout)
				if err != nil {
					if ignoreNSErrors {
						emptyData = true
						clusterName = "Not available"
					} else {
						return fmt.Errorf("unable to use nslookup against %s: %v", nslookupAddress, err)
					}
				}

				if !emptyData {
					result, err = ns.Lookup("NameService/string/health/HTTPHealthURL")
					if err != nil {
						return err
					}

					clusterName, err = ns.Lookup("Cluster/name")
					if err != nil {
						return err
					}

					_ = ns.Close()
				}

				// format returned is [http://127.0.0.1:6676/, http://127.0.0.1:6677/]
				result = strings.Replace(result, "[", "", 1)
				result = strings.Replace(result, "]", "", 1)
				result = strings.Replace(result, " ", "", -1)
				endpoints, err = parseHealthEndpoints(result)
				if err != nil && !emptyData {
					return err
				}
			}

			monitoringData := gatherMonitorData(dataFetcher, endpoints)

			printWatchHeader(cmd)

			cmd.Println("\nHEALTH MONITORING")
			cmd.Println("------------------")
			if nslookupAddress != "" {
				cmd.Println("Name Service:   ", nslookupAddress)
				cmd.Println("Cluster Name:   ", clusterName)

			} else {
				cmd.Println("Endpoints:     ", endpoints)
			}
			cmd.Println("All Nodes Safe: ", falseBoolFormatter(fmt.Sprintf("%v", isMonitoringDataSafe(monitoringData))))

			cmd.Println()

			cmd.Println(FormatHealthMonitoring(monitoringData))

			if healthTimeout > 0 {
				elapsedSeconds := int32(time.Since(startTime).Seconds())
				if isMonitoringDataSafe(monitoringData) {
					cmd.Printf("All health endpoints are safe reached in %d seconds\n", elapsedSeconds)
					return nil
				}
				if elapsedSeconds > healthTimeout {
					return fmt.Errorf("all health endpoints NOT safe in %d seconds", elapsedSeconds)
				}

				cmd.Printf("Waiting for all health endpoints to be safe within %d seconds", healthTimeout)
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

func isMonitoringDataSafe(monitoringData []config.HealthMonitoring) bool {
	if len(monitoringData) == 0 {
		return false
	}

	for _, v := range monitoringData {
		if v.Safe != http200 || v.Ready != http200 || v.Live != http200 || v.Started != http200 {
			return false
		}
	}
	return true
}

func gatherMonitorData(dataFetcher fetcher.Fetcher, endpoints []string) []config.HealthMonitoring {
	var (
		result      = make([]config.HealthMonitoring, len(endpoints))
		httpFetcher fetcher.Fetcher
	)

	// we use a mock http fetcher just so we can use the http methods if we do not already have one
	if dataFetcher == nil {
		httpFetcher, _ = fetcher.GetFetcherOrError("http", "", "", "")
	} else {
		httpFetcher = dataFetcher
	}

	for i, v := range endpoints {
		var (
			wg           sync.WaitGroup
			httpResult   = make([]string, 4)
			routineCount = 4
			nodeID       = "n/a"
		)

		healthURLS := []string{
			getHealthEndpoint(v, "started"),
			getHealthEndpoint(v, "live"),
			getHealthEndpoint(v, "ready"),
			getHealthEndpoint(v, "safe"),
		}

		if getNodeID {
			routineCount++
		}

		// issue concurrent requests
		wg.Add(routineCount)

		for j := 0; j < 4; j++ {
			go func(healthURL string, index int) {
				defer wg.Done()
				httpResult[index] = httpFetcher.GetResponseCode(healthURL)
			}(healthURLS[j], j)
		}

		if getNodeID {
			// extract the port from the URL
			parsed, _ := url.Parse(v)
			port := parsed.Port()

			proxyResults, err := dataFetcher.GetProxySummaryJSON()
			if err == nil {
				var proxiesSummary = config.ProxiesSummary{}
				err = json.Unmarshal(proxyResults, &proxiesSummary)
				if err == nil {
					// loop through each entry and see if we have a match for the protocol
					for _, vv := range proxiesSummary.Proxies {
						if strings.Contains(vv.HostIP, port) {
							nodeID = vv.NodeID
							break
						}
					}
				}
			}
			wg.Done()
		}
		wg.Wait()

		result[i] = config.HealthMonitoring{Endpoint: v,
			Started: httpResult[0],
			Live:    httpResult[1],
			Ready:   httpResult[2],
			Safe:    httpResult[3],
			NodeID:  nodeID,
		}
	}

	return result
}

func init() {
	getHealthCmd.Flags().StringVarP(&healthSubType, "sub-type", "s", all, "health sub-type")
	getHealthCmd.Flags().StringVarP(&healthName, "name", "n", all, "health name")
	getHealthCmd.Flags().BoolVarP(&healthSummary, "summary", "S", false, "if true, returns a summary across nodes")

	monitorHealthCmd.Flags().BoolVarP(&getNodeID, "node-id", "N", false, "if true, returns the node id using the current context")
	monitorHealthCmd.Flags().BoolVarP(&ignoreNSErrors, "ignore-errors", "I", false, "if true, ignores nslookup errors")
	monitorHealthCmd.Flags().Int32VarP(&timeout, "timeout", "t", 30, timeoutMessage)
	monitorHealthCmd.Flags().StringVarP(&healthEndpoints, "endpoints", "e", "", "csv list of health endpoints")
	monitorHealthCmd.Flags().StringVarP(&nslookupAddress, "nslookup", "n", "", "host:port to connect to to lookup health endpoints")
	monitorHealthCmd.Flags().Int32VarP(&healthTimeout, "health-timeout", "T", 0, "timeout to wait for all health checks to be status 200")
}
