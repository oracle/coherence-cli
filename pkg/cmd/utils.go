/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const federationServiceMsg = "service %s does not exist or is not a federated service"

func displayErrorAndExit(cmd *cobra.Command, message string) {
	_, _ = fmt.Fprintln(os.Stderr, "Error: "+message)
	_, _ = fmt.Fprintln(os.Stderr, "Provide the --help flag to display full help")
	os.Exit(1)
}

// ServiceExists returns true if a service exists
func ServiceExists(dataFetcher fetcher.Fetcher, serviceName string) (bool, error) {
	var (
		servicesSummary = config.ServicesSummaries{}
		serviceResult   []byte
		err             error
	)
	serviceResult, err = dataFetcher.GetServiceDetailsJSON()
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(serviceResult, &servicesSummary)
	if err != nil {
		return false, err
	}

	found := false
	for _, v := range servicesSummary.Services {
		if v.ServiceName == serviceName {
			found = true
			break
		}
	}

	return found, nil
}

// GetListOfCacheServices returns a list of cache services
func GetListOfCacheServices(servicesSummary config.ServicesSummaries) []string {
	var cacheServices = make([]string, 0)
	for _, value := range servicesSummary.Services {
		var serviceName = value.ServiceName
		if (utils.IsDistributedCache(value.ServiceType) ||
			value.ServiceType == "ReplicatedCache") && !utils.SliceContains(cacheServices, serviceName) {

			cacheServices = append(cacheServices, serviceName)
		}
	}

	return cacheServices
}

// GetPersistenceServices returns a list of persistence services
func GetPersistenceServices(dataFetcher fetcher.Fetcher) ([]string, error) {
	servicesSummary, err := GetServices(dataFetcher)
	if err != nil {
		return nil, err
	}

	persistenceServices := make([]string, 0)
	for _, value := range servicesSummary.Services {
		if !utils.SliceContains(persistenceServices, value.ServiceName) && utils.IsDistributedCache(value.ServiceType) {
			persistenceServices = append(persistenceServices, value.ServiceName)
		}
	}

	return persistenceServices, nil
}

// GetServices returns a list of services
func GetServices(dataFetcher fetcher.Fetcher) (config.ServicesSummaries, error) {
	var (
		servicesResult  []byte
		servicesSummary = config.ServicesSummaries{}
		err             error
	)

	servicesResult, err = dataFetcher.GetServiceDetailsJSON()
	if err != nil {
		return servicesSummary, err
	}

	err = json.Unmarshal(servicesResult, &servicesSummary)
	if err != nil {
		return servicesSummary, err
	}

	return servicesSummary, nil
}

// GetFederatedServices returns a list of federated services
func GetFederatedServices(dataFetcher fetcher.Fetcher) ([]string, error) {
	servicesSummary, err := GetServices(dataFetcher)
	if err != nil {
		return nil, err
	}

	federatedServices := make([]string, 0)
	for _, value := range servicesSummary.Services {
		if !utils.SliceContains(federatedServices, value.ServiceName) && value.ServiceType == constants.FederatedService {
			federatedServices = append(federatedServices, value.ServiceName)
		}
	}

	return federatedServices, nil
}

// GetSnapshots returns the snapshots for a service
func GetSnapshots(dataFetcher fetcher.Fetcher, serviceName string) ([]string, error) {
	var coordinator = config.PersistenceCoordinator{}

	coordData, err := dataFetcher.GetPersistenceCoordinator(serviceName)
	if err != nil {
		return constants.EmptyString, err
	}

	err = json.Unmarshal(coordData, &coordinator)
	if err != nil {
		return constants.EmptyString, err
	}

	return coordinator.Snapshots, nil
}

// GetArchivedSnapshots retrieves the archived snapshots for a service
func GetArchivedSnapshots(dataFetcher fetcher.Fetcher, serviceName string) ([]string, error) {
	var (
		archivedData      []byte
		snapshotsArchived = config.Archives{}
		err               error
	)

	archivedData, err = dataFetcher.GetArchivedSnapshots(serviceName)
	if err != nil {
		var errMsg = err.Error()
		// 404 = not found means no snapshots and 400 bad request means no archiver
		if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "400") {
			return constants.EmptyString, nil
		}
		return constants.EmptyString, err
	}

	if len(archivedData) > 0 {
		err = json.Unmarshal(archivedData, &snapshotsArchived)
		if err != nil {
			return constants.EmptyString, err
		}
		return snapshotsArchived.Snapshots, nil
	}
	return constants.EmptyString, err
}

// UnmarshalThreadDump unmarshal a thread dump
func UnmarshalThreadDump(jsonData []byte) (string, error) {
	type threadDump struct {
		State string `json:"state"`
	}

	data := threadDump{}

	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		return "", err
	}

	return data.State, nil
}

// GetMachineList returns a list of machines
func GetMachineList(dataFetcher fetcher.Fetcher) (map[string]string, error) {
	var (
		err           error
		members       = config.Members{}
		membersResult []byte
	)

	membersResult, err = dataFetcher.GetMemberDetailsJSON(false)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(membersResult, &members)
	if err != nil {
		return nil, utils.GetError("unable to decode member details", err)
	}

	// create a list of the unique machine names and one node from the machine to query for details
	machinesMap := make(map[string]string)

	for _, value := range members.Members {
		machineName := value.UnicastAddress
		if _, ok := machinesMap[machineName]; !ok {
			// does not exist to add it
			machinesMap[machineName] = value.NodeID
		}
	}

	return machinesMap, nil
}

// IssueReporterCommand issues a reporter command
func IssueReporterCommand(nodeID, command string, cmd *cobra.Command) error {
	var (
		err             error
		connection      string
		dataFetcher     fetcher.Fetcher
		action          string
		reportersResult []byte
		reporters       = config.Reporters{}
	)

	if !utils.IsValidInt(nodeID) {
		return fmt.Errorf("invalid node id %s", nodeID)
	}

	connection, dataFetcher, err = GetConnectionAndDataFetcher()
	if err != nil {
		return err
	}

	// retrieve the reporter details to check if it is already in the requested state
	// we could not worry about this
	reportersResult, err = dataFetcher.GetReportersJSON()
	if err != nil {
		return err
	}

	err = json.Unmarshal(reportersResult, &reporters)
	if err != nil {
		return err
	}

	// find the reporter
	for _, value := range reporters.Reporters {
		if value.NodeID == nodeID {
			if value.State == "Error" {
				// ignore
			} else {
				if command == "start" && value.State != "Stopped" {
					return fmt.Errorf("the reporter on node %s is already started", nodeID)
				} else if command == "stop" && value.State == "Stopped" {
					return fmt.Errorf("the reporter on node %s is already stopped", nodeID)
				}
			}
		}
	}

	cmd.Println(FormatCurrentCluster(connection))

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to %s the reporter on node %s? (y/n) ",
		command, nodeID)) {
		return nil
	}

	if command == "start" {
		action = "started"
		err = dataFetcher.StartReporter(nodeID)
	} else {
		action = "stopped"
		err = dataFetcher.StopReporter(nodeID)
	}

	if err != nil && strings.Contains(err.Error(), "Not Found") {
		return fmt.Errorf("cannot find a reporter on node %s", nodeID)
	}

	if err != nil {
		return err
	}

	cmd.Printf("Reporter has been "+action+" on node %s\n", nodeID)
	return nil
}

// IssueFederationCommand issues a federation command
func IssueFederationCommand(cmd *cobra.Command, serviceName, command, participant, mode string) error {
	var (
		err                        error
		dataFetcher                fetcher.Fetcher
		connection                 string
		federatedServices          []string
		finalSummariesDestinations []config.FederationSummary
		participants               = make([]string, 0)
		description                string
	)

	if mode != "" && (mode != fetcher.WithSync && mode != fetcher.NoBacklog) {
		return fmt.Errorf("mode must be either blank, " + fetcher.WithSync + " or " + fetcher.NoBacklog)
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

	cmd.Println(FormatCurrentCluster(connection))

	if !utils.SliceContains(federatedServices, serviceName) {
		return fmt.Errorf(federationServiceMsg, serviceName)
	}

	finalSummariesDestinations, err = getFederationSummaries(federatedServices, outgoing, dataFetcher)
	if err != nil {
		return err
	}

	// now we have a service name, check to see we have a valid participant
	found := false

	for _, value := range finalSummariesDestinations {
		participants = append(participants, value.ParticipantName)
		if value.ParticipantName == participant {
			found = true
		}
	}

	if participant != "all" && !found {
		return fmt.Errorf("unable to find participant %s for federated service %s", participant, serviceName)
	}

	description = command
	if command == "start" {
		if startMode != "" {
			description += " (" + startMode + ")"
		}
	}

	// confirm the operation
	if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to %s federation for service %s for participants %v ? (y/n) ",
		description, serviceName, participants)) {
		return nil
	}

	_, err = dataFetcher.InvokeFederationOperation(serviceName, command, participant, startMode)
	if err != nil {
		return err
	}

	cmd.Println(OperationCompleted)

	return nil
}

// GetNodeIds returns the node id for the current cluster
func GetNodeIds(dataFetcher fetcher.Fetcher) ([]string, error) {
	var (
		members       = config.Members{}
		membersResult []byte
		err           error
	)

	membersResult, err = dataFetcher.GetMemberDetailsJSON(false)
	if err != nil {
		return constants.EmptyString, err
	}

	err = json.Unmarshal(membersResult, &members)
	if err != nil {
		return constants.EmptyString, utils.GetError("unable to decode member details", err)
	}

	var nodeIDArray = make([]string, 0)
	for _, value := range members.Members {
		nodeIDArray = append(nodeIDArray, value.NodeID)
	}
	return nodeIDArray, nil
}

// ErrorSink holds errors from multiple go routines
type ErrorSink struct {
	sync.RWMutex
	errors []error
}

// createErrorSync creates an error sync
func createErrorSink() ErrorSink {
	return ErrorSink{
		errors: make([]error, 0),
	}
}

// GetErrors returns the errors for an ErrorSync
func (e *ErrorSink) GetErrors() []error {
	return e.errors
}

// AppendError appends an error
func (e *ErrorSink) AppendError(err error) {
	e.Lock()
	defer e.Unlock()
	e.errors = append(e.errors, err)
}

// ByteArraySink is a thread safe byte array
type ByteArraySink struct {
	sync.RWMutex
	values [][]byte
}

// ByteArray creates a byte array sync
func createByteArraySink() ByteArraySink {
	return ByteArraySink{
		values: make([][]byte, 0),
	}
}

// GetByteArrays returns the values for an GetByteArrays
func (b *ByteArraySink) GetByteArrays() [][]byte {
	return b.values
}

// AppendByteArray appends a byte array
func (b *ByteArraySink) AppendByteArray(bytes []byte) {
	b.Lock()
	defer b.Unlock()
	b.values = append(b.values, bytes)
}

// GetURLContents returns the contents at the given url as a []byte
func GetURLContents(resourceURL string) ([]byte, error) {
	var (
		err       error
		req       *http.Request
		resp      *http.Response
		body      []byte
		buffer    bytes.Buffer
		URL       = url.URL{}
		httpProxy = os.Getenv("HTTP_PROXY")
	)
	cookies, _ := cookiejar.New(nil)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false, MinVersion: tls.VersionTLS12},
	}

	if httpProxy != "" {
		proxy, err := URL.Parse(httpProxy)
		if err != nil {
			return body, errors.New("unable to parse HTTP_PROXY environment variable")
		}
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{Transport: tr,
		Timeout: time.Duration(fetcher.RequestTimeout) * time.Second,
		Jar:     cookies,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	req, err = http.NewRequest("GET", stableURL, bytes.NewBuffer(constants.EmptyByte))
	if err != nil {
		return body, err
	}

	resp, err = client.Do(req)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return body, fmt.Errorf("unable to issue GET to %s: response=%s",
			resourceURL, resp.Status)
	}

	_, err = io.Copy(&buffer, resp.Body)
	if err != nil {
		return body, err
	}

	body = buffer.Bytes()
	return body, nil
}

// validateNodeIDs gets the node id list from the nodeIDArray and validates
func getNodeIDs(nodeIDs string, nodeIDArray []string) ([]string, error) {
	nodeIDList := strings.Split(nodeIDs, ",")
	for _, value := range nodeIDList {
		if !utils.IsValidInt(value) {
			return nodeIDList, fmt.Errorf("invalid value for node id of %s", value)
		}

		if !utils.SliceContains(nodeIDArray, value) {
			return nodeIDList, fmt.Errorf("no node with node id %s exists in this cluster", value)
		}
	}
	return nodeIDList, nil
}
