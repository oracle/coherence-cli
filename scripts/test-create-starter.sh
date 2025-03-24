#!/bin/bash

#
# Copyright (c) 2025 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Test various command related to creating starter projects
# environment variables COM and COHERENCE_VERSION accepted

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

echo "Testing Helidon Starter"
runCommand create starter helidon-starter -y -f helidon
cd helidon-starter
mvn clean install
java -jar target/helidon.jar > ${LOGS_DIR}/helidon.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..

echo "Testing Spring Boot Starter"
runCommand create starter springboot-starter -y -f springboot
cd springboot-starter
mvn clean install
java -jar target/springboot-1.0-SNAPSHOT.jar > ${LOGS_DIR}/springboot.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..

echo "Testing Micronaut Starter"
runCommand create starter micronaut-starter -y -f micronaut
cd micronaut-starter
mvn clean install
java -jar target/micronaut-1.0-SNAPSHOT-shaded.jar > ${LOGS_DIR}/micronaut.log 2>&1 &
PID=$!
echo "Sleeping for 30..."
sleep 30
run_test
kill -9 $PID
cd ..
