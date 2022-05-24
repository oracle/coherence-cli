/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	. "github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/discovery"
	"testing"
)

func TestValidateTimeout(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(validateTimeout(1)).To(BeNil())
	g.Expect(validateTimeout(0)).To(Not(BeNil()))
	g.Expect(validateTimeout(-1)).To(Not(BeNil()))
}

func TestSanitizeConnectionName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(sanitizeConnectionName("$TIM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName(",TIM ")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T'IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T(IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T)IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T\"IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T[IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T]IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T\\IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T$IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T#IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T@IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T/IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T;IM")).To(Equal("TIM"))
	g.Expect(sanitizeConnectionName("T!IM")).To(Equal("TIM"))
}

func TestFormatCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	cluster := discovery.DiscoveredCluster{ClusterName: "tim", Host: "localhost", NSPort: 7574}
	g.Expect(formatCluster(cluster)).To(Equal("Cluster: tim, Name Service address: localhost:7574\n"))
}
