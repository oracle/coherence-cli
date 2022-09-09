#!/bin/bash

#
# Copyright (c) 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Test various command related to creating/ starting/ stopping and scaling clusters
# environment variables COM and COHERENCE_VERSION accepted

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
echo "Commercial: ${COM}"
echo

# Default command
COHCTL="$DIR/bin/cohctl --config-dir ${CONFIG_DIR}"

function pause() {
   echo "sleeping..."
   sleep 5
}

function wait_for_ready() {
  counter=10
  PORT=$1
  if [ -z "$PORT" ] ; then
    PORT=30000
  fi
  pause
  echo "waiting for management to be ready on $PORT ..."
  while [ $counter -gt 0 ]
  do
    curl http://127.0.0.1:${PORT}/management/coherence/cluster > /dev/null 2>&1
    ret=$?
    if [ $ret -eq 0 ] ; then
        echo "Management ready"
        pause
        return 0
    fi
    pause
    let counter=counter-1
  done
  echo "Management failed to be ready"
  save_logs
  exit 1
}

function message() {
    echo "========================================================="
    echo "$*"
}

function save_logs() {
    mkdir -p build/_output/test-logs
    cp ${CONFIG_DIR}/logs/local/*.log build/_output/test-logs || true
}

function runCommand() {
    echo "========================================================="
    echo "Running command: cohctl $*"
    $COHCTL $* > $OUTPUT 2>&1
    ret=$?
    cat $OUTPUT
    if [ $ret -ne 0 ] ; then
      echo "Command failed"
      # copy the log files
      save_logs
      exit 1
    fi
}

runCommand version

runCommand set debug on

# Create a cluster
message "Create Cluster"
runCommand create cluster local -y -v $VERSION $COM
runCommand set context local

wait_for_ready

runCommand get clusters
runCommand get members

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 3 members
grep "[3,3,3]" $OUTPUT

# Scale the cluster to 6 members
message "Scale Cluster to 6 members"
runCommand scale cluster local -y -r 6
pause && pause && pause

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 6 members
grep "[6,6,6]" $OUTPUT

# Shutdown
runCommand stop cluster local -y

message "Startup cluster with 5 members"
runCommand start cluster local -y -r 5
wait_for_ready

runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"
grep "[5,5,5,]" $OUTPUT

runCommand stop cluster local -y
runCommand remove cluster local -y
pause

message "Start cluster using different HTTP port"
runCommand create cluster local -H 30001 -l 9 -y $COM -v $VERSION
wait_for_ready 30001

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
wait_for_ready 30001

runCommand get members
grep "1,024 MB" $OUTPUT > /dev/null 2>&1
echo "Pausing for a bit"

runCommand stop cluster local -y

pause
runCommand remove cluster local -y

message "Run CohQL"
runCommand create cluster local -y -M 512m -S $COM -v $VERSION
wait_for_ready

echo "insert into test key(1) value(1);" > /tmp/file.cohql
runCommand start cohql -f /tmp/file.cohql
runCommand get caches
runCommand describe cache test -s PartitionedCache

runCommand stop cluster local -y
pause
runCommand remove cluster local -y

# Don't run concurrent test for commercial
if [ -z "$COM" ] ; then
  message "Create cluster with executor"
  runCommand create cluster local -y -M 512m -a coherence-concurrent -v $VERSION
  wait_for_ready

  runCommand get executors
  grep default $OUTPUT

  runCommand stop cluster local -y
  pause
  runCommand remove cluster local -y
fi







