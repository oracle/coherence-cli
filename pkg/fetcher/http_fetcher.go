/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// required to ensure HTTPFetcher implements Fetcher
var (
	_        Fetcher = HTTPFetcher{}
	username string
	password string
)

// HTTPFetcher is an implementation of a Fetcher to retrieve data from Management over REST
type HTTPFetcher struct {
	URL            string
	ConnectionType string
	WebLogicServer bool
	Username       string
	ClusterName    string
}

// GetClusterDetailsJSON returns cluster details in raw json
func (h HTTPFetcher) GetClusterDetailsJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get cluster information", err)
	}
	return result, nil
}

// GetMemberDetailsJSON returns members details in raw json. verbose indicates to
// retrieve all fields rather than selected fields
func (h HTTPFetcher) GetMemberDetailsJSON(verbose bool) ([]byte, error) {
	var fields = ""
	if !verbose {
		// select certain fields otherwise in large clusters fields such as transportStatus
		// can be extremely large and cause performance issues
		fields = "&fields=nodeId,unicastAddress,unicastPort,roleName,memberName,machineName," +
			"rackName,siteName,processName,memoryMaxMB,memoryAvailableMB,receiverSuccessRate," +
			"publisherSuccessRate,tracingSamplingRatio"
	}
	result, err := httpGetRequest(h, "/members/?links="+fields)
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get get members information", err)
	}
	return result, nil
}

// GetSingleMemberDetailsJSON returns a single members details in raw json
func (h HTTPFetcher) GetSingleMemberDetailsJSON(nodeID string) ([]byte, error) {
	result, err := httpGetRequest(h, "/members/"+nodeID+"?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get member information nodeId = "+nodeID, err)
	}
	return result, nil
}

// GetManagementJSON returns the management information
func (h HTTPFetcher) GetManagementJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/management?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get management information", err)
	}
	return result, nil
}

// GetHTTPSessionDetailsJSON returns Coherence*Web Http session details in raw json
func (h HTTPFetcher) GetHTTPSessionDetailsJSON() ([]byte, error) {
	var (
		links          = config.ItemLinks{}
		urls           = make([]string, 0)
		err            error
		result         []byte
		finalResult    []byte
		allResults     = config.GenericDetails{}
		genericDetails = config.GenericDetails{}
	)

	result, err = httpGetRequest(h, "/webApplications")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get Coherence*Web information", err)
	}
	if len(result) == 0 {
		// no http sessions
		return constants.EmptyByte, nil
	}

	err = json.Unmarshal(result, &links)
	if err != nil {
		return constants.EmptyByte, utils.GetError("unable to unmarshall Coherence*Web Links result", err)
	}

	// process each of the links and save the link to each of the web apps
	for _, linkValues := range links.Links {
		for _, linkValue := range linkValues.Links {
			if linkValue.Rel == "self" {
				urls = append(urls, linkValue.Href)
			}
		}
	}

	// now process each web app and get the json for members
	for _, url := range urls {
		result, err = httpGetRequestAbsolute(h, url+"/members/?links=")
		if err != nil {
			return constants.EmptyByte, utils.GetError("unable to retrieve Coherence*Web result", err)
		}

		err = json.Unmarshal(result, &genericDetails)
		if err != nil {
			return constants.EmptyByte, utils.GetError("unable to unmarshall Coherence*Web Links result", err)
		}

		allResults.Details = append(allResults.Details, genericDetails.Details...)
	}

	// convert the object back to JSON
	finalResult, err = json.Marshal(allResults)
	if err != nil {
		return constants.EmptyByte, utils.GetError("unable to marshal Coherence*Web final result", err)
	}

	return finalResult, nil
}

// GetExtendedMemberInfoJSON returns a single members extended info
func (h HTTPFetcher) GetExtendedMemberInfoJSON(result []byte, nodeID string, tokens []string) ([][]byte, error) {
	var (
		links        = config.Links{}
		extendedData = make([][]byte, 0)
		finalNodeID  = nodeID
	)

	type memberName struct {
		MemberName string `json:"memberName"`
	}

	if h.IsWebLogicServer() {
		// unmarshal the result (single member) and retrieve the member name
		var member = memberName{}
		err := json.Unmarshal(result, &member)
		if err != nil {
			return extendedData, utils.GetError("unable to unmarshall member", err)
		}
		finalNodeID = member.MemberName
	}

	result, err := httpGetRequest(h, "/members/"+finalNodeID+"?fields=none")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return extendedData, utils.GetError("unable to get member links nodeId = "+finalNodeID, err)
	}
	if len(result) == 0 {
		return extendedData, nil
	}
	err = json.Unmarshal(result, &links)
	if err != nil {
		return extendedData, utils.GetError("unable to unmarshall extended result", err)
	}

	var newData []byte
	// go through each link and get the data
	for _, value := range links.Links {
		// only fetch the data if the URL link contains at least one of the values in the tokens
		found := false
		for _, token := range tokens {
			if strings.Contains(value.Rel, token) {
				found = true
			}
		}
		if found {
			newData, err = getLinkData(h, "/members/"+finalNodeID+"/"+value.Rel)
			if err != nil && !strings.Contains(err.Error(), "404") {
				return extendedData, utils.GetError("unable to retrieve link data", err)
			}
			extendedData = append(extendedData, newData)
		}
	}
	return extendedData, nil
}

// GetServiceDetailsJSON returns member details in raw json
func (h HTTPFetcher) GetServiceDetailsJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/services/members/?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get service information", err)
	}
	return result, nil
}

// SetMemberAttribute sets the given attribute for a member
func (h HTTPFetcher) SetMemberAttribute(memberID, attribute string, value interface{}) ([]byte, error) {
	var valueString = getJSONValueString(value)

	payload := []byte(fmt.Sprintf("{\"%s\": %s}", attribute, valueString))
	result, err := httpPostRequest(h, "/members/"+memberID, payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError(
			fmt.Sprintf("cannot set value %vfor attribute %s ", value, attribute), err)
	}
	return result, nil
}

// SetManagementAttribute sets the given management attribute for a cluster
func (h HTTPFetcher) SetManagementAttribute(attribute string, value interface{}) ([]byte, error) {
	var valueString = getJSONValueString(value)
	payload := []byte(fmt.Sprintf("{\"%s\": %s}", attribute, valueString))

	result, err := httpPostRequest(h, "/management/", payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError(
			fmt.Sprintf("cannot set management value %vfor attribute %s ", value, attribute), err)
	}
	return result, nil
}

// SetCacheAttribute sets the given attribute for a cache
func (h HTTPFetcher) SetCacheAttribute(memberID, serviceName, cacheName, tier, attribute string, value interface{}) ([]byte, error) {
	var valueString = getJSONValueString(value)
	payload := []byte(fmt.Sprintf("{\"%s\": %s}", attribute, valueString))

	result, err := httpPostRequest(h, "/services/"+getSafeServiceName(h, serviceName)+
		"/caches/"+url.PathEscape(cacheName)+"/members/"+memberID+"?tier="+tier, payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError(
			fmt.Sprintf("cannot set value %v for attribute %s for cache %s/%s and member %s", value, attribute,
				serviceName, cacheName, memberID), err)
	}
	return result, nil
}

// SetServiceAttribute sets the given attribute for a service
func (h HTTPFetcher) SetServiceAttribute(memberID, serviceName, attribute string, value interface{}) ([]byte, error) {
	var valueString = getJSONValueString(value)
	payload := []byte(fmt.Sprintf("{\"%s\": %s}", attribute, valueString))

	result, err := httpPostRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/members/"+memberID, payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError(
			fmt.Sprintf("cannot set value %v for attribute %s for service %s and member %s", value, attribute,
				serviceName, memberID), err)
	}
	return result, nil
}

// GetExecutorsJSON returns executor details in raw json
func (h HTTPFetcher) GetExecutorsJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/executors/members?links=")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get executors information", err)
	}

	if len(result) == 0 {
		return constants.EmptyByte, nil
	}

	return result, nil
}

// GetSingleServiceDetailsJSON returns a single service details in raw json
func (h HTTPFetcher) GetSingleServiceDetailsJSON(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get service information for service "+serviceName, err)
	}
	return result, nil
}

// GetScheduledDistributionsJSON returns scheduled distributions for a service
func (h HTTPFetcher) GetScheduledDistributionsJSON(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+
		"/partition/scheduledDistributions?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get scheduled distributions for service "+serviceName, err)
	}
	return result, nil
}

// GetServicePartitionsJSON returns partition information for a service
func (h HTTPFetcher) GetServicePartitionsJSON(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+
		"/partition/?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get scheduled distributions for service "+serviceName, err)
	}
	return result, nil
}

// GetServiceMembersDetailsJSON returns all the service member details for a service
func (h HTTPFetcher) GetServiceMembersDetailsJSON(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/members/?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get service member information for service "+serviceName, err)
	}
	return result, nil
}

// GetCachesSummaryJSON returns summary caches details for a service
func (h HTTPFetcher) GetCachesSummaryJSON(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/caches?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get caches summary information for service "+serviceName, err)
	}
	return result, nil
}

// GetCachesSummaryJSONAllServices returns summary caches details for all services
func (h HTTPFetcher) GetCachesSummaryJSONAllServices() ([]byte, error) {
	result, err := httpGetRequest(h, "/caches?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get caches summary information", err)
	}
	return result, nil
}

// GetProxySummaryJSON returns proxy server summary
func (h HTTPFetcher) GetProxySummaryJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/services/proxy/members/?links=")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get proxy information", err)
	}
	return result, nil
}

// GetProxyConnectionsJSON returns the proxy connections for the specified service and node
func (h HTTPFetcher) GetProxyConnectionsJSON(serviceName, nodeID string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/members/"+
		nodeID+"/proxy/connections?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get proxy connections for service "+serviceName+
			", nodeId "+nodeID, err)
	}
	return result, nil
}

// GetThreadDump retrieves a thread dump from a member
func (h HTTPFetcher) GetThreadDump(memberID string) ([]byte, error) {
	result, err := httpGetRequest(h, "/members/"+memberID+"/state")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot logMemberState for member "+memberID, err)
	}
	return result, nil
}

// ShutdownMember shuts down a member
func (h HTTPFetcher) ShutdownMember(memberID string) ([]byte, error) {
	result, err := httpPostRequest(h, "/members/"+memberID+"/shutdown", constants.EmptyByte)
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot shutdown member "+memberID, err)
	}
	return result, nil
}

// DumpClusterHeap instructs the cluster to dump the cluster heap for the role.
// role of "all" indicates all members
func (h HTTPFetcher) DumpClusterHeap(role string) ([]byte, error) {
	var (
		payload = constants.EmptyByte
	)
	if role != "all" {
		payload = []byte("{\"role\": \"" + role + "\"}")
	}
	result, err := httpPostRequest(h, "/dumpClusterHeap", payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot dump cluster heap for role "+role, err)
	}
	return result, nil
}

// ConfigureTracing instructs the cluster to configure tracing for the role or all members
func (h HTTPFetcher) ConfigureTracing(role string, tracingRatio float32) ([]byte, error) {
	var (
		payload []byte
	)
	if role == "all" {
		role = ""
	}

	payload = []byte("{\"role\": \"" + role + "\", \"tracingRatio\": " + fmt.Sprintf("%v", tracingRatio) + "}")

	result, err := httpPostRequest(h, "/configureTracing", payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot configureTracing for role "+role, err)
	}
	return result, nil
}

// LogClusterState instructs the cluster to log cluster state for the role
func (h HTTPFetcher) LogClusterState(role string) ([]byte, error) {
	var (
		payload = constants.EmptyByte
	)
	if role != "all" {
		payload = []byte("{\"role\": \"" + role + "\"}")
	}
	result, err := httpPostRequest(h, "/logClusterState", payload)
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot log cluster state for role "+role, err)
	}
	return result, nil
}

// GetCacheMembers retrieves cache member details
func (h HTTPFetcher) GetCacheMembers(serviceName, cacheName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/caches/"+
		url.PathEscape(cacheName)+"/members?links=")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get cache members for service "+serviceName+
			" and cache = "+cacheName, err)
	}
	return result, nil
}

// GetPersistenceCoordinator retrieves persistence coordinator details
func (h HTTPFetcher) GetPersistenceCoordinator(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/persistence?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get persistence coordinator for service "+serviceName, err)
	}
	return result, nil
}

// GetMemberOSJson returns the OS information for the member
func (h HTTPFetcher) GetMemberOSJson(memberID string) ([]byte, error) {
	result, err := httpGetRequest(h, "/members/"+memberID+"/platform/operatingSystem?links=")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get Member OS for member "+memberID, err)
	}
	return result, nil
}

// GetReportersJSON returns reporters in raw json
func (h HTTPFetcher) GetReportersJSON() ([]byte, error) {
	result, err := httpGetRequest(h, "/reporters?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get reporters:", err)
	}
	return result, nil
}

// GetReporterJSON returns reporter for a node in raw json
func (h HTTPFetcher) GetReporterJSON(nodeID string) ([]byte, error) {
	result, err := httpGetRequest(h, "/reporters/"+nodeID+"?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get reporter for node "+nodeID, err)
	}
	return result, nil
}

// StartReporter starts the reporter on a member
func (h HTTPFetcher) StartReporter(nodeID string) error {
	_, err := issueReporterCommand(h, nodeID, "start")
	return err
}

// StopReporter stops the reporter on a member
func (h HTTPFetcher) StopReporter(nodeID string) error {
	_, err := issueReporterCommand(h, nodeID, "stop")
	return err
}

// GetElasticDataDetails retrieves elastic data details for the type of flash or ram
func (h HTTPFetcher) GetElasticDataDetails(journalType string) ([]byte, error) {
	if journalType != "ram" && journalType != "flash" {
		return constants.EmptyByte, errors.New("journal type must be flash or ram")
	}
	result, err := httpGetRequest(h, "/journal/"+journalType+"/members?links=")
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get Journal details for type "+journalType, err)
	}
	if len(result) == 0 {
		return constants.EmptyByte, nil
	}

	return result, nil
}

// GetArchivedSnapshots retrieves the list of archives snapshots
func (h HTTPFetcher) GetArchivedSnapshots(serviceName string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+"/persistence/archives?links=")
	if err != nil {
		return constants.EmptyByte, utils.GetError("cannot get archives for "+serviceName, err)
	}
	return result, nil
}

// InvokeFederationOperation invokes a federation operation against a service and participant
func (h HTTPFetcher) InvokeFederationOperation(serviceName, command, participant, mode string) ([]byte, error) {
	var (
		err error
		url = "/services/" + getSafeServiceName(h, serviceName) + "/federation/"
	)

	if participant != "all" {
		url += "participants/" + getSafeServiceName(h, participant) + "/"
	}

	if command == "start" {
		if mode == WithSync {
			url += "startWithSync"
		} else if mode == NoBacklog {
			url += "startWithNoBacklog"
		} else if mode == "" {
			url += "start"
		} else {
			return constants.EmptyByte, fmt.Errorf("invalid mode of %s", mode)
		}
	} else {
		url += command
	}

	_, err = httpPostRequest(h, url, constants.EmptyByte)
	if err != nil {
		return constants.EmptyByte, fmt.Errorf("unable to perforam %s for service %s", command, serviceName)
	}

	return constants.EmptyByte, nil
}

// InvokeServiceOperation invokes a service operation such as suspend or resume
func (h HTTPFetcher) InvokeServiceOperation(serviceName, operation string) ([]byte, error) {
	var (
		err      error
		finalURL = "/services/" + getSafeServiceName(h, serviceName) + "/"
	)
	if operation == SuspendService {
		finalURL += "suspend"
	} else if operation == ResumeService {
		finalURL += "resume"
	} else {
		return constants.EmptyByte, errors.New("invalid operation " + operation)
	}
	_, err = httpPostRequest(h, finalURL, constants.EmptyByte)
	if err != nil {
		return constants.EmptyByte, fmt.Errorf("unable to %s. %v", operation, err)
	}

	return constants.EmptyByte, nil
}

// InvokeServiceMemberOperation invokes a service operation such as start, stop, shutdown against a node
func (h HTTPFetcher) InvokeServiceMemberOperation(serviceName, nodeID, operation string) ([]byte, error) {
	var (
		err      error
		finalURL = "/services/" + getSafeServiceName(h, serviceName) + "/members/" + nodeID + "/" + operation
	)

	_, err = httpPostRequest(h, finalURL, constants.EmptyByte)
	if err != nil {
		return constants.EmptyByte, fmt.Errorf("unable to invoke %s against service %s and node %s. %v",
			operation, serviceName, nodeID, err)
	}

	return constants.EmptyByte, nil

}

// InvokeSnapshotOperation invokes a snapshot operation against a service
func (h HTTPFetcher) InvokeSnapshotOperation(serviceName, snapshotName, operation string, archived bool) ([]byte, error) {
	var (
		err      error
		finalURL = "/services/" + getSafeServiceName(h, serviceName) + "/persistence/"
	)
	if operation == CreateSnapshot {
		_, err = httpPostRequest(h, finalURL+"snapshots/"+getSafeServiceName(h, snapshotName), constants.EmptyByte)
		return constants.EmptyByte, err
	} else if operation == RemoveSnapshot {
		if archived {
			_, err = httpDeleteRequest(h, finalURL+"archives/"+getSafeServiceName(h, snapshotName))
		} else {
			_, err = httpDeleteRequest(h, finalURL+"snapshots/"+getSafeServiceName(h, snapshotName))
		}
		return constants.EmptyByte, err
	} else if operation == RecoverSnapshot {
		_, err = httpPostRequest(h, finalURL+"snapshots/"+getSafeServiceName(h, snapshotName)+"/recover", constants.EmptyByte)
		return constants.EmptyByte, err
	} else if operation == RetrieveSnapshot {
		_, err = httpPostRequest(h, finalURL+"archives/"+getSafeServiceName(h, snapshotName)+"/retrieve", constants.EmptyByte)
		return constants.EmptyByte, err
	} else if operation == ArchiveSnapshot {
		_, err = httpPostRequest(h, finalURL+"archives/"+getSafeServiceName(h, snapshotName), constants.EmptyByte)
		if err != nil {
			return constants.EmptyByte, fmt.Errorf("unable to archive snapshot. Please ensure you have an archiver setup for your service. %v", err)
		}
		return constants.EmptyByte, err
	} else {
		return constants.EmptyByte, fmt.Errorf("invalid snapshot operation %s", operation)
	}
}

// StartJFR starts a JFR. type is "role", "cluster" or "node" and target is the role or node
func (h HTTPFetcher) StartJFR(jfrName, directory, jfrType, target string, duration int32) ([]byte, error) {
	var (
		err      error
		finalURL = getInitialURL("jfrStart", jfrType, target)
		response []byte
	)

	// append the common options
	options := "name=" + jfrName + ",filename=" + directory
	if duration > 0 {
		options += fmt.Sprintf(",duration=%ds", duration)
	}

	finalURL += "&options=" + url.QueryEscape(options)

	response, err = httpPostRequest(h, finalURL, constants.EmptyByte)
	if err != nil {
		return nil, utils.GetError("unable to start jfr", err)
	}

	return response, nil
}

// DumpJFR dumps a JFR. type is "cluster" or "node" and target is the node id if type "node"
func (h HTTPFetcher) DumpJFR(jfrName, jfrType, target, filename string) ([]byte, error) {
	return jfrOperation(h, jfrName, DumpJFR, jfrType, target, filename)
}

// StopJFR stops a JFR. type is "cluster" or "node" and target is the node id if type "node"
func (h HTTPFetcher) StopJFR(jfrName, jfrType, target string) ([]byte, error) {
	return jfrOperation(h, jfrName, StopJFR, jfrType, target, "")
}

// CheckJFR checks a JFR. type is "cluster" or "node" and target is the node id if type "node"
func (h HTTPFetcher) CheckJFR(jfrName, jfrType, target string) ([]byte, error) {
	return jfrOperation(h, jfrName, CheckJFR, jfrType, target, "")
}

// GetFederationStatisticsJSON returns federation statistics for a service and type
func (h HTTPFetcher) GetFederationStatisticsJSON(serviceName, federationType string) ([]byte, error) {
	result, err := httpGetRequest(h, "/services/"+getSafeServiceName(h, serviceName)+
		"/federation/statistics/"+federationType+"/participants?links=")
	// workaround bug with incoming returning 500 if no federation, ignore 404 as this means no incoming
	if err != nil && !strings.Contains(err.Error(), "500") && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get federation statistics for "+serviceName+" and "+federationType, err)
	}
	if len(result) == 0 {
		return constants.EmptyByte, nil
	}
	return result, nil
}

// jfrOperation issues a jfrStop, jfrDump or jfrCheck. type is "cluster" or "node" and target is the node id if type "node"
func jfrOperation(h HTTPFetcher, jfrName, operation, jfrType, target, filename string) ([]byte, error) {
	var (
		err      error
		finalURL = getInitialURL(operation, jfrType, target)
		response []byte
	)

	finalURL += "&options="
	if jfrName != "" {
		// append the common options
		finalURL += url.QueryEscape("name=" + jfrName)

		if filename != "" {
			finalURL += "," + url.QueryEscape("filename="+filename)
		}
	}

	response, err = httpPostRequest(h, finalURL, constants.EmptyByte)
	if err != nil {
		return nil, utils.GetError("unable to issue"+operation, err)
	}

	return response, nil
}

// getInitialURL returns an initial URL for a JFR operation
func getInitialURL(jfrOperation, jfrType, target string) string {
	finalURL := "/diagnostic-cmd/" + jfrOperation + "?links="
	if jfrType == JfrTypeRole {
		finalURL += "&role=" + url.QueryEscape(target)
	} else if jfrType == JfrTypeNode {
		finalURL = "/members/" + target + "/diagnostic-cmd/" + jfrOperation + "?links="
	}
	return finalURL
}

// issueReporterCommand issues a reporter command for a node
func issueReporterCommand(h HTTPFetcher, nodeID, command string) ([]byte, error) {
	_, err := httpPostRequest(h, "/reporters/"+nodeID+"/"+command, constants.EmptyByte)
	if err != nil {
		return nil, utils.GetError("cannot issue "+command+" reporter on "+nodeID, err)
	}

	return constants.EmptyByte, nil
}

// GetURL returns the URL
func (h HTTPFetcher) GetURL() string {
	return h.ConnectionType
}

// GetType returns the connection type
func (h HTTPFetcher) GetType() string {
	return h.ConnectionType
}

// IsWebLogicServer returns true if the connection is a WebLogic server connection
func (h HTTPFetcher) IsWebLogicServer() bool {
	return h.WebLogicServer
}

// GetUsername returns the username
func (h HTTPFetcher) GetUsername() string {
	return h.Username
}

// GetClusterName returns the cluster name
func (h HTTPFetcher) GetClusterName() string {
	return h.ClusterName
}

// getSafeServiceName returns a safe name with quotes removed if connected to WLS and encoded
func getSafeServiceName(h HTTPFetcher, serviceName string) string {
	if h.IsWebLogicServer() || strings.Contains(serviceName, "\"") {
		serviceName = strings.ReplaceAll(serviceName, "\"", "")
	}
	return url.PathEscape(serviceName)
}

// setUsernamePassword accepts a username and password from the terminal with
// the password not displayed
func setUsernamePassword() error {
	if username == "" {
		fmt.Print("Enter username: ")
		_, _ = fmt.Scanln(&username)
	}

	if username == "" {
		return errors.New("you must enter a username")
	}

	if ReadPassStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if read := scanner.Scan(); !read {
			return errors.New("if you have specified to read password from stdin, you must also provide a username")
		}
		password = scanner.Text()
	} else {
		fmt.Print("Enter password: ")
		passwordByte, err := term.ReadPassword(0)
		if err != nil {
			return err
		}

		password = string(passwordByte)
		fmt.Println()
	}
	return nil
}

// HttpGetRequest issues a HTTP GET request for the given a relative url
func httpGetRequest(h HTTPFetcher, urlAppend string) ([]byte, error) {
	return httpRequest(h, "GET", urlAppend, false, constants.EmptyByte)
}

// httpGetRequestAbsolute issues a HTTP GET request for the given absolute url
func httpGetRequestAbsolute(h HTTPFetcher, urlAppend string) ([]byte, error) {
	return httpRequest(h, "GET", urlAppend, true, constants.EmptyByte)
}

// HttpPostRequest issues a HTTP POST request for the given url
func httpPostRequest(h HTTPFetcher, urlAppend string, body []byte) ([]byte, error) {
	return httpRequest(h, "POST", urlAppend, false, body)
}

// httpDeleteRequest issues a HTTP DELETE request for the given url
func httpDeleteRequest(h HTTPFetcher, urlAppend string) ([]byte, error) {
	return httpRequest(h, "DELETE", urlAppend, false, constants.EmptyByte)
}

// HttpRequest issues a HTTP request for the given url
func httpRequest(h HTTPFetcher, requestType, urlAppend string, absolute bool, content []byte) ([]byte, error) {
	var (
		finalURL        string
		err             error
		req             *http.Request
		resp            *http.Response
		body            []byte
		unsanitizedBody []byte
		buffer          bytes.Buffer
		URL             = url.URL{}
		httpProxy       = os.Getenv("HTTP_PROXY")
		proxy           *url.URL
	)

	// if the username and password was sent in then use it
	if h.Username != "" {
		username = h.Username
	}

	// if using WebLogic Server and no username/password then prompt for
	// Note: In future this may be also if Auth required
	if h.IsWebLogicServer() && (username == "" || password == "") {
		err = setUsernamePassword()
		if err != nil {
			return constants.EmptyByte, err
		}
	}

	if !absolute {
		// For WebLogic server only append the cluster name if it is set.
		// The clusterName will be "" if it is the initial add cluster request
		// we must also check if the user has specified the cluster name already
		// on the URL in the case where there are more than one Coherence
		// clusters in the WebLogic cluster
		if h.IsWebLogicServer() {
			var hasNoClusterOnURL = strings.HasSuffix(h.URL, "/clusters")
			if h.ClusterName != "" && hasNoClusterOnURL {
				// Append the cluster name as there is no cluster on the URL
				finalURL = h.URL + "/" + url.PathEscape(h.ClusterName) + urlAppend
			} else {
				// the cluster name must be on the URL so just set it
				finalURL = h.URL + urlAppend

				// set the cluster to the value on the URL
				if !hasNoClusterOnURL {
					h.ClusterName = h.URL[strings.Index(h.URL, "/clusters")+10 : len(h.URL)]
				}
			}
		} else {
			finalURL = h.URL + urlAppend
		}
	} else {
		finalURL = urlAppend
	}

	var empty = make([]byte, 0)
	start := time.Now()

	cookies, _ := cookiejar.New(nil)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: IgnoreInvalidCerts}, //nolint
	}

	if httpProxy != "" {
		proxy, err = URL.Parse(httpProxy)
		if err != nil {
			return constants.EmptyByte, fmt.Errorf("unable to parse HTTP_PROXY environment variable: %s, %v", httpProxy, err)
		}
		tr.Proxy = http.ProxyURL(proxy)
		Logger.Info("Using HTTP Proxy", []zapcore.Field{zap.String("URL", httpProxy)}...)
	}

	client := &http.Client{Transport: tr,
		Timeout: time.Duration(RequestTimeout) * time.Second,
		Jar:     cookies,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	req, err = http.NewRequest(requestType, finalURL, bytes.NewBuffer(content))
	if err != nil {
		return constants.EmptyByte, err
	}
	if h.IsWebLogicServer() {
		// required for WLS REST requests
		req.Header.Set("X-Requested-By", "Coherence-CLI")

	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}

	if requestType == "POST" && len(content) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err = client.Do(req)
	if err != nil {
		return empty, err
	}

	defer resp.Body.Close()

	_, err = io.Copy(&buffer, resp.Body)
	if err != nil {
		return empty, err
	}

	if resp.StatusCode != 200 {
		return empty, fmt.Errorf("response=%s, url=%s", resp.Status, finalURL)
	}

	body = buffer.Bytes()
	if len(body) > 0 && !isValidJSON(body) {
		return empty, errors.New("invalid JSON body")
	}

	if DebugEnabled {
		fields := []zapcore.Field{
			zap.String("type", requestType),
			zap.String("url", finalURL),
			zap.String("content", string(content)),
			zap.Int("responseCode", resp.StatusCode),
			zap.String("time", fmt.Sprintf("%v", time.Since(start))),
			zap.String("requestTimeout", fmt.Sprintf("%d seconds", RequestTimeout)),
		}
		Logger.Info("Http Request time", fields...)
	}
	if err != nil {
		// always log error
		Logger.Error("Http Request error", []zapcore.Field{
			zap.String("url", finalURL),
			zap.String("content", string(content)),
			zap.Reflect("error", err),
		}...)
		return empty, err
	}

	// WebLogic Server adds extra items nodes when there is no cluster, so we need to unpack
	if h.IsWebLogicServer() && h.ClusterName == "" {
		var result interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			return empty, err
		}

		// result is a map[string]interface{} with first entry
		wlsResult := result.(map[string]interface{})

		if len(wlsResult) == 0 {
			return empty, fmt.Errorf("unable to decode WLS response: %v", result)
		}

		// unpack items, which is an []interface{}
		newBody := wlsResult["items"].([]interface{})

		unsanitizedBody, err = json.Marshal(newBody[0])
		if err != nil {
			return empty, err
		}
		if !isValidJSON(unsanitizedBody) {
			return empty, errors.New("JSON body is invalid")
		}
		body = unsanitizedBody
	}

	return body, err
}

// GetLinkData returns the data from the absolute url
func getLinkData(h HTTPFetcher, url string) ([]byte, error) {
	result, err := httpGetRequest(h, url)
	if err != nil && !strings.Contains(err.Error(), "404") {
		return constants.EmptyByte, utils.GetError("cannot get member links from "+url, err)
	}
	return result, nil
}

// getJSONValueString returns a json representation of a value
func getJSONValueString(value interface{}) string {
	switch value.(type) {
	case string:
		return fmt.Sprintf("\"%v\"", value)
	default:
		// default we are assuming is a number
		return fmt.Sprintf("%v", value)
	}
}

func isValidJSON(data []byte) bool {
	var mapJSON map[string]interface{}
	if err := json.Unmarshal(data, &mapJSON); err != nil {
		return false
	}

	return true
}
