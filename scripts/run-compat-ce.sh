#!/bin/bash

#
# Copyright (c) 2021, 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 22.06.15"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=22.06.15 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.2-0-5"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=14.1.2-0-5 make clean build-test-images test-e2e-standalone

echo "Coherence CE 15.1.1-0-1"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=15.1.1-0-1 make clean build-test-images test-e2e-standalone

echo "Coherence CE 15.1.1-0-1 with Executor"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,executor COHERENCE_VERSION=15.1.1-0-1 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-22"
COHERENCE_VERSION=14.1.1-0-22 make clean build-test-images test-e2e-standalone

echo "Coherence CE 15.1.1-0-1 with Topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,topics COHERENCE_VERSION=15.1.1-0-1 make clean build-test-images test-e2e-topics

echo "Coherence CE 15.1.1-0-1 with View Caches"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,views COHERENCE_VERSION=15.1.1-0-1 make clean build-view-images test-e2e-views

# Security Enabled. Keep the CLI configuration separate from other matrix runs.
export COMPUTERNAME=server1
CERTS_DIR="$(pwd)/test/test_utils/certs"
export COHERENCE_TLS_CERTS_PATH="${CERTS_DIR}/guardians-ca.crt"
export COHERENCE_TLS_CLIENT_CERT="${CERTS_DIR}/star-lord.crt"
export COHERENCE_TLS_CLIENT_KEY="${CERTS_DIR}/star-lord.key"
trap 'make test-cluster-shutdown' EXIT
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=26.07 PROFILES=,secure make clean certs build-test-images test-cluster-startup
make cohctl
./scripts/test-tls.sh
