/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
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

// getHTTPSessionsCmd represents the get http-sessions command.
var getHTTPSessionsCmd = &cobra.Command{
	Use:   "http-sessions",
	Short: "display Coherence*Web Http session information for a cluster",
	Long:  `The 'get http-sessions' command displays Coherence*Web Http session information for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			jsonResult   string
			httpSessions = config.HTTPSessionSummaries{}
			dataFetcher  fetcher.Fetcher
			connection   string
			err          error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			printWatchHeader(cmd)

			results, err := dataFetcher.GetHTTPSessionDetailsJSON()
			if err != nil {
				return err
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				jsonResult, err = utils.GetJSONPathResults(results, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(jsonResult)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(results))
			} else {
				cmd.Println(FormatCurrentCluster(connection))

				if len(results) > 0 {
					err = json.Unmarshal(results, &httpSessions)
				}
				if err != nil {
					return utils.GetError("unable to decode Coherence*Web details", err)
				}

				cmd.Print(FormatHTTPSessions(DeduplicateSessions(httpSessions), true))
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

// DeduplicateSessions removes duplicated http session details.
func DeduplicateSessions(httpSummary config.HTTPSessionSummaries) []config.HTTPSessionSummary {
	// the current results include 1 entry for each http session and member, so we need to remove duplicates
	var finalSessions = make([]config.HTTPSessionSummary, 0)

	for _, value := range httpSummary.HTTPSessions {
		// check to see if this service and member already exists in the finalServices
		if len(finalSessions) == 0 {
			// no entries so add it anyway
			value.MemberCount = 1
			value.SessionAverageTotal = int64(value.SessionAverageSize)
			value.TotalReapDuration = value.AverageReapDuration
			finalSessions = append(finalSessions, value)
		} else {
			var foundIndex = -1
			for i, v := range finalSessions {
				if v.AppID == value.AppID {
					foundIndex = i
					break
				}
			}
			if foundIndex >= 0 {
				// update the existing service
				finalSessions[foundIndex].MemberCount++
				finalSessions[foundIndex].ReapedSessionsTotal += value.ReapedSessionsTotal
				finalSessions[foundIndex].TotalReapDuration += value.AverageReapDuration
				finalSessions[foundIndex].SessionUpdates += value.SessionUpdates
				finalSessions[foundIndex].SessionAverageTotal += int64(value.SessionAverageSize)
			} else {
				// new service
				value.MemberCount = 1
				value.SessionAverageTotal = int64(value.SessionAverageSize)
				value.TotalReapDuration = value.AverageReapDuration
				finalSessions = append(finalSessions, value)
			}
		}
	}

	// work out any averages
	var count = len(finalSessions)
	if count > 0 {
		for i := range finalSessions {
			memberCount := int64(finalSessions[i].MemberCount)
			finalSessions[i].AverageReapDuration = finalSessions[i].TotalReapDuration / memberCount
			finalSessions[i].SessionAverageSize = int32(finalSessions[i].SessionAverageTotal / memberCount)
		}
	}
	return finalSessions
}

// describeHTTPSessionCmd represents the describe http-session command.
var describeHTTPSessionCmd = &cobra.Command{
	Use:               "http-session application-id",
	Short:             "describe a http session",
	Long:              `The 'describe http-session' command shows information related to a specific Coherence*Web application.`,
	ValidArgsFunction: completionHTTPSessions,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a single application id")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			found       bool
			err         error
			dataFetcher fetcher.Fetcher
			firstMember []byte
			connection  string
		)

		applicationID := args[0]

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var (
				httpSessions  = config.HTTPSessionSummaries{}
				finalSessions = make([]config.HTTPSessionSummary, 0)
			)

			// check the application id exists
			results, err := dataFetcher.GetHTTPSessionDetailsJSON()
			if err != nil {
				return err
			}

			err = json.Unmarshal(results, &httpSessions)
			if err != nil {
				return utils.GetError("unable to decode Coherence*Web details", err)
			}

			for _, value := range httpSessions.HTTPSessions {
				if value.AppID == applicationID {
					found = true
					if len(firstMember) == 0 {
						// save the first member to format
						firstMember, err = json.Marshal(value)
						if err != nil {
							return err
						}
					}
					finalSessions = append(finalSessions, value)
				}
			}

			if !found {
				return fmt.Errorf("unable to find application id %s", applicationID)
			}

			if isJSONPathOrJSON() {
				return processJSONOutput(cmd, results)
			}

			printWatchHeader(cmd)
			cmd.Println(FormatCurrentCluster(connection))

			cmd.Println("HTTP SESSION DETAILS")
			cmd.Println("--------------------")
			value, err := FormatJSONForDescribe(firstMember, false, "App Id", "Type",
				"Session Cache Name", "Overflow Cache Name")
			if err != nil {
				return err
			}
			cmd.Println(value)
			cmd.Print(FormatHTTPSessions(finalSessions, false))

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
