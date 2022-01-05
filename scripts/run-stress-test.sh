#!/bin/bash

#
# Copyright (c) 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run a stress test on the CLI.
# It is assumed that a cluster with http management on localhost:30000 is running

set -e

if [ $# -ne 2 ] ; then
  echo "Usage: $0 [cohctl-binary] iterations"
  exit 1
fi

COHCTL=$1
ITERS=$2

if [ ! -x $COHCTL ] ; then
  echo "Cannot execute $COHCTL"
  exit 2
fi

CONFIG_DIR=/tmp/$$.config
trap "rm -rf $CONFIG_DIR" 0 1 2 3

mkdir -p $CONFIG_DIR

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
