/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

// ResetTableSorting resets the table sort and descending flags for tests.
func ResetTableSorting() {
	tableSorting = ""
	descendingFlag = false
}
