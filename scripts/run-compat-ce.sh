#!/bin/bash

#
# Copyright (c) 2021, 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 22.09-SNAPSHOT"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17 COHERENCE_VERSION=22.09-SNAPSHOT make clean build-test-images test-e2e-standalone

echo "Coherence CE 22.06.2"
COHERENCE_VERSION=22.06.2 make clean build-test-images test-e2e-standalone

echo "Coherence CE 22.06.2 with Executor"
PROFILES=,executor COHERENCE_VERSION=22.06.2 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-10"
COHERENCE_VERSION=14.1.1-0-10 make clean build-test-images test-e2e-standalone
