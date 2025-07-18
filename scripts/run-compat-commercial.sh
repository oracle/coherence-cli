#!/bin/bash

#
# Copyright (c) 2021, 2025 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence GE 14.1.2-0-3"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,commercial COHERENCE_VERSION=14.1.2-0-3 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone

# Federation Tests
echo "Coherence Federation Test GE 14.1.2-0-3"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,federation COHERENCE_VERSION=14.1.2-0-3 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence Federation Test 14.1.2-0-3"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,federation COHERENCE_VERSION=14.1.2-0-3 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-federation-images test-e2e-federation

echo "Coherence GE 14.1.2-0-3 with topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,topics-commercial COHERENCE_VERSION=14.1.2-0-3 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-topics

echo "Coherence GE 14.1.2-0-3 with views"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,views COHERENCE_GROUP_ID=com.oracle.coherence COHERENCE_VERSION=14.1.2-0-3 make clean build-view-images test-e2e-views
