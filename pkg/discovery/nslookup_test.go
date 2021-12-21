/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package discovery

import (
	"fmt"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

var (
	defaultTimeout int32 = 30
)

func TestInvalidHostIp(t *testing.T) {
	g := NewGomegaWithT(t)

	_, err := Open("host:123:123", defaultTimeout)
	g.Expect(err).To(Not(BeNil()))

	_, err = Open("host:1233f", defaultTimeout)
	g.Expect(err).To(Not(BeNil()))

	_, err = Open("host:-1", defaultTimeout)
	g.Expect(err).To(Not(BeNil()))

	_, err = Open("host:1023", defaultTimeout)
	g.Expect(err).To(Not(BeNil()))

	_, err = Open("host:65536", defaultTimeout)
	g.Expect(err).To(Not(BeNil()))
}

func TestParseResults(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(len(parseResults(""))).To(Equal(0))
	g.Expect(len(parseResults("[123]"))).To(Equal(1))
	g.Expect(len(parseResults("[123, 123]"))).To(Equal(2))
	g.Expect(len(parseResults("[123, 123, 456]"))).To(Equal(3))
	result := parseResults("[A, BB, CCC]")
	g.Expect(len(result)).To(Equal(3))
	for _, v := range result {
		valid := v == "A" || v == "BB" || v == "CCC"
		g.Expect(valid).To(BeTrue())
	}
}

// TestBasicConnection tests a connection to local host + the ENV CLUSTER_PORT
func TestBasicConnection(t *testing.T) {
	var (
		g           = NewGomegaWithT(t)
		clusterPort = os.Getenv("CLUSTER_PORT")
		hostName    = "localhost"
		clusters    []ClusterNSPort
	)

	if clusterPort != "" {
		hostName += ":" + clusterPort
	}
	ns, err := Open(hostName, defaultTimeout)
	g.Expect(err).To(BeNil())

	s, err := ns.Lookup(ClusterNameLookup)
	fmt.Println("result: ", s)
	g.Expect(err).To(BeNil())

	s, err = ns.Lookup(ClusterInfoLookup)
	fmt.Println("result: ", s)
	g.Expect(err).To(BeNil())

	s, err = ns.Lookup(JMXLookup)
	fmt.Println("result: ", s)
	g.Expect(err).To(BeNil())

	s, err = ns.Lookup(NSPrefix + ManagementLookup)
	fmt.Println("result: ", s)
	g.Expect(err).To(BeNil())

	s, err = ns.Lookup("dummy")
	fmt.Println("result: ", s)
	g.Expect(s).To(BeEmpty())
	g.Expect(err).To(BeNil())

	// get the list of discovered clusters and their local NS ports
	clusters, err = ns.DiscoverNameServicePorts()
	g.Expect(err).To(BeNil())
	g.Expect(len(clusters)).To(Equal(3))

	for _, value := range clusters {
		validCluster := value.ClusterName == "cluster1" || value.ClusterName == "cluster2" ||
			value.ClusterName == "cluster3"
		g.Expect(validCluster).To(Equal(true))
		g.Expect(value.HostName).To(Equal("localhost"))
	}

	ns.Close()
}
