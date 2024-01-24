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
	"strings"
	"sync"
	"time"
)

const (
	destinations       = "destinations"
	origins            = "origins"
	outgoing           = "outgoing"
	incoming           = "incoming"
	participantMessage = "participant to apply to"
	supplyService      = "you must provide a service name"
	federationUse      = "federation service-name"
	replicateAll       = "replicateAll"
)

var (
	participant              string
	startMode                string
	replicateAllParticipant  string
	describeFederationType   string
	federationAttributeName  string
	federationAttributeValue string
)

// getFederationCmd represents the get federation command.
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
	ValidArgs: []string{all, destinations, origins},
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
		} else if args[0] == all {
			target = all
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

		for {
			var (
				finalSummariesDestinations []config.FederationSummary
				finalSummariesOrigins      []config.FederationSummary
			)

			if target == outgoing || target == all {
				finalSummariesDestinations, err = getFederationSummaries(federatedServices, outgoing, dataFetcher)
				if err != nil {
					return err
				}
			}
			if target == incoming || target == all {
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
				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))

				if len(finalSummariesDestinations) > 0 {
					cmd.Println(FormatFederationSummary(finalSummariesDestinations, destinations))
				}
				if len(finalSummariesOrigins) > 0 {
					cmd.Println(FormatFederationSummary(finalSummariesOrigins, origins))
				}
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

// startFederationCmd represents the start federation command.
var startFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "start federation for a service",
	Long: `The 'start federation' command starts federation on a service. There
are various options available using '-M' including:
- with-sync - start after federating all cache entries
- no-backlog - clear any initial backlog and start federating
You may also specify a participant otherwise the command will apply to all participants.`,
	ValidArgsFunction: completionFederatedService,
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

// setFederationCmd represents the set federation command.
var setFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "set an attribute for a federated service",
	Long: `The 'set federation' command sets an attribute for a federated service. The
following attribute names are allowed: traceLogging.`,
	ValidArgsFunction: completionFederatedService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], "set", participant, "")
	},
}

// stopFederationCmd represents the stop federation command.
var stopFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "stop federation for a service",
	Long: `The 'stop federation' command stops federation on a service. There
You may also specify a participant otherwise the command will apply to all participants.`,
	ValidArgsFunction: completionFederatedService,
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

// replicateAllCmd represents the replicate all command.
var replicateAllCmd = &cobra.Command{
	Use:   "all service-name",
	Short: "initiate a replication of all cache entries for a federated service",
	Long: `The 'replicate all' command replicates all caches for a federated service.
You must specify a participant to replicate for.`,
	ValidArgsFunction: completionFederatedService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, supplyService)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return IssueFederationCommand(cmd, args[0], replicateAll, replicateAllParticipant, "")
	},
}

// pauseFederationCmd represents the pause federation command.
var pauseFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "Pause federation for a service",
	Long: `The 'pause' command stops federation on a service.
You may also specify a participant otherwise the command will apply to all participants.`,
	ValidArgsFunction: completionFederatedService,
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

// getFederationSummaries returns federation summaries for the specified target.
func getFederationSummaries(federatedServices []string, target string, dataFetcher fetcher.Fetcher) ([]config.FederationSummary, error) {
	var (
		data                []byte
		err                 error
		federationSummaries = config.FederationSummaries{}
		finalSummaries      []config.FederationSummary
	)
	// retrieve the details for each service
	for _, value := range federatedServices {
		data, err = dataFetcher.GetFederationStatistics(value, target)
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

// describeFederationCmd represents the describe federation command.
var describeFederationCmd = &cobra.Command{
	Use:   federationUse,
	Short: "describe federation details for a given service and participant",
	Long: `The 'describe federation' command displays the federation details for a given
service, type and participant. Specify -T to set type outgoing or incoming and -p for participant.`,
	ValidArgsFunction: completionFederatedService,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a service name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err               error
			dataFetcher       fetcher.Fetcher
			connection        string
			federatedServices []string
			service           = args[0]
			nodeIDArray       []string
			errorSink         = createErrorSink()
			byteSink          = createByteArraySink()
			wg                sync.WaitGroup
			output            string
			target            string
		)

		if participant == all {
			return errors.New("please provide a participant")
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

		// validate the federated service is valid
		if !utils.SliceContains(federatedServices, service) {
			return fmt.Errorf(federationServiceMsg, service)
		}

		// validate the type
		if describeFederationType != destinations && describeFederationType != origins {
			return fmt.Errorf("type must be %s or %s", destinations, origins)
		}

		// get all node Id's
		nodeIDArray, err = GetNodeIds(dataFetcher)
		if err != nil {
			return err
		}

		var nodesLen = len(nodeIDArray)
		if nodesLen == 0 {
			return errors.New("no members found")
		}

		// retrieve federation details for all services and members for the participant
		// http://127.0.0.1:8080/management/coherence/cluster/services/{service}/members/{member}/federation/statistics/{outgoing|incoming}/participants/{participant}?links=

		wg.Add(nodesLen)

		if describeFederationType == destinations {
			target = outgoing
		} else {
			target = incoming
		}
		for _, value := range nodeIDArray {
			go func(nodeID string) {
				var (
					err1   error
					result []byte
				)
				defer wg.Done()
				result, err1 = dataFetcher.GetFederationDetails(service, target, nodeID, participant)
				if err1 != nil {
					errorSink.AppendError(err1)
				} else {
					byteSink.AppendByteArray(result)
				}
			}(value)
		}

		wg.Wait()
		errorList := errorSink.GetErrors()

		if len(errorList) > 0 {
			return utils.GetErrors(errorList)
		}

		// now we have the details
		results := byteSink.GetByteArrays()
		numResults := len(results)

		// check to see if all results were empty and this will indicate no matches for participant and type
		found := false
		for _, v := range results {
			if len(v) > 0 {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unable to find participant %s for service %s and type %s", participant, service, describeFederationType)
		}

		if isJSONPathOrJSON() {
			// encode for json output
			finalData := make([]byte, 0)
			finalData = append(finalData, []byte("{ \"items\": [")...)

			for i, v := range byteSink.GetByteArrays() {
				finalData = append(finalData, v...)
				// only append "," if not last entry
				if i < numResults-1 {
					finalData = append(finalData, []byte(",")...)
				}
			}
			finalData = append(finalData, []byte("]}")...)

			if err = processJSONOutput(cmd, finalData); err != nil {
				return err
			}
		} else {
			var sb strings.Builder

			sb.WriteString(FormatCurrentCluster(connection))

			sb.WriteString("\nFEDERATION DETAILS\n")
			sb.WriteString("------------------\n")

			sb.WriteString(fmt.Sprintf("Service:     %s\n", service))
			sb.WriteString(fmt.Sprintf("Type:        %s\n", describeFederationType))
			sb.WriteString(fmt.Sprintf("Participant: %s\n\n", participant))

			if verboseOutput {
				for _, v := range results {
					output, err = FormatJSONForDescribe(v, true,
						"Node Id", "Service", "Name", "Type")
					if err != nil {
						return err
					}
					sb.WriteString(output + "\n")
				}
			} else {
				// not verbose output so unmarshall the original results
				var federationData = make([]config.FederationDescription, 0)
				for _, v := range results {
					if len(v) > 0 {
						var entry = config.FederationDescription{}
						err = json.Unmarshal(v, &entry)
						if err != nil {
							return utils.GetError("unable to unmarshal federation details", err)
						}
						federationData = append(federationData, entry)
					}
				}

				sb.WriteString(FormatFederationDetails(federationData, describeFederationType))
			}

			cmd.Println(sb.String())
		}

		return nil
	},
}

func init() {
	startFederationCmd.Flags().StringVarP(&participant, "participant", "p", all, participantMessage)
	startFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	startFederationCmd.Flags().StringVarP(&startMode, "start-mode", "M", "",
		"the start mode. Leave blank for normal or specify "+fetcher.WithSync+" or "+fetcher.NoBacklog)

	stopFederationCmd.Flags().StringVarP(&participant, "participant", "p", all, participantMessage)
	stopFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	setFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setFederationCmd.Flags().StringVarP(&federationAttributeName, "attribute", "a", "", "attribute name to set")
	_ = setFederationCmd.MarkFlagRequired("attribute")
	setFederationCmd.Flags().StringVarP(&federationAttributeValue, "value", "v", "", "attribute value to set")
	_ = setFederationCmd.MarkFlagRequired("value")

	replicateAllCmd.Flags().StringVarP(&replicateAllParticipant, "participant", "p", "", participantMessage)
	_ = replicateAllCmd.MarkFlagRequired("participant")
	replicateAllCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	pauseFederationCmd.Flags().StringVarP(&participant, "participant", "p", all, participantMessage)
	pauseFederationCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	describeFederationCmd.Flags().StringVarP(&participant, "participant", "p", all, participantMessage)
	describeFederationCmd.Flags().StringVarP(&describeFederationType, "type", "T", outgoing, "type to describe "+outgoing+" or "+incoming)
	describeFederationCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false,
		"include verbose output including all attributes")
}
