#!/bin/bash

#
# Copyright (c) 2025 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Test various command related to creating starter projects using Polyglot clients

pwd

# Use WORKSPACE directory if running under Jenkins
TEMP_DIR=/tmp
[ ! -z "$WORKSPACE" ] && TEMP_DIR=$WORKSPACE

CONFIG_DIR=${TEMP_DIR}/$$.create.starter
DIR=`pwd`
OUTPUT=${TEMP_DIR}/$$.output
STARTER_DIR=${TEMP_DIR}/$$.starter
LOGS_DIR=$DIR/build/_output/test-logs

mkdir -p ${CONFIG_DIR} ${STARTER_DIR} ${LOGS_DIR}
trap "rm -rf $CONFIG_DIR $OUTPUT $STARTER_DIR" EXIT SIGINT

echo
echo "Config Dir:  ${CONFIG_DIR}"
echo "Starter Dir: ${STARTER_DIR}"
echo "Logs Dir:    ${LOGS_DIR}"
echo

# Default command
COHCTL="$DIR/bin/cohctl --config-dir ${CONFIG_DIR}"

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

cd ${STARTER_DIR}

function run_test() {
    curl -X POST -H "Content-Type: application/json" -d '{"id": 1, "name": "Tim", "balance": 1000}' http://localhost:8080/api/customers
    curl -s http://localhost:8080/api/customers/1
    curl -s http://localhost:8080/api/customers
    curl -X DELETE http://localhost:8080/api/customers/1
}

echo "Start Docker Image"
docker run -d -p 1408:1408 -p 30000:30000 ghcr.io/oracle/coherence-ce:25.03.1
echo "Sleeping 30..."
sleep 30

echo "Testing Python Starter"
runCommand create starter python-starter -y -l python
cd python-starter
pip install -r requiements.txt
python main.py > ${LOGS_DIR}/python.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..

echo "Testing JavaScript Starter"
runCommand create starter javascript-starter -y -l javascript
cd javascript-starter
npm install
node main.js > ${LOGS_DIR}/javascript.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..

echo "Testing Go Starter"
runCommand create starter go-starter -y -l go
cd go-starter
go mod tidy
go run main.go > ${LOGS_DIR}/go.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..
