/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/onsi/gomega"
	"testing"
)

func TestPanelDecoding(t *testing.T) {
	var (
		g      = gomega.NewGomegaWithT(t)
		result []string
	)

	_, err := parseLayout("")
	g.Expect(err).To(gomega.HaveOccurred())

	result, err = parseLayout("one")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(len(result)).To(gomega.Equal(1))
	g.Expect(result[0]).To(gomega.Equal("one"))

	result, err = parseLayout("one:two")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(len(result)).To(gomega.Equal(2))
	g.Expect(result[0]).To(gomega.Equal("one"))
	g.Expect(result[1]).To(gomega.Equal("two"))
	result, err = parseLayout("one:two,three:four")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(len(result)).To(gomega.Equal(3))
	g.Expect(result[0]).To(gomega.Equal("one"))
	g.Expect(result[1]).To(gomega.Equal("two,three"))
	g.Expect(result[2]).To(gomega.Equal("four"))
}

func TestValidatePanels(t *testing.T) {
	var g = gomega.NewGomegaWithT(t)

	err := validatePanels([]string{"members"})
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	err = validatePanels([]string{"membersSummary"})
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	err = validatePanels([]string{"caches"})
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	err = validatePanels([]string{"cluxxxxxxxxster"})
	g.Expect(err).To(gomega.HaveOccurred())
}
