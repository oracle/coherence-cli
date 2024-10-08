/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package constants

const (
	// FederatedService defines the type for a federated service
	FederatedService = "FederatedCache"

	// DistributedService defines the type for a distributed service
	DistributedService = "DistributedCache"

	// PagedTopic defines the type for a PagedTopic service
	PagedTopic = "PagedTopic"

	// NoOperation defines the no-op message
	NoOperation = "no operation was carried out"

	JSON     = "json"
	TABLE    = "table"
	WIDE     = "wide"
	JSONPATH = "jsonpath="

	RAMJournal   = "ramJournal"
	FlashJournal = "flashJournal"
)

var (
	EmptyByte      = make([]byte, 0)
	EmptyByteArray = make([][]byte, 0)
	EmptyString    = make([]string, 0)
)
