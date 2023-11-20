#!/bin/bash

#
# Copyright (c) 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run a stress test on the CLI.
# It is assumed that a cluster with http management on localhost:30000 is running

set -e

if [ $# -ne 3 ] ; then
  echo "Usage: $0 cohctl-binary iterations build-output"
  exit 1
fi

COHCTL=$1
ITERS=$2
BUILD_OUTPUT=$3

CONFIG_DIR=${BUILD_OUTPUT}/config
mkdir -p ${CONFIG_DIR}
#trap "rm -rf $CONFIG_DIR" 0 1 2 3

echo "Redirecting output to $BUILD_OUTPUT/stress.log..."
exec > $BUILD_OUTPUT/stress.log 2>&1

mkdir -p $CONFIG_DIR

# Add some data
curl http://localhost:8080/populate

ARGS="--config-dir $CONFIG_DIR"

$COHCTL $ARGS add cluster local -u localhost:30000
$COHCTL $ARGS get clusters
$COHCTL $ARGS set context local

count=0
while [ $count -lt $ITERS ]
do
    $COHCTL $ARGS describe cluster local -v -o wide
    let count=count+1
    echo "Count: $count  `date`"
done
