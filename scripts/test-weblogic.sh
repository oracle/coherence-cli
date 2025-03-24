#!/bin/bash

#
# Copyright (c) 2022, 2024 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Run a suite of tests against WebLogic Server
# A WebLogic Server instance from the multi-server example should be
# running on the specified URL with weblogic/welcome1
#
# Note: the environment variable PRE_1412 is set to true if we are running
#       WLS 14.1.1.0 or below

set -e

if [ $# -ne 1 ] ; then
   echo "Usage: $0 WebLogic-URL"
   exit
fi

URL=$1
CONFIG_DIR=/tmp/$$.weblogic
DIR=`pwd`
OUTPUT=/tmp/$$.output

mkdir -p ${CONFIG_DIR}
trap "rm -rf ${CONFIG_DIR} $OUTPUT" EXIT SIGINT

echo
echo "Using URL:  ${URL}"
echo "Config Dir: ${CONFIG_DIR}"
echo

# Default command
COHCTL="$DIR/bin/cohctl -i --config-dir ${CONFIG_DIR} -U weblogic"

function runCommand() {
    echo "========================================================="
    echo "Running command: cohctl $*"
    echo "welcome1" | $COHCTL $* > $OUTPUT
    cat $OUTPUT
}

# If this has not been set then assume its true
if [ -z "$PRE_1412" ]; then
  PRE_1412=true
fi


# Sleep to ensure WLS is ready to accept requests
sleep 30

runCommand version

runCommand set debug on

# Add cluster
runCommand get clusters
runCommand add cluster -u $URL/management/coherence/latest/clusters wls
runCommand get clusters
runCommand set context wls

# Describe the cluster
runCommand describe cluster wls
runCommand describe cluster wls -v -o wide
runCommand describe cluster wls -o json

# Machines
runCommand get machines
runCommand get machines -o jsonpath="$.items[0].machineName"

MACHINE=`cat $OUTPUT | sed -e 's/\[\"//' -e 's/\"]//'`
if [ "${MACHINE}" != "null"] ; then
  runCommand describe machine ${MACHINE}
  runCommand describe machine ${MACHINE} -o json
fi

# Members
runCommand get members
runCommand get members -o wide
runCommand describe member 1
runCommand describe member 2
runCommand describe member 3
runCommand describe member 4
runCommand describe member 4 -o json

# Services
runCommand get services
runCommand get services -o wide
runCommand get services -o json
runCommand get services -t DistributedCache
runCommand get services -w -a NODE-SAFE

if [ "$PRE_1412" == "true" ] ; then
  runCommand describe service '"ExampleGAR:PartitionedPofCache"'
  runCommand describe service '"ExampleGAR:PartitionedPofCache"' -o wide
else
  runCommand describe service '"demo:DistributedCache"'
  runCommand describe service '"demo:DistributedCache"' -o wide
fi

# Caches
runCommand get caches
runCommand get caches -o wide
runCommand get caches -o json
if [ "$PRE_1412" == "true" ]; then
  runCommand describe cache contacts -s '"ExampleGAR:PartitionedPofCache"'
  runCommand describe cache contacts -s '"ExampleGAR:PartitionedPofCache"' -o wide
  runCommand describe cache contacts -s '"ExampleGAR:PartitionedPofCache"' -o json
else
  runCommand describe cache Price -s '"demo:DistributedCache"'
  runCommand describe cache Price -s '"demo:DistributedCache"' -o wide
  runCommand describe cache Price -s '"demo:DistributedCache"' -o json
fi

# Persistence
runCommand get persistence
runCommand get snapshots

# Reporters
runCommand get reporters
runCommand start reporter 1 -y

sleep 10
runCommand describe reporter 1
grep Started $OUTPUT

runCommand stop reporter 1 -y || true
sleep 10
runCommand describe reporter 1
grep Stopped $OUTPUT

# Diagnostics
runCommand log cluster-state -y
runCommand dump cluster-heap -y
runCommand retrieve thread-dumps -O /tmp -y all
rm /tmp/thread-dump-node-*

# Miscellaneous
runCommand get debug
runCommand set debug on
grep on $OUTPUT

runCommand set debug off
runCommand get debug
grep off $OUTPUT

runCommand get timeout
runCommand set timeout 120
runCommand get timeout
grep 120 $OUTPUT

runCommand set timeout 60
runCommand get timeout
grep 60 $OUTPUT

runCommand get management
runCommand get ignore-certs
runCommand set ignore-certs true
runCommand get ignore-certs
grep true $OUTPUT
runCommand set ignore-certs false
runCommand get ignore-certs
grep false $OUTPUT

runCommand remove cluster wls -y


