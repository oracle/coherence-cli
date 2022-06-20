/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package constants

// FederatedService defines the type for a federated service
const FederatedService = "FederatedCache"

// DistributedService defines the type for a distributed service
const DistributedService = "DistributedCache"

// PagedTopic defines the type for a PagedTopic service
const PagedTopic = "PagedTopic"

// NoOperation defines the no-op message
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
