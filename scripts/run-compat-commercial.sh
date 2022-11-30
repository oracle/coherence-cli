#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence GE 14.1.1-0-7"
PROFILES=,commercial COHERENCE_VERSION=14.1.1-0-7 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

echo "Coherence GE 12.2.1-4-11"
PROFILES=,commercial COHERENCE_VERSION=12.2.1-4-11 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

echo "Coherence GE 14.1.2-0-0-SNAPSHOT"
PROFILES=,commercial COHERENCE_VERSION=14.1.2-0-0-SNAPSHOT COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

echo "Coherence GE 15.1.1-0-0-SNAPSHOT with topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,topics-commercial COHERENCE_VERSION=15.1.1-0-0-SNAPSHOT COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-topics

# Federation Tests
echo "Coherence Federation Test GE 14.1.1-0-7"
PROFILES=,federation COHERENCE_VERSION=14.1.1-0-7 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence Federation Test GE 12.2.1-4-11"
PROFILES=,federation COHERENCE_VERSION=12.2.1-4-11 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence Federation Test 14.1.2-0-0-SNAPSHOT"
PROFILES=,federation COHERENCE_VERSION=14.1.2-0-0-SNAPSHOT COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation


