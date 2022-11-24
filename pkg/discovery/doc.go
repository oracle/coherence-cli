/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

/*
Package discovery provides an implementation of Coherence NSLookup.

Example:

	// open a connection to the default NameService port of 7574.
	ns, err := discovery.Open("127.0.0.1:7574", 5)
	if err != nil {
	    log.Fatal(err)
	}
	defer ns.Close()

	// return the cluster name of the cluster on this port
	clusterName, err = ns.Lookup(discovery.ClusterNameLookup)
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Println("Cluster name is", clusterName)

	// lookup other information such as management ports, metrics ports, etc
	var discoveredCluster discovery.DiscoveredCluster
	discoveredCluster, err = ns.DiscoverClusterInfo()
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Println("Management URLS", discoveredCluster.ManagementURLs)

	// lookup any foreign clusters also registered with this port
	clusterNames, err = ns.Lookup(discovery.NSPrefix + discovery.ClusterForeignLookup)
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Println("Foreign clusters are", clusterNames)

*/
package discovery
