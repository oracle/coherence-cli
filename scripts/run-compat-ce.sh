#!/bin/bash

#
# Copyright (c) 2021, 2023 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 22.06.6"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 COHERENCE_VERSION=22.06.6 make clean build-test-images test-e2e-standalone

echo "Coherence CE 23.09.1"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 COHERENCE_VERSION=23.09.1 make clean build-test-images test-e2e-standalone

echo "Coherence CE 23.09.1 with Executor"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,executor COHERENCE_VERSION=23.09.1 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-15"
COHERENCE_VERSION=14.1.1-0-15 make clean build-test-images test-e2e-standalone

echo "Coherence CE 23.09.1 with Topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 PROFILES=,topics COHERENCE_VERSION=23.09.1 make clean build-test-images test-e2e-topics

