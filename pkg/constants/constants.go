/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package constants

const FederatedService = "FederatedCache"
const DistributedService = "DistributedCache"
const NoOperation = "no operation was carried out"

var (
	EmptyByte   = make([]byte, 0)
	EmptyString = make([]string, 0)
)

const JSON = "json"
const TABLE = "table"
const WIDE = "wide"
const JSONPATH = "jsonpath="

const RAMJournal = "ramJournal"
const FlashJournal = "flashJournal"
