/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

var (
	healthSubType string
	healthName    string
	healthSummary bool
)

// getHealthCmd represents the get health command
var getHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "display health information for a cluster",
	Long:  `The 'get health' command displays the health for members of a cluster.`,
	Args:  cobra.ExactArgs(0),
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

		for {
			var (
				healthData      []byte
				healthSummaries = config.HealthSummaries{}
			)

			healthData, err = dataFetcher.GetMembersHealth()
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(healthData, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(healthData))
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
		if healthSubType != "all" && value.SubType != healthSubType {
			continue
		}
		if healthName != "all" && value.Name != healthName {
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

// findIndex finds the index of the entry for name and subType, -1 means no entry found
func findIndex(health []config.HealthSummaryShort, name, subType string) int32 {
	for i, value := range health {
		if value.Name == name && value.SubType == subType {
			return int32(i)
		}
	}

	return -1 // not found
}

func init() {
	getHealthCmd.Flags().StringVarP(&healthSubType, "sub-type", "s", "all", "health sub-type")
	getHealthCmd.Flags().StringVarP(&healthName, "name", "n", "all", "health name")
	getHealthCmd.Flags().BoolVarP(&healthSummary, "summary", "S", false, "if true, returns a summary across nodes")
}
