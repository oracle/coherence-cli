/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package federation

import (
	"github.com/oracle/coherence-cli/test/common"
	"testing"
)

//
// Run the test suite against federated clusters
//

// RunTestFederationCommands tests federation commands.
func TestFederationCommands(t *testing.T) {
	common.RunTestFederationCommands(t)
}
