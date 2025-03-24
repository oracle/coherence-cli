#!/bin/bash

#
# Copyright (c) 2024, 2025 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Test monitor cluster

pwd

if [ $# -ne 1 ] ; then
   echo "Usage: $0 Coherence-Version"
   exit
fi

VERSION=$1

# Use WORKSPACE directory if running under Jenkins
TEMP_DIR=/tmp
[ ! -z "$WORKSPACE" ] && TEMP_DIR=$WORKSPACE

export CONFIG_DIR=${TEMP_DIR}/$$.create
export DIR=`pwd`
OUTPUT=${TEMP_DIR}/$$.output

mkdir -p ${CONFIG_DIR}
trap "rm -rf $CONFIG_DIR $OUTPUT" EXIT SIGINT

echo
echo "Config Dir: ${CONFIG_DIR}"
echo "Version:    ${VERSION}"
echo "Commercial: ${COM}"
echo

# Build the Java project so we get any deps downloaded

COHERENCE_GROUP_ID=com.oracle.coherence.ce
if [ ! -z "$COM" ] ; then
  COHERENCE_GROUP_ID=com.oracle.coherence
fi

mvn -f java/coherence-cli-test dependency:build-classpath -Dcoherence.group.id=${COHERENCE_GROUP_ID} -Dcoherence.version=${VERSION}

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
runCommand create cluster local -y -v $VERSION $COM -S com.tangosol.net.Coherence

wait_for_ready

expect -f $DIR/scripts/monitor.expect
ret=$?

# Shutdown
runCommand stop cluster local -y
runCommand remove cluster local -y

exit $ret
