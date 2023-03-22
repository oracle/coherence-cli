/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package standalone

import (
	"github.com/oracle/coherence-cli/test/common"
	"testing"
)

//
// Run the test suite against a standalone Coherence Cluster
//

// TestDiscoverClustersCommands tests discover clusters commands.
func TestDiscoverClustersCommands(t *testing.T) {
	common.RunTestDiscoverClustersCommands(t)
}

// TestNSLookupCommands tests nslookup commands.
func TestNSLookupCommands(t *testing.T) {
	common.RunTestNSLookupCommands(t)
}
