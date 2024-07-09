/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/config"
	"testing"
)

func TestDecodeMemberDetails(t *testing.T) {
	var (
		result   []config.DepartedMembers
		g        = gomega.NewGomegaWithT(t)
		invalid1 = []string{"rubbish"}
		invalid2 = []string{"Id=4, Timestamp=2024-03-26 08:11:00.537, Address=127.0.0.1:50250, MachineId=10131, Location=machine:localhost,process:6601,member:storage-4, Role=CoherenceServer)"}
		invalid3 = []string{"MemberId=4, Timestamp=2024-03-26 08:11:00.537, Address=127.0.0.1:50250, MachineId=10131, Location=machine:localhost,process:6601,member:storage-4, Role=CoherenceServer)"}
		valid1   = []string{"Member(Id=4, Timestamp=2024-03-26 08:11:00.537, Address=127.0.0.1:50250, MachineId=10131, Location=machine:localhost,process:6601,member:storage-4, Role=CoherenceServer)"}
		valid2   = []string{
			"Member(Id=4, Timestamp=2024-03-26 08:11:00.537, Address=127.0.0.1:50250, MachineId=10131, Location=machine:localhost,process:6601,member:storage-4, Role=CoherenceServer)",
			"Member(Id=3, Timestamp=2024-03-26 08:11:00.536, Address=127.0.0.1:50259, MachineId=10135, Location=machine:localhost,process:6601,member:storage-5, Role=CoherenceServer1)"}
	)

	_, err := decodeDepartedMembers(invalid1)
	g.Expect(err).To(gomega.HaveOccurred())

	_, err = decodeDepartedMembers(invalid2)
	g.Expect(err).To(gomega.HaveOccurred())

	_, err = decodeDepartedMembers(invalid3)
	g.Expect(err).To(gomega.HaveOccurred())

	result, err = decodeDepartedMembers(valid1)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).To(gomega.Not(gomega.BeNil()))
	g.Expect(len(result)).To(gomega.Equal(1))
	g.Expect(result[0].NodeID).To(gomega.Equal("4"))
	g.Expect(result[0].TimeStamp).To(gomega.Equal("2024-03-26 08:11:00.537"))
	g.Expect(result[0].Address).To(gomega.Equal("127.0.0.1:50250"))
	g.Expect(result[0].MachineID).To(gomega.Equal("10131"))
	g.Expect(result[0].Location).To(gomega.Equal("machine:localhost,process:6601,member:storage-4"))
	g.Expect(result[0].Role).To(gomega.Equal("CoherenceServer"))

	result, err = decodeDepartedMembers(valid2)
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).To(gomega.Not(gomega.BeNil()))
	g.Expect(len(result)).To(gomega.Equal(2))
	g.Expect(result[0].NodeID).To(gomega.Equal("4"))
	g.Expect(result[0].TimeStamp).To(gomega.Equal("2024-03-26 08:11:00.537"))
	g.Expect(result[0].Address).To(gomega.Equal("127.0.0.1:50250"))
	g.Expect(result[0].MachineID).To(gomega.Equal("10131"))
	g.Expect(result[0].Location).To(gomega.Equal("machine:localhost,process:6601,member:storage-4"))
	g.Expect(result[0].Role).To(gomega.Equal("CoherenceServer"))

	g.Expect(result[1].NodeID).To(gomega.Equal("3"))
	g.Expect(result[1].TimeStamp).To(gomega.Equal("2024-03-26 08:11:00.536"))
	g.Expect(result[1].Address).To(gomega.Equal("127.0.0.1:50259"))
	g.Expect(result[1].MachineID).To(gomega.Equal("10135"))
	g.Expect(result[1].Location).To(gomega.Equal("machine:localhost,process:6601,member:storage-5"))
	g.Expect(result[1].Role).To(gomega.Equal("CoherenceServer1"))
}

func TestParseHealthEndpoints(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	_, err := parseHealthEndpoints("")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred())) // special empty case

	_, err = parseHealthEndpoints("rubbish")
	g.Expect(err).To(gomega.HaveOccurred())

	values, err := parseHealthEndpoints("http://127.0.0.1:7767")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(len(values)).To(gomega.Equal(1))
	g.Expect(values).To(gomega.Equal([]string{"http://127.0.0.1:7767"}))

	_, err = parseHealthEndpoints("http://127.0.0.1:7767,3333")
	g.Expect(err).To(gomega.HaveOccurred())

	values, err = parseHealthEndpoints("http://127.0.0.1:7767,http://127.0.0.1:7768")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(len(values)).To(gomega.Equal(2))
	g.Expect(values).To(gomega.Equal([]string{"http://127.0.0.1:7767", "http://127.0.0.1:7768"}))
}

func TestGetHealthEndpoint(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(getHealthEndpoint("http://127.0.0.1:7767", "live")).To(gomega.Equal("http://127.0.0.1:7767/live"))
	g.Expect(getHealthEndpoint("http://127.0.0.1:7767/", "live")).To(gomega.Equal("http://127.0.0.1:7767/live"))
}
