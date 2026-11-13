#!/bin/bash

#
# Copyright (c) 2022, 2026 Oracle and/or its affiliates.
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
MANAGEMENT_PORT=${COHCTL_TEST_HTTP_PORT:-30000}
ALT_MANAGEMENT_PORT=$((MANAGEMENT_PORT + 1))
CLUSTER_PORT=${COHCTL_TEST_CLUSTER_PORT:-7574}

# Use WORKSPACE directory if running under Jenkins
TEMP_DIR=${TMPDIR:-/tmp}
[ ! -z "$WORKSPACE" ] && TEMP_DIR=$WORKSPACE

CONFIG_DIR=${TEMP_DIR}/$$.create
DIR=`pwd`
OUTPUT=${TEMP_DIR}/$$.output

mkdir -p ${CONFIG_DIR}
function cleanup() {
  if [ -n "$COHCTL" ] && [ -f "${CONFIG_DIR}/cohctl.yaml" ]; then
    $COHCTL stop cluster local -y >/dev/null 2>&1 || true
  fi
  rm -rf "${CONFIG_DIR}" "$OUTPUT"
  [ -z "$SCRIPT" ] || rm -f "$SCRIPT"
  [ -z "$LOGS_DEST" ] || rm -rf "$LOGS_DEST"
}
trap cleanup EXIT
trap 'exit 130' SIGINT

echo
echo "Config Dir: ${CONFIG_DIR}"
echo "Version:    ${VERSION}"
echo "Commercial: ${COM}"
echo

# Build the Java project so we get any deps downloaded

COHERENCE_GROUP_ID=com.oracle.coherence.ce
COMMERCIAL_ARGS=()
if [ ! -z "$COM" ] ; then
  COHERENCE_GROUP_ID=com.oracle.coherence
  COMMERCIAL_ARGS+=(--commercial)
fi

mvn -f java/coherence-cli-test dependency:build-classpath -Dcoherence.group.id=${COHERENCE_GROUP_ID} -Dcoherence.version=${VERSION}
DEPENDENCY_STATUS=$?
if [ "$DEPENDENCY_STATUS" -ne 0 ]; then
  echo "Dependency resolution failed before cluster creation (exit $DEPENDENCY_STATUS)"
  exit "$DEPENDENCY_STATUS"
fi

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
    PORT=$MANAGEMENT_PORT
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
    if [ "$1" = create ] && [ "$2" = cluster ]; then
      case " $* " in *" -H "*|*" --http-port "*) ;; *) set -- "$@" -H "$MANAGEMENT_PORT" ;; esac
      set -- "$@" -p "$CLUSTER_PORT"
    fi
    $COHCTL "$@" > $OUTPUT 2>&1
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
runCommand create cluster local -y -v $VERSION "${COMMERCIAL_ARGS[@]}" -S com.tangosol.net.Coherence --machine machine1

wait_for_ready

runCommand get clusters
runCommand get members

# Check the option to specify host:port for name service
runCommand get members -c localhost:${CLUSTER_PORT}

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 3 members
grep "[3,3,3]" $OUTPUT

# Scale the cluster to 6 members
message "Scale Cluster to 6 members and delay each by 5 seconds"
runCommand scale cluster local -r 6 -D 5s --machine machine2
pause && pause && pause

# Check the members of PartitionedCache
runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"

# must be 6 members
grep "[6,6,6]" $OUTPUT

# Restart the cluster
runCommand restart cluster local -r 6 -y

pause
wait_for_ready

# Shutdown
runCommand stop cluster local -y

message "Startup cluster with 5 members"
runCommand start cluster local -r 5 --site site1 -N
wait_for_ready

runCommand get services -o jsonpath="$.items[?(@.name=='PartitionedCache')].memberCount"
grep "[5,5,5,]" $OUTPUT

# Monitor the heath
runCommand monitor health -n localhost:${CLUSTER_PORT} -N
grep "NODE ID" $OUTPUT
grep "STARTED" $OUTPUT

runCommand stop cluster local -y
runCommand remove cluster local -y
pause && pause && pause

# Don't run for commercial or SNAPSHOT
if [ -z "$COM" -a -z "`echo $VERSION | grep SNAPSHOT`" ] ; then
  echo "Test -F Startup and profile"
  runCommand set profile grpc -v "-Dcoherence.grpc.server.port=1408" -y
  runCommand create cluster local -y -v $VERSION "${COMMERCIAL_ARGS[@]}" -P grpc -a coherence-grpc-proxy -F
  wait_for_ready

  runCommand stop cluster local -y
  runCommand start cluster local -P grpc -F
  wait_for_ready

  runCommand stop cluster local -y
  runCommand remove cluster local -y

  pause && pause && pause
fi

message "Test CohQL commands via extend and gRPC Proxy via NS"
SCRIPT=`mktemp "${TEMP_DIR}/cohctl-query.XXXXXX"`
cat > $SCRIPT <<EOF
insert into test key(1) value('one')
EOF

# Don't run concurrent test for commercial or if we have a snapshot
if [ -z "$COM" -a -z "`echo $VERSION | grep SNAPSHOT`" ] ; then
  runCommand create cluster local -y -v $VERSION "${COMMERCIAL_ARGS[@]}" -a coherence-grpc-proxy,coherence-java-client
  wait_for_ready
  runCommand start cohql -X -f $SCRIPT
  runCommand start cohql -X -f $SCRIPT
  runCommand start cohql -G -f $SCRIPT
  runCommand start cohql -G -f $SCRIPT
  runCommand stop cluster local -y
  runCommand remove cluster local -y
  rm $SCRIPT || true
fi

pause && pause && pause

LOGS_DEST=$(mktemp -d "${TEMP_DIR}/cohctl-logs.XXXXXX")
message "Test different log location - ${LOGS_DEST}"
runCommand create cluster local -y -M 512m -I "${COMMERCIAL_ARGS[@]}" -v $VERSION -L ${LOGS_DEST} --rack rack1
wait_for_ready

# check to see a storage-0.log file exists
if [ ! -f ${LOGS_DEST}/local/storage-0.log ] ; then
  echo "Specifying -L ${LOGS_DEST} did not work for create"
  runCommand stop cluster local -y || true
  runCommand remove cluster local -y || true
  exit 1
fi

runCommand stop cluster local -y
pause && pause

runCommand start cluster local -r 4
wait_for_ready

# Check the log file for member 4 exists
if [ ! -f ${LOGS_DEST}/local/storage-3.log ] ; then
  echo "Specifying -L ${LOGS_DEST} did not work for start cluster"
  runCommand stop cluster local -y || true
  runCommand remove cluster local -y || true
  exit 1
fi

runCommand scale cluster local -r 5
pause && pause

# Check the log file for member 5 exists
if [ ! -f ${LOGS_DEST}/local/storage-4.log ] ; then
  echo "Specifying -L ${LOGS_DEST} did not work for scale cluster"
  runCommand stop cluster local -y || true
  runCommand remove cluster local -y || true
  exit 1
fi

runCommand stop cluster local -y
runCommand remove cluster local -y
pause && pause && pause

message "Start cluster using different HTTP port"
runCommand create cluster local -H ${ALT_MANAGEMENT_PORT} -l 9 "${COMMERCIAL_ARGS[@]}" -v $VERSION -y
wait_for_ready ${ALT_MANAGEMENT_PORT}

message "Add a cluster to point to newly created cluster on port ${ALT_MANAGEMENT_PORT}"
runCommand add cluster local2 -u http://127.0.0.1:${ALT_MANAGEMENT_PORT}/management/coherence/cluster
runCommand get members -c local2
runCommand remove cluster local2 -y

runCommand stop cluster local -y
pause

message "Startup cluster using different memory setting"
runCommand clear default-heap
runCommand start cluster local -r 4 -M 1g
runCommand set bytes-format m
wait_for_ready ${ALT_MANAGEMENT_PORT}

runCommand set context local
runCommand get members
grep "1,024 MB" $OUTPUT > /dev/null 2>&1
echo "Pausing for a bit"

runCommand stop cluster local -y

pause
runCommand remove cluster local -y

message "Run CohQL"
runCommand create cluster local -y -M 512m -I "${COMMERCIAL_ARGS[@]}" -v $VERSION
wait_for_ready

SCRIPT=`mktemp "${TEMP_DIR}/cohctl-query.XXXXXX"`
echo "insert into test key(1) value(1);" > "$SCRIPT"
runCommand start cohql -f "$SCRIPT"
runCommand get caches
runCommand describe cache test -s PartitionedCache

runCommand stop cluster local -y
pause
runCommand remove cluster local -y

# Don't run concurrent test for commercial or if we have a snapshot
if [ -z "$COM" -a -z "`echo $VERSION | grep SNAPSHOT`" ] ; then
  message "Create cluster with executor"
  runCommand create cluster local -y -M 512m -a coherence-concurrent -v $VERSION
  wait_for_ready

  runCommand get executors
  grep default $OUTPUT

  runCommand stop cluster local -y
  pause
  runCommand remove cluster local -y
fi

pause

# Don't run gradle tests on commercial or snapshots until we figure out gradle proxy
if [ -z "$COM" -a -z "`echo $VERSION | grep SNAPSHOT`" ] ; then
  # Setup to create a cluster using gradle

  runCommand set use-gradle true
  runCommand get use-gradle

  message "Create Cluster Using Gradle"
  gradle -v
  runCommand create cluster local -y -v $VERSION "${COMMERCIAL_ARGS[@]}"

  wait_for_ready

  runCommand get clusters
  runCommand get members

  runCommand stop cluster local -y
  pause
  runCommand remove cluster local -y
fi

# Inspect actual management listeners and perform JDK-only JMX/PID checks.
# Missing socket inspection or a real interface is missing coverage, not a pass.
MANAGEMENT_TEST_ARGS=()
if [ -n "$COM" ]; then
  MANAGEMENT_TEST_ARGS+=(--commercial)
fi
if [ -n "$COHCTL_MANAGEMENT_TEST_OUTPUT" ]; then
  MANAGEMENT_TEST_ARGS+=(--output "$COHCTL_MANAGEMENT_TEST_OUTPUT")
fi
python3 "$DIR/scripts/test-local-management.py" "$VERSION" --cli "$DIR/bin/cohctl" "${MANAGEMENT_TEST_ARGS[@]}" || exit 1

# Test the environment variable for setting the home directory
message "Setting COHCTL_HOME to ${CONFIG_DIR}"
export COHCTL_HOME=${CONFIG_DIR}
$DIR/bin/cohctl version





