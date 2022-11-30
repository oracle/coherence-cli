/*
 * Copyright (c) 2022, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package topics

import (
	"github.com/oracle/coherence-cli/test/common"
	"testing"
)

//
// Run the test suite for topics
//

// RunTestFederationCommands tests federation commands
func TestTopicsCommands(t *testing.T) {
	common.RunTestTopicsCommands(t)
}
