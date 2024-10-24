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
	"time"
)

// getMachinesCmd represents the get machines command.
var getMachinesCmd = &cobra.Command{
	Use:   "machines",
	Short: "display machines for a cluster",
	Long:  `The 'get machines' command displays the machines for a cluster.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		var (
			dataFetcher fetcher.Fetcher
			jsonData    []byte
			connection  string
			err         error
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			var machines []config.Machine

			// create a list of the unique machine names and one node from the machine to query for details
			machinesMap, err := GetMachineList(dataFetcher)
			if err != nil {
				return err
			}

			if OutputFormat != constants.TABLE {
				jsonData, err = getOSJson(machinesMap, dataFetcher)
				if err != nil {
					return err
				}
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) {
				result, err := utils.GetJSONPathResults(jsonData, OutputFormat)
				if err != nil {
					return err
				}
				cmd.Println(result)
			} else if OutputFormat == constants.JSON {
				cmd.Println(string(jsonData))
			} else {
				printWatchHeader(cmd)

				cmd.Println(FormatCurrentCluster(connection))

				machines, err = getMachines(machinesMap, dataFetcher)
				if err != nil {
					return err
				}

				cmd.Print(FormatMachines(machines))
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

// getMachines returns the machines given a map of machine names.
func getMachines(machinesMap map[string]string, dataFetcher fetcher.Fetcher) ([]config.Machine, error) {
	var (
		err      error
		data     []byte
		machine  = config.Machine{}
		machines = make([]config.Machine, 0)
	)
	// now we have the list of machines,  iterate and get the individual details
	for k, v := range machinesMap {
		data, err = dataFetcher.GetMemberOSJson(v)
		if err != nil {
			return machines, err
		}

		if len(data) == 0 {
			continue
		}

		err = json.Unmarshal(data, &machine)
		if err != nil {
			return machines, err
		}
		machine.MachineName = k

		machines = append(machines, machine)
	}

	return machines, nil
}

// getOSJson returns the json for the selected machines in the map.
func getOSJson(machinesMap map[string]string, dataFetcher fetcher.Fetcher) ([]byte, error) {
	var (
		finalGeneric = config.GenericDetails{}
		finalData    []byte
		err          error
	)

	// now we have the list of machines and one node, we iterate and get the individual details
	for k, v := range machinesMap {
		data, err := dataFetcher.GetMemberOSJson(v)
		if err != nil {
			return constants.EmptyByte, err
		}

		generic := make(map[string]interface{})

		if len(data) != 0 {
			err = json.Unmarshal(data, &generic)
			if err != nil {
				return constants.EmptyByte, utils.GetError("unable to unmarshall data", err)
			}
		}

		// add the machine name
		generic["machineName"] = k

		finalGeneric.Details = append(finalGeneric.Details, generic)
	}

	finalData, err = json.Marshal(finalGeneric)
	if err != nil {
		return constants.EmptyByte, err
	}

	return finalData, nil
}

// describeMachineCmd represents the describe machine command.
var describeMachineCmd = &cobra.Command{
	Use:               "machine machine-name",
	Short:             "describe a machine",
	Long:              `The 'describe machine' command shows information related to a particular machine.`,
	ValidArgsFunction: completionMachines,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a single machine name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		type machineEntry struct {
			Detail []interface{} `json:"items"`
		}
		var (
			jsonData    []byte
			newData     []byte
			err         error
			dataFetcher fetcher.Fetcher
			entry       = machineEntry{}
			connection  string
		)

		machineName := args[0]

		// retrieve the current context or the value from "-c"
		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// create a list of the unique machine names and one node from the machine to query for details
		machinesMap, err := GetMachineList(dataFetcher)
		if err != nil {
			return err
		}

		found := false
		newMachineMap := make(map[string]string)
		for k, value := range machinesMap {
			if k == machineName {
				found = true
				newMachineMap[k] = value
				break
			}
		}

		if !found {
			return fmt.Errorf("unable to find machine %s", machineName)
		}

		jsonData, err = getOSJson(newMachineMap, dataFetcher)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			jsonPathResult, err := utils.GetJSONPathResults(jsonData, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(jsonPathResult)
			return nil
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(jsonData))
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			cmd.Println("MACHINE DETAILS")
			cmd.Println("---------------")

			// we need to only get the items node
			err = json.Unmarshal(jsonData, &entry)
			if err != nil {
				return err
			}

			if len(entry.Detail) != 1 {
				return errors.New("unable to decode json entry: " + string(jsonData))
			}

			// re-marshal it so we can only see the details
			newData, err = json.Marshal(entry.Detail[0])
			if err != nil {
				return err
			}

			value, err := FormatJSONForDescribe(newData, true, "Machine Name")
			if err != nil {
				return err
			}
			cmd.Println(value)
		}

		return nil
	},
}
