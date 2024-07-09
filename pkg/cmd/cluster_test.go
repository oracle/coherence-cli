/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/onsi/gomega"
	"github.com/oracle/coherence-go-client/coherence/discovery"
	"os"
	"testing"
)

func TestValidateTimeout(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(validateTimeout(1)).To(gomega.BeNil())
	g.Expect(validateTimeout(0)).To(gomega.Not(gomega.BeNil()))
	g.Expect(validateTimeout(-1)).To(gomega.Not(gomega.BeNil()))
}

func TestSanitizeConnectionName(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(sanitizeConnectionName("$TIM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName(",TIM ")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T'IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T(IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T)IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T\"IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T[IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T]IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T\\IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T$IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T#IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T@IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T/IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T;IM")).To(gomega.Equal("TIM"))
	g.Expect(sanitizeConnectionName("T!IM")).To(gomega.Equal("TIM"))
}

func TestFormatCluster(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	cluster := discovery.DiscoveredCluster{ClusterName: "tim", Host: "localhost", NSPort: 7574}
	g.Expect(formatCluster(cluster)).To(gomega.Equal("Cluster: tim, Name Service address: localhost:7574\n"))
}

func TestGetMavenClasspath(t *testing.T) {
	var (
		g  = gomega.NewGomegaWithT(t)
		ce = "com.oracle.coherence.ce"
	)
	home, _ := os.UserHomeDir()

	path, err := getMavenClasspath(ce, "coherence", "22.06", fileTypeJar)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(path).To(gomega.Equal(home + "/.m2/repository/com/oracle/coherence/ce/coherence/22.06/coherence-22.06.jar"))

	path, err = getMavenClasspath(ce, "coherence", "22.09", fileTypeJar)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(path).To(gomega.Equal(home + "/.m2/repository/com/oracle/coherence/ce/coherence/22.09/coherence-22.09.jar"))

	path, err = getMavenClasspath(ce, "coherence", "22.09", fileTypePom)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(path).To(gomega.Equal(home + "/.m2/repository/com/oracle/coherence/ce/coherence/22.09/coherence-22.09.pom"))
}
