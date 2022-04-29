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
	"strings"
	"sync"
	"time"
)

var (
	// automaticallyConfirm will automatically confirm and operation if true
	automaticallyConfirm bool

	// ArchivedSnapshots indicates if we are working with archived snapshots
	ArchivedSnapshots bool

	archiveMessage = "if true, returns archived snapshots, otherwise local snapshots"
)

const provideSnapshot = "you must provide a snapshot name"
const snapshotUse = "snapshot snapshot-name"

// createSnapshotCmd represents the create snapshot command
var createSnapshotCmd = &cobra.Command{
	Use:   snapshotUse,
	Short: "create a snapshot for a given service",
	Long: `The 'create snapshot' command creates a snapshot for a given service. If you 
do not specify the -y option you will be prompted to confirm the operation.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideSnapshot)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return invokePersistenceOperation(cmd, fetcher.CreateSnapshot, args[0], false)
	},
}

// getSnapshotsCmd represents the get snapshots command
var getSnapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "display snapshots for a cluster",
	Long: `The 'get snapshots' command displays snapshots for a cluster. If 
no service name is specified then all services are queried. By default 
local snapshots are shown, but you can use the -a option to show archived snapshots.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err         error
			connection  string
			data        []byte
			dataFetcher fetcher.Fetcher
			snapshots   = make([]config.Snapshots, 0)
			wg          sync.WaitGroup
			errorSink   = createErrorSink()
			m           = sync.RWMutex{}
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		// get the services
		servicesResult, err := GetPersistenceServices(dataFetcher)
		if err != nil {
			return err
		}

		// if a service was specified then validate
		if serviceName != "" {
			if !utils.SliceContains(servicesResult, serviceName) {
				return fmt.Errorf("cannot find service named %s", serviceName)
			}
			servicesResult = []string{serviceName}
		}

		// retrieve the snapshot list concurrently for each service
		wg.Add(len(servicesResult))
		for _, service := range servicesResult {
			go func(serviceNameValue string) {
				defer wg.Done()
				var (
					snapshotList []string
					coordData    []byte
					coordinator  = config.PersistenceCoordinator{}
				)
				newSnapshots := make([]string, 0)
				if ArchivedSnapshots {
					snapshotList, err = GetArchivedSnapshots(dataFetcher, serviceNameValue)
					if err != nil {
						errorSink.AppendError(err)
						return
					}
					newSnapshots = append(newSnapshots, snapshotList...)
				} else {
					coordData, err = dataFetcher.GetPersistenceCoordinator(serviceNameValue)
					if err != nil {
						errorSink.AppendError(err)
						return
					}

					err = json.Unmarshal(coordData, &coordinator)
					if err != nil {
						errorSink.AppendError(err)
						return
					}

					newSnapshots = append(newSnapshots, coordinator.Snapshots...)
				}

				// protect the slice for update
				m.Lock()
				defer m.Unlock()
				snapshots = append(snapshots, config.Snapshots{ServiceName: serviceNameValue, Snapshots: newSnapshots})
			}(service)
		}

		// wait for the results
		wg.Wait()
		errorList := errorSink.GetErrors()

		if len(errorList) > 0 {
			return utils.GetErrors(errorList)
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			data, err = json.Marshal(snapshots)
			if err != nil {
				return err
			}
			result, err := utils.GetJSONPathResults(data, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(result)
		} else if OutputFormat == constants.JSON {
			data, err = json.Marshal(snapshots)
			if err != nil {
				return err
			}
			cmd.Println(string(data))
		} else {
			cmd.Println(FormatCurrentCluster(connection))
			for {
				if watchEnabled {
					cmd.Println("\n" + time.Now().String())
				}

				cmd.Println(FormatSnapshots(snapshots, ArchivedSnapshots))

				// check to see if we should exit if we are not watching
				if !watchEnabled {
					break
				}
				// we are watching so sleep and then repeat until CTRL-C
				time.Sleep(time.Duration(watchDelay) * time.Second)
			}
		}

		return nil
	},
}

// recoverSnapshotCmd represents the recover snapshot command
var recoverSnapshotCmd = &cobra.Command{
	Use:   snapshotUse,
	Short: "recover a snapshot for a given service",
	Long: `The 'recover snapshot' command recovers a snapshot for a given service. 
WARNING: Issuing this command will destroy all service data and replaced with the
data from the requested snapshot.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideSnapshot)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return invokePersistenceOperation(cmd, fetcher.RecoverSnapshot, args[0], true)
	},
}

// removeSnapshotCmd represents the remove snapshot command
var removeSnapshotCmd = &cobra.Command{
	Use:   snapshotUse,
	Short: "remove a snapshot for a given service",
	Long: `The 'remove snapshot' command removes a snapshot for a given service. 
By default local snapshots are removed, but you can use the -a option to remove archived snapshots.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideSnapshot)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return invokePersistenceOperation(cmd, fetcher.RemoveSnapshot, args[0], true)
	},
}

// archiveSnapshotCmd represents the archive snapshot command
var archiveSnapshotCmd = &cobra.Command{
	Use:   snapshotUse,
	Short: "archive a snapshot for a given service",
	Long: `The 'archive snapshot' command archives a snapshot for a given service. You must
have an archiver setup on the service for this to be successful.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideSnapshot)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return invokePersistenceOperation(cmd, fetcher.ArchiveSnapshot, args[0], true)
	},
}

// retrieveSnapshotCmd represents the retrieve snapshot command
var retrieveSnapshotCmd = &cobra.Command{
	Use:   snapshotUse,
	Short: "retrieve an archived snapshot for a given service",
	Long:  `The 'retrieve snapshot' command retrieves an archived snapshot for a given service.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideSnapshot)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ArchivedSnapshots = true
		return invokePersistenceOperation(cmd, fetcher.RetrieveSnapshot, args[0], true)
	},
}

func init() {
	setPersistenceFlags(createSnapshotCmd)
	setPersistenceFlags(recoverSnapshotCmd)
	setPersistenceFlags(archiveSnapshotCmd)
	setPersistenceFlags(retrieveSnapshotCmd)

	getSnapshotsCmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	getSnapshotsCmd.Flags().BoolVarP(&ArchivedSnapshots, "archived", "a", false, archiveMessage)

	setPersistenceFlags(removeSnapshotCmd)
	removeSnapshotCmd.Flags().BoolVarP(&ArchivedSnapshots, "archived", "a", false, archiveMessage)
}

// setPersistenceFlags sets common flags for persistence operations
func setPersistenceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&serviceName, serviceNameOption, serviceNameOptionShort, "", serviceNameDescription)
	_ = cmd.MarkFlagRequired(serviceNameOption)
	cmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}

// invokePersistenceOperation invokes a persistence operation
func invokePersistenceOperation(cmd *cobra.Command, operation, snapshotName string, mustExist bool) error {
	var (
		err            error
		connection     string
		result         []byte
		snapshotList   []string
		localSnapshots []string
		dataFetcher    fetcher.Fetcher
		response       string
		msg            = operation
		prefix         = "a"
		servicesResult []string
	)

	if snapshotName == "" {
		return errors.New("you must supply a snapshot name")
	}

	snapshotName = utils.SanitizeSnapshotName(snapshotName)

	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	// get the services
	servicesResult, err = GetPersistenceServices(dataFetcher)
	if err != nil {
		return err
	}

	if serviceName == "" {
		return errors.New("you must supply a service name")
	}
	// if a service was specified then validate
	if !utils.SliceContains(servicesResult, serviceName) {
		return fmt.Errorf("cannot find persistence service named %s", serviceName)
	}

	// depending upon the operation, check if the snapshot exists
	if ArchivedSnapshots {
		snapshotList, err = GetArchivedSnapshots(dataFetcher, serviceName)
		prefix = "an archived"
	} else {
		snapshotList, err = GetSnapshots(dataFetcher, serviceName)
	}
	if err != nil {
		return err
	}

	if operation == fetcher.RetrieveSnapshot {
		// do extra check to ensure a local snapshot does not exist if we are trying to retrieve
		localSnapshots, err = GetSnapshots(dataFetcher, serviceName)
		if err != nil {
			return err
		}
		if utils.SliceContains(localSnapshots, snapshotName) {
			return fmt.Errorf("a local snapshot named %s exists. You must remove if before you can retrieve", snapshotName)
		}
	}

	if !mustExist && utils.SliceContains(snapshotList, snapshotName) {
		return fmt.Errorf("%s snapshot named %s already exists for service %s", prefix, snapshotName, serviceName)
	}

	if mustExist && !utils.SliceContains(snapshotList, snapshotName) {
		return fmt.Errorf("%s snapshot named %s does not exist for service %s", prefix, snapshotName, serviceName)
	}

	if ArchivedSnapshots {
		if operation == fetcher.RemoveSnapshot {
			msg = fetcher.RemoveArchivedSnapshot
		} else if operation == fetcher.RetrieveSnapshot {
			msg = fetcher.RetrieveSnapshot
		}
	}

	cmd.Println(FormatCurrentCluster(connection))

	// confirmation
	if !automaticallyConfirm {
		cmd.Printf("Are you sure you want to perform %s for snapshot %s and service %s? (y/n) ",
			msg, snapshotName, serviceName)
		_, err = fmt.Scanln(&response)
		if response != "y" || err != nil {
			cmd.Println(constants.NoOperation)
			return nil
		}
	}

	result, err = dataFetcher.InvokeSnapshotOperation(serviceName, snapshotName, operation, ArchivedSnapshots)
	if err != nil {
		return utils.GetError(fmt.Sprintf("unable to carry out %s on service %s with snapshot %s",
			msg, serviceName, snapshotName), err)
	}

	var resultString = ""
	if len(result) > 0 {
		resultString = string(result)
	}

	cmd.Printf("Operation %s for snapshot %s on service %s invoked %s\n",
		msg, snapshotName, serviceName, resultString)
	cmd.Println("Please use 'cohctl get persistence' to check for idle status to ensure the operation completed")

	return nil
}
