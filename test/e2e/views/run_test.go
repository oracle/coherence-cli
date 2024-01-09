/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package views

import (
	"github.com/oracle/coherence-cli/test/common"
	"testing"
)

//
// Run the test suite for views
//

// TestViewCacheCommands tests views commands.
func TestViewCacheCommands(t *testing.T) {
	common.RunTestViewCacheCommands(t)
}
