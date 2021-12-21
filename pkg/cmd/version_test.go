/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestVersionUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(isVersionUpdateAvailable("1.0.0", "1.0.1")).To(Equal(true))
	g.Expect(isVersionUpdateAvailable("1.0.0-RC1", "1.0.0")).To(Equal(true))
	g.Expect(isVersionUpdateAvailable("1.0.1-RC1", "1.0.0")).To(Equal(true))
}
