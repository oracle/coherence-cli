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

				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				cmd.Println(FormatCurrentCluster(connection))

				var (
					count    = len(healthSummaries.Summaries)
					filtered = filterHealth(healthSummaries)
				)

				if count > 0 && len(filtered) == 0 {
					return fmt.Errorf("filter on sub-type=%s and name=%s returned no entries", healthSubType, healthName)
				}

				cmd.Println(FormatMemberHealth(filtered))
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

func init() {
	getHealthCmd.Flags().StringVarP(&healthSubType, "sub-type", "s", "all", "health sub-type")
	getHealthCmd.Flags().StringVarP(&healthName, "name", "n", "all", "health name")
}
