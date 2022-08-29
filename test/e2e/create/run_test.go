/*
 * Copyright (c) 2022, Oracle and/or its affiliates.
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

// TestRunTestCreateCommands tests create cluster commands
func TestRunTestCreateCommands(t *testing.T) {
	common.RunTestCreateCommands(t)
}
