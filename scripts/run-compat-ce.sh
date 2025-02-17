#!/bin/bash

#
# Copyright (c) 2021, 2024 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run compatability tests
set -e

echo "Coherence CE 22.06.10"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk COHERENCE_VERSION=22.06.10 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.2-0-0"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk COHERENCE_VERSION=14.1.2-0-0 make clean build-test-images test-e2e-standalone

echo "Coherence CE 24.09.3"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk COHERENCE_VERSION=24.09.3 make clean build-test-images test-e2e-standalone

echo "Coherence CE 24.09 with Executor"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk PROFILES=,executor COHERENCE_VERSION=24.09 make clean build-test-images test-e2e-standalone

echo "Coherence CE 14.1.1-0-19"
COHERENCE_VERSION=14.1.1-0-19 make clean build-test-images test-e2e-standalone

echo "Coherence CE 24.09.3 with Topics"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk PROFILES=,topics COHERENCE_VERSION=24.09.3 make clean build-test-images test-e2e-topics

echo "Coherence CE 24.09.3 with View Caches"
COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk PROFILES=,views COHERENCE_VERSION=24.09.3 make clean build-view-images test-e2e-views

# Security Enabled
export COMPUTERNAME=server1
export COHERENCE_TLS_CERTS_PATH=`pwd`/test/test_utils/certs/guardians-ca.crt
export COHERENCE_TLS_CLIENT_CERT=`pwd`/test/test_utils/certs/star-lord.crt
export COHERENCE_TLS_CLIENT_KEY=`pwd`/test/test_utils/certs/star-lord.key COHERENCE_BASE_IMAGE=arm64v8/openjdk:17-jdk COHERENCE_VERSION=24.09 PROFILES=,secure make clean certs build-test-images test-cluster-startup
cohctl set ignore-certs true
cohctl add cluster tls -u https://127.0.0.1:30000/management/coherence/cluster
cohctl get members

