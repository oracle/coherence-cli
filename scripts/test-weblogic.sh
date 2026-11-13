#!/bin/bash

#
# Copyright (c) 2022, 2026 Oracle and/or its affiliates.
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

if [ $# -lt 1 ] || [ $# -gt 2 ] || { [ $# -eq 2 ] && [ "$2" != "--tls-settings-only" ]; }; then
   echo "Usage: $0 http[s]://WebLogic-URL [--tls-settings-only]" >&2
   exit 1
fi

URL=${1%/}
case "$URL" in
    https://*) TLS_SCHEME=https ;;
    http://*) TLS_SCHEME=http ;;
    *) echo 'WebLogic URL must use http:// or https://' >&2; exit 1 ;;
esac

# Use a private directory, including logs and diagnostic output. Never remove
# another run's files or copy over a shared /tmp/cohctl.log.
CONFIG_DIR=$(mktemp -d "${WORKSPACE:-${TMPDIR:-/tmp}}/cohctl-weblogic.XXXXXX")
DIR=$(pwd)
OUTPUT="$CONFIG_DIR/output"
trap 'rm -rf "$CONFIG_DIR"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

echo
echo "Using URL:  ${URL}"
echo "Config Dir: ${CONFIG_DIR}"
echo

COHCTL=("$DIR/bin/cohctl" -i --config-dir "$CONFIG_DIR" -U weblogic)

function runCommand() {
    local result=0
    echo "========================================================="
    echo "Running command: cohctl $*"
    echo "welcome1" | "${COHCTL[@]}" "$@" > "$OUTPUT" || result=$?
    cat "$OUTPUT"
    return "$result"
}

function checkTLSSettings() {
    runCommand get ignore-certs -c wls
    if [ "$TLS_SCHEME" = https ]; then
        grep -Fx 'Current setting: false' "$OUTPUT"
        runCommand set ignore-certs true -c wls
        runCommand get ignore-certs -c wls
        grep -Fx 'Current setting: true' "$OUTPUT"
        runCommand get members -c wls
        runCommand set ignore-certs false -c wls
        runCommand get ignore-certs -c wls
        grep -Fx 'Current setting: false' "$OUTPUT"
        runCommand get members -c wls
    else
        grep -Fx 'Current setting: n/a' "$OUTPUT"
        for setting in true false; do
            if runCommand set ignore-certs "$setting" -c wls 2>"$CONFIG_DIR/rejected.err"; then
                echo "HTTP unexpectedly accepted ignore-certs $setting" >&2
                exit 1
            fi
            grep -F 'must have a valid HTTPS URL to set ignore-certs' "$CONFIG_DIR/rejected.err"
        done
        runCommand get ignore-certs -c wls
        grep -Fx 'Current setting: n/a' "$OUTPUT"
    fi
}

# If this has not been set then assume its true
if [ -z "$PRE_1412" ]; then
  PRE_1412=true
fi


# Sleep to ensure WLS is ready to accept requests
if [ "${2:-}" != --tls-settings-only ]; then sleep 30; fi

runCommand version

runCommand set debug on

# Add cluster
runCommand get clusters
runCommand add cluster -u "$URL/management/coherence/latest/clusters" wls
runCommand get clusters
runCommand set context wls
checkTLSSettings
if [ "${2:-}" = --tls-settings-only ]; then
    runCommand remove cluster wls -y
    exit 0
fi

# Describe the cluster
runCommand describe cluster wls
runCommand describe cluster wls -v -o wide
runCommand describe cluster wls -o json

# Machines
runCommand get machines
runCommand get machines -o jsonpath="$.items[0].machineName"

MACHINE=$(sed -e 's/\[\"//' -e 's/\"]//' "$OUTPUT")
if [ "${MACHINE}" != "null" ] ; then
  runCommand describe machine "${MACHINE}"
  runCommand describe machine "${MACHINE}" -o json
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
grep Started "$OUTPUT"

runCommand stop reporter 1 -y || true
sleep 10
runCommand describe reporter 1
grep Stopped "$OUTPUT"

# Diagnostics
runCommand log cluster-state -y
runCommand dump cluster-heap -y
runCommand retrieve thread-dumps -O "$CONFIG_DIR" -y all

# Miscellaneous
runCommand get debug
runCommand set debug on
grep on "$OUTPUT"

runCommand set debug off
runCommand get debug
grep off "$OUTPUT"

runCommand get timeout
runCommand set timeout 120
runCommand get timeout
grep 120 "$OUTPUT"

runCommand set timeout 60
runCommand get timeout
grep 60 "$OUTPUT"

runCommand get management

runCommand remove cluster wls -y


