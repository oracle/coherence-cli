#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 21.12-SNAPSHOT"
COHERENCE_VERSION=21.12-SNAPSHOT make clean build-test-images test-e2e-standalone

echo "Coherence CE 21.12"
COHERENCE_VERSION=21.12 make clean build-test-images test-e2e-standalone

echo "Coherence CE 21.12 with Executor"
PROFILES=,executor COHERENCE_VERSION=21.12 make clean build-test-images test-e2e-standalone

echo "Coherence CE 21.06.2"
COHERENCE_VERSION=21.06.2 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-7"
COHERENCE_VERSION=14.1.1-0-7 make clean build-test-images test-e2e-standalone