#!/bin/bash

#
# Copyright (c) 2021, 2024 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence GE 14.1.1-0-19"
PROFILES=,commercial COHERENCE_VERSION=14.1.1-0-19 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

echo "Coherence GE 12.2.1-4-23"
PROFILES=,commercial COHERENCE_VERSION=12.2.1-4-23 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

# Federation Tests
echo "Coherence Federation Test GE 14.1.1-0-19"
PROFILES=,federation COHERENCE_VERSION=14.1.1-0-19 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence Federation Test GE 12.2.1-4-23"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,federation COHERENCE_VERSION=12.2.1-4-23 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence Federation Test 14.1.2-0-0-SNAPSHOT"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,federation COHERENCE_VERSION=14.1.2-0-0-SNAPSHOT COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence GE 14.1.2-0-0-SNAPSHOT with topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,topics-commercial COHERENCE_VERSION=14.1.2-0-0-SNAPSHOT COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-topics

echo "Coherence GE 14.1.2-0-0-SNAPSHOT with views"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,views COHERENCE_GROUP_ID=com.oracle.coherence COHERENCE_VERSION=14.1.2-0-0-SNAPSHOT make clean build-view-images test-e2e-views
