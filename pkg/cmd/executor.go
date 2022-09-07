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
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

var (
	executorAttributeName   string
	executorAttributeValue  string
	executorValidAttributes = []string{"traceLogging"}
)

const provideExecutorName = "you must provide an executor name"
const cannotFindExecutors = "unable to find any executors in this cluster"

// getExecutorsCmd represents the get executors command
var getExecutorsCmd = &cobra.Command{
	Use:   "executors",
	Short: "display executors for a cluster",
	Long:  `The 'get executors' command displays the executors for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher     fetcher.Fetcher
			executorsResult []byte
		)

		connection, dataFetcher, err := GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var executors config.Executors

			executors, err = getExecutorDetails(dataFetcher, OutputFormat == constants.TABLE)
			if err != nil {
				return err
			}

			if len(executors.Executors) == 0 {
				if !watchEnabled {
					return nil
				}
				continue
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				// encode the struct so we get the updated fields
				executorsResult, err = json.Marshal(executors)
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(executorsResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else if OutputFormat == constants.JSON {
					cmd.Println(string(executorsResult))
				}
			} else {
				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				cmd.Println(FormatCurrentCluster(connection))
				cmd.Print(FormatExecutors(executors.Executors, true))
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

// describeExecutorCmd represents the describe executor command
var describeExecutorCmd = &cobra.Command{
	Use:   "executor executor-name",
	Short: "describe an executor",
	Long:  `The 'describe executor' command shows information related to a specific executor.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideExecutorName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err            error
			dataFetcher    fetcher.Fetcher
			connection     string
			executors      = config.Executors{}
			finalExecutors = config.Executors{}
			executorData   []byte
			executor       = args[0]
			result         string
		)

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		executors, err = getExecutorDetails(dataFetcher, false)
		if err != nil {
			return err
		}

		if len(executors.Executors) == 0 {
			return errors.New(cannotFindExecutors)
		}

		// filter out any executors without the name
		for _, value := range executors.Executors {
			if value.Name == executor {
				finalExecutors.Executors = append(finalExecutors.Executors, value)
			}
		}

		if len(finalExecutors.Executors) == 0 {
			return fmt.Errorf("unable to find executor with name %s", executor)
		}

		executorData, err = json.Marshal(finalExecutors)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			result, err := utils.GetJSONPathResults(executorData, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(executorData))
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			cmd.Println("EXECUTOR DETAILS")
			cmd.Println("----------------")

			for _, executor := range finalExecutors.Executors {
				executorData, err = json.Marshal(executor)
				if err != nil {
					return err
				}
				result, err = FormatJSONForDescribe(executorData, true, "Name", "Member Id")
				if err != nil {
					return err
				}
				cmd.Println(result)
			}

		}

		return nil
	},
}

// setExecutorCmd represents the set executor command
var setExecutorCmd = &cobra.Command{
	Use:   "executor executor-name",
	Short: "set an executor attribute",
	Long: `The 'set executor' command sets an attribute for a specific executor across
all nodes. The following attribute names are allowed: traceLogging.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide an executor name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err            error
			dataFetcher    fetcher.Fetcher
			executors      = config.Executors{}
			finalExecutors = config.Executors{}
			executor       = args[0]
			response       string
			actualValue    interface{}
		)

		if !utils.SliceContains(executorValidAttributes, executorAttributeName) {
			return fmt.Errorf("attribute name %s is invalid. Please choose one of\n%v",
				executorAttributeName, executorValidAttributes)
		}

		if executorAttributeName == executorValidAttributes[0] {
			if executorAttributeValue != "true" && executorAttributeValue != "false" {
				return fmt.Errorf("value for %s should be true or false", executorAttributeName)
			}
			actualValue, _ = strconv.ParseBool(executorAttributeValue)
		}

		// retrieve the current context or the value from "-c"
		_, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		executors, err = getExecutorDetails(dataFetcher, false)
		if err != nil {
			return err
		}

		if len(executors.Executors) == 0 {
			return errors.New("unable to find any executors in this cluster")
		}

		// filter out any executors without the name
		for _, value := range executors.Executors {
			if value.Name == executor {
				finalExecutors.Executors = append(finalExecutors.Executors, value)
			}
		}

		if len(finalExecutors.Executors) == 0 {
			return fmt.Errorf("unable to find executor with name %s", executor)
		}

		if !automaticallyConfirm {
			cmd.Printf("Are you sure you want to set the value of attribute %s to %s for %s? (y/n) ",
				executorAttributeName, executorAttributeValue, executor)
			_, err = fmt.Scanln(&response)
			if response != "y" || err != nil {
				cmd.Println(constants.NoOperation)
				return nil
			}
		}

		_, err = dataFetcher.SetExecutorAttribute(executor, executorAttributeName, actualValue)
		if err != nil {
			return err
		}
		cmd.Println(OperationCompleted)

		return nil
	},
}

// getExecutorDetails returns the executor details for the cluster
// if summary is true then the data is summarised by name
func getExecutorDetails(dataFetcher fetcher.Fetcher, summary bool) (config.Executors, error) {
	var (
		executorsResult []byte
		executors       = config.Executors{}
		err             error
		emptyResult     = config.Executors{}
	)

	executorsResult, err = dataFetcher.GetExecutorsJSON()
	if err != nil {
		return emptyResult, err
	}

	// return if no executors found
	if len(executorsResult) == 0 {
		return emptyResult, nil
	}

	err = json.Unmarshal(executorsResult, &executors)
	if err != nil {
		return emptyResult, utils.GetError("unable to decode executors details", err)
	}

	// only include a summary of the executors
	if summary {
		finalExecutors := config.Executors{}
		for _, value := range executors.Executors {
			// find the executor in the finalExecutors slice
			index := -1
			for i, v := range finalExecutors.Executors {
				if v.Name == value.Name {
					index = i
				}
			}

			if index == -1 {
				// not found, do append the current executor
				value.MemberCount = 1
				finalExecutors.Executors = append(finalExecutors.Executors, value)
			} else {
				// update the count
				finalExecutors.Executors[index].MemberCount++
				finalExecutors.Executors[index].TasksInProgressCount += value.TasksInProgressCount
				finalExecutors.Executors[index].TasksCompletedCount += value.TasksCompletedCount
				finalExecutors.Executors[index].TasksRejectedCount += value.TasksRejectedCount
			}
		}
		return finalExecutors, nil
	}

	return executors, nil
}

func init() {
	setExecutorCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	setExecutorCmd.Flags().StringVarP(&executorAttributeName, "attribute", "a", "", "attribute name to set")
	_ = setExecutorCmd.MarkFlagRequired("attribute")
	setExecutorCmd.Flags().StringVarP(&executorAttributeValue, "value", "v", "", "attribute value to set")
	_ = setExecutorCmd.MarkFlagRequired("value")
}
