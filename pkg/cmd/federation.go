/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
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

const destinations = "destinations"
const origins = "origins"
const outgoing = "outgoing"
const incoming = "incoming"
const participantMessage = "participant to apply to"
const supplyService = "you must provide a service name"
const federationUse = "federation service-name"

var (
	participant             string
	startMode               string
	replicateAllParticipant string
)

// getFederationCmd represents the get federation command
var getFederationCmd = &cobra.Command{
	Use:   "federation {destinations|origins|all}",
	Short: "display federation details for a cluster",
	Long: `The 'get federation' command displays the federation details for a cluster. 
You must specify either destinations, origins or all to show both. You 
can also specify '-o wide' to display addition information.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide either all, "+destinations+" or "+origins)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err               error
			dataFetcher       fetcher.Fetcher
			connection        string
			federatedServices []string
			target            string
		)

		if args[0] == destinations {
			target = outgoing
		} else if args[0] == origins {
			target = incoming
		} else if args[0] == "all" {
			target = "all"
		} else {
			return fmt.Errorf("you must specify either %s, %s or all", destinations, origins)
		}

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// filter the federated services only
		federatedServices, err = GetFederatedServices(dataFetcher)
		if err != nil {
			return err
		}

		if len(federatedServices) == 0 {
			return nil
		}

		for {
			var (
				finalSummariesDestinations []config.FederationSummary
				finalSummariesOrigins      []config.FederationSummary
			)

			if target == outgoing || target == "all" {
				finalSummariesDestinations, err = getFederationSummaries(federatedServices, outgoing, dataFetcher)
				if err != nil {
					return err
				}
			}
			if target == incoming || target == "all" {
				finalSummariesOrigins, err = getFederationSummaries(federatedServices, incoming, dataFetcher)
				if err != nil {
					return err
				}
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				// encode for json output
				jsonDataDest, _ := json.Marshal(finalSummariesDestinations)
				jsonDataOrigins, _ := json.Marshal(finalSummariesOrigins)
				finalData, err := utils.CombineByteArraysForJSON([][]byte{jsonDataDest, jsonDataOrigins},
					[]string{"destinations", "origins"})
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(finalData, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(finalData))
				}
			} else {
				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				cmd.Println(FormatCurrentCluster(connection))

				if len(finalSummariesDestinations) > 0 {
					cmd.Println(FormatFederationSummary(finalSummariesDestinations, destinations))
				}
				if len(finalSummariesOrigins) > 0 {
					cmd.Println(FormatFederationSummary(finalSummariesOrigins, origins))
				}
			}

			// check to see if we should exit if we are not watching
			if !watchEnabled {
				break
			}
			// we are watching services so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// startFederationCmd represents the start federation command
var startFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "start federation for a service",
	Long: `The 'start federation' command starts federation on a service. There
are various options available using '-M' including:
- with-sync - start after federating all cache entries
- no-backlog - clear any initial backlog and start federating
You may also specify a participant otherwise the command will apply to all participants.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], "start", participant, startMode)
	},
}

// stopFederationCmd represents the stop federation command
var stopFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "stop federation for a service",
	Long: `The 'stop federation' command stops federation on a service. There
You may also specify a participant otherwise the command will apply to all participants.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], "stop", participant, "")
	},
}

// replicateAllCmd represents the replicate all command
var replicateAllCmd = &cobra.Command{
	Use:   "all service-name",
	Short: "initiate a replication of all cache entries for a federated service",
	Long: `The 'replicate all' command replicates all caches for a federated service.
You must specify a participant to replicate for.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], "replicateAll", replicateAllParticipant, "")
	},
}

// pauseFederationCmd represents the pause federation command
var pauseFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "Pause federation for a service",
	Long: `The 'pause' command stops federation on a service.
You may also specify a participant otherwise the command will apply to all participants.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], "pause", participant, "")
	},
}

// getFederationSummaries returns federation summaries for the specified target
func getFederationSummaries(federatedServices []string, target string, dataFetcher fetcher.Fetcher) ([]config.FederationSummary, error) {
	var (
		data                []byte
		err                 error
		federationSummaries = config.FederationSummaries{}
		finalSummaries      []config.FederationSummary
	)
	// retrieve the details for each service
	for _, value := range federatedServices {
		data, err = dataFetcher.GetFederationStatisticsJSON(value, target)
		if err != nil {
			return finalSummaries, err
		}

		if len(data) == 0 {
			return finalSummaries, nil
		}
		err = json.Unmarshal(data, &federationSummaries)
		if err != nil {
			return finalSummaries, utils.GetError("unable to unmarshall federation summary", err)
		}
		// stamp the service name
		for i := range federationSummaries.Services {
			federationSummaries.Services[i].ServiceName = value
		}
		finalSummaries = append(finalSummaries, federationSummaries.Services...)
	}

	return finalSummaries, nil
}

func init() {
	startFederationCmd.Flags().StringVarP(&participant, "participant", "p", "all", participantMessage)
	startFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	startFederationCmd.Flags().StringVarP(&startMode, "start-mode", "M", "",
		"the start mode. Leave blank for normal or specify "+fetcher.WithSync+" or "+fetcher.NoBacklog)

	stopFederationCmd.Flags().StringVarP(&participant, "participant", "p", "all", participantMessage)
	stopFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	replicateAllCmd.Flags().StringVarP(&replicateAllParticipant, "participant", "p", "", participantMessage)
	_ = replicateAllCmd.MarkFlagRequired("participant")
	replicateAllCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	pauseFederationCmd.Flags().StringVarP(&participant, "participant", "p", "all", participantMessage)
	pauseFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}
