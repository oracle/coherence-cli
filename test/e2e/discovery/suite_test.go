/*
 * Copyright (c) 2021, 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package standalone

import (
	"fmt"
	"os"
	"testing"
)

// The entry point for the test suite.
func TestMain(m *testing.M) {
	exitCode := m.Run()

	fmt.Printf("Tests completed with return code %d\n", exitCode)

	os.Exit(exitCode)
}
