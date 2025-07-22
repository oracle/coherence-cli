#!/bin/bash

#
# Copyright (c) 2021, 2025 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 22.06.13"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=22.06.13 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.2-0-3"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=14.1.2-0-3 make clean build-test-images test-e2e-standalone

echo "Coherence CE 25.03.2"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=25.03.2 make clean build-test-images test-e2e-standalone

echo "Coherence CE 25.03.2 with Executor"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,executor COHERENCE_VERSION=25.03.2 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-22"
COHERENCE_VERSION=14.1.1-0-22 make clean build-test-images test-e2e-standalone

echo "Coherence CE 25.03.2 with Topics"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,topics COHERENCE_VERSION=25.03.2 make clean build-test-images test-e2e-topics

echo "Coherence CE 25.03.2 with View Caches"
COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 PROFILES=,views COHERENCE_VERSION=25.03.2 make clean build-view-images test-e2e-views

# Security Enabled
export COMPUTERNAME=server1
export COHERENCE_TLS_CERTS_PATH=`pwd`/test/test_utils/certs/guardians-ca.crt
export COHERENCE_TLS_CLIENT_CERT=`pwd`/test/test_utils/certs/star-lord.crt
export COHERENCE_TLS_CLIENT_KEY=`pwd`/test/test_utils/certs/star-lord.key COHERENCE_BASE_IMAGE=gcr.io/distroless/java17-debian12 COHERENCE_VERSION=24.09 PROFILES=,secure make clean certs build-test-images test-cluster-startup
cohctl set ignore-certs true
cohctl add cluster tls -u https://127.0.0.1:30000/management/coherence/cluster
cohctl get members

