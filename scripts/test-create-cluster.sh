#!/bin/bash

#
# Copyright (c) 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Test various command related to creating/ starting/ stopping and scaling clusters

pwd

if [ $# -ne 1 ] ; then
   echo "Usage: $0 Coherence-Version"
   exit
fi

VERSION=$1
CONFIG_DIR=/tmp/$$.create
DIR=`pwd`
OUTPUT=/tmp/$$.output

mkdir -p ${CONFIG_DIR}
trap "cp ${CONFIG_DIR}/cohctl.log /tmp && rm -rf ${CONFIG_DIR} $OUTPUT" 0 1 2 3

echo
echo "Config Dir: ${CONFIG_DIR}"
echo "Version:    ${VERSION}"
echo

# Default command
COHCTL="$DIR/bin/cohctl --config-dir ${CONFIG_DIR}"

function pause() {
   echo "sleeping..."
   sleep 5
}

function message() {
    echo "========================================================="
    echo "$*"
}

function runCommand() {
    echo "========================================================="
    echo "Running command: cohctl $*"
    $COHCTL $* > $OUTPUT 2>&1
    ret=$?
    cat $OUTPUT
    if [ $ret -ne 0 ] ; then
      echo "Command failed"
      exit 1
    fi
}

runCommand version

runCommand set debug on

# Create a cluster
message "Create Cluster"
runCommand create cluster local -y -v $VERSION
runCommand set context local

# Wait for startup
pause

runCommand get clusters
runCommand get members

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 3 members
grep "[3,3,3]" $OUTPUT

# Scale the cluster to 6 members
message "Scale Cluster to 6 members"
runCommand scale cluster local -y -r 6
pause

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 6 members
grep "[6,6,6]" $OUTPUT

# Shutdown
runCommand stop cluster local -y

message "Startup cluster with 5 members"
runCommand start cluster local -y -r 5
pause && pause && pause

runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"
grep "[5,5,5,]" $OUTPUT

runCommand stop cluster local -y
runCommand remove cluster local -y
pause

message "Start cluster using different HTTP port"
runCommand create cluster local -H 30001 -l 9 -y
pause

message "Add a cluster to point to newly created cluster on port 30001"
runCommand add cluster local2 -u http://127.0.0.1:30001/management/coherence/cluster
runCommand get members -c local2
runCommand remove cluster local2 -y

runCommand stop cluster local -y
pause

message "Startup cluster using different memory setting"
runCommand clear default-heap
runCommand start cluster local -r 4 -M 1g -y
runCommand set bytes-format m
pause

runCommand get members
grep "1,024 MB" $OUTPUT > /dev/null 2>&1
echo "Pausing for a bit"

runCommand stop cluster local -y

pause
runCommand remove cluster local -y

message "Run CohQL"
runCommand create cluster local -y -M 512m -S
pause

echo "insert into test key(1) value(1);" > /tmp/file.cohql
runCommand start cohql -f /tmp/file.cohql
runCommand get caches
runCommand describe cache test -s PartitionedCache

runCommand stop cluster local -y
pause
runCommand remove cluster local -y

message "Create cluster with executor"
runCommand create cluster local -y -M 512m -a coherence-concurrent
pause

runCommand get executors
grep default $OUTPUT

runCommand stop cluster local -y
pause
runCommand remove cluster local -y







