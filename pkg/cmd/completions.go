/*
 * Copyright (c) 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/spf13/cobra"
)

import (
	"encoding/json"
	"github.com/oracle/coherence-cli/pkg/config"
)

// contains functions for doing custom command completion.

var emptySlice = make([]string, 0)

// completionAllClusters provides a completion function to return all clusters.
func completionAllClusters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getAllClusters(false), cobra.ShellCompDirectiveNoFileComp
}

// completionAllManualClusters provides a completion function to return all clusters that were manually created.
func completionAllManualClusters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getAllClusters(true), cobra.ShellCompDirectiveNoFileComp
}

// completionCaches provides a completion function to return all cache names.
func completionCaches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var (
		cacheSummaries config.CacheSummaries
		caches         = make([]string, 0)
	)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	data, err := dataFetcher.GetCachesSummaryJSONAllServices()
	if err != nil || len(data) == 0 || string(data) == "{}" {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	err = json.Unmarshal(data, &cacheSummaries)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for _, v := range cacheSummaries.Caches {
		caches = append(caches, v.CacheName)
	}

	return caches, cobra.ShellCompDirectiveNoFileComp
}

// completionTopics provides a completion function to return all topic names.
func completionTopics(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var topics = make([]string, 0)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	// get the topics and services
	topicsDetails, err := getTopics(dataFetcher, serviceName)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for _, v := range topicsDetails.Details {
		topics = append(topics, v.TopicName)
	}

	return topics, cobra.ShellCompDirectiveNoFileComp
}

// completionService provides a completion function to return all services in a cluster.
func completionService(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var results = make([]string, 0)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	var servicesSummary = config.ServicesSummaries{}

	servicesResult, err := dataFetcher.GetServiceDetailsJSON()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	err = json.Unmarshal(servicesResult, &servicesSummary)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for _, v := range servicesSummary.Services {
		results = append(results, v.ServiceName)
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

// completionPersistenceService provides a completion function to return all persistence services in a cluster.
func completionPersistenceService(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	servicesResult, err := GetPersistenceServices(dataFetcher)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	return servicesResult, cobra.ShellCompDirectiveNoFileComp
}

// completionFederatedService provides a completion function to return all federated services in a cluster.
func completionElasticData(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{ram, flash}, cobra.ShellCompDirectiveNoFileComp
}

// completionFederatedService provides a completion function to return all federated services in a cluster.
func completionFederatedService(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	federatedServices, err := GetFederatedServices(dataFetcher)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	return federatedServices, cobra.ShellCompDirectiveNoFileComp
}

// completionNodeId provides a completion function to return all node ids in a cluster.
func completionNodeId(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	nodeIDArray, err := GetNodeIds(dataFetcher)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	return nodeIDArray, cobra.ShellCompDirectiveNoFileComp
}

// completionExecutors provides a completion function to return all executor names in a cluster.
func completionExecutors(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var results = make([]string, 0)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	executors, err := getExecutorDetails(dataFetcher, true)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for _, v := range executors.Executors {
		results = append(results, v.Name)
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

// completionExecutors provides a completion function to return all machines in a cluster.
func completionMachines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var results = make([]string, 0)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	machinesMap, err := GetMachineList(dataFetcher)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for k := range machinesMap {
		results = append(results, k)
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

// completionHttpSessions provides a completion function to return all http sessions in a cluster.
func completionHttpSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var (
		results      = make([]string, 0)
		httpSessions = config.HTTPSessionSummaries{}
	)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	sessions, err := dataFetcher.GetHTTPSessionDetailsJSON()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	if len(sessions) > 0 {
		err = json.Unmarshal(sessions, &httpSessions)
	}

	for _, v := range httpSessions.HTTPSessions {
		results = append(results, v.AppID)
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

// completionHttpServers provides a completion function to return all HTTP servers in a cluster.
func completionHttpServers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getProxies(httpString)
}

// completionProxies provides a completion function to return all proxies in a cluster.
func completionProxies(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getProxies(tcpString)
}

func getProxies(protocol string) ([]string, cobra.ShellCompDirective) {
	var (
		proxiesSummary = config.ProxiesSummary{}
		proxies        = make([]string, 0)
	)

	_, dataFetcher, err := GetConnectionAndDataFetcher()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	proxyResults, err := dataFetcher.GetProxySummaryJSON()
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	err = json.Unmarshal(proxyResults, &proxiesSummary)
	if err != nil {
		return emptySlice, cobra.ShellCompDirectiveNoFileComp
	}

	for _, v := range proxiesSummary.Proxies {
		if v.Protocol == protocol {
			proxies = append(proxies, v.ServiceName)
		}
	}

	return proxies, cobra.ShellCompDirectiveNoFileComp
}

// getAllClusters
func getAllClusters(manualOnly bool) []string {
	var clusters = make([]string, 0)

	for _, v := range Config.Clusters {
		if !manualOnly || (manualOnly && v.ManuallyCreated) {
			clusters = append(clusters, v.Name)
		}
	}
	return clusters
}
