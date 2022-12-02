#!/bin/bash

#
# Copyright (c) 2021, 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Generate output to be included into the commands reference documentation.

set -e

PWD=`pwd`
DOCS_DIR=$1
mkdir -p $DOCS_DIR
BIN_DIR=$2
COHCTL=${BIN_DIR}/cohctl
TMP_FILE=/tmp/generate.$$

trap "rm -f $TMP_FILE" 0 1 2 3

# Creates a adoc section for the given command help
function create_doc {
  FILE_PREFIX=$1
  shift
  COMMAND="$*"

  echo "$COMMAND"
  eval $COMMAND > $TMP_FILE

  DESC=`cat $TMP_FILE  | sed -n '/^Usage:/!p;//q'`
  USAGE=`cat $TMP_FILE | sed -ne '/^Usage:/,$ p' | sed -n '/^Flags:/!p;//q'`
  FLAGS=`cat $TMP_FILE | sed -ne '/^Flags:/,$ p' | sed -n '/^Global Flags:/!p;//q'`

  (echo "// # tag::text[]" && \
   echo "$DESC" && echo && \
   echo "$USAGE" | sed 's/Usage:/*Usage*~----~/' | tr '~' '\n' && echo "----" && echo && \
   echo "$FLAGS" | sed 's/Flags:/*Flags*~----~/' | tr '~' '\n' && echo "----" && echo && \
   echo "// # end::text[]" ) > ${FILE_PREFIX}.adoc
}

# Cluster
create_doc $DOCS_DIR/add_cluster "${COHCTL} add cluster --help"
create_doc $DOCS_DIR/remove_cluster "${COHCTL} remove cluster --help"
create_doc $DOCS_DIR/get_clusters "${COHCTL} get clusters --help"
create_doc $DOCS_DIR/describe_cluster "${COHCTL} describe cluster --help"
create_doc $DOCS_DIR/discover_clusters "${COHCTL} discover clusters --help"

# Create Cluster
create_doc $DOCS_DIR/create_cluster "${COHCTL} create cluster --help"
create_doc $DOCS_DIR/scale_cluster "${COHCTL} scale cluster --help"
create_doc $DOCS_DIR/start_cluster "${COHCTL} start cluster --help"
create_doc $DOCS_DIR/stop_cluster "${COHCTL} stop cluster --help"
create_doc $DOCS_DIR/start_console "${COHCTL} start console --help"
create_doc $DOCS_DIR/start_cohql "${COHCTL} start cohql --help"
create_doc $DOCS_DIR/start_class "${COHCTL} start class --help"
create_doc $DOCS_DIR/get_profiles "${COHCTL} get profiles --help"
create_doc $DOCS_DIR/set_profile "${COHCTL} set profile --help"
create_doc $DOCS_DIR/remove_profile "${COHCTL} remove profile --help"

# Context
create_doc $DOCS_DIR/set_context "${COHCTL} set context --help"
create_doc $DOCS_DIR/get_context "${COHCTL} get context --help"
create_doc $DOCS_DIR/clear_context "${COHCTL} clear context --help"

# Members
create_doc $DOCS_DIR/get_members "${COHCTL} get members --help"
create_doc $DOCS_DIR/describe_member "${COHCTL} describe member --help"
create_doc $DOCS_DIR/set_member "${COHCTL} set member --help"
create_doc $DOCS_DIR/shutdown_member "${COHCTL} shutdown member --help"

# Machines
create_doc $DOCS_DIR/get_machines "${COHCTL} get machines --help"
create_doc $DOCS_DIR/describe_machine "${COHCTL} describe machine --help"

# Federation
create_doc $DOCS_DIR/get_federation "${COHCTL} get federation --help"
create_doc $DOCS_DIR/set_federation "${COHCTL} set federation --help"
create_doc $DOCS_DIR/describe_federation "${COHCTL} describe federation --help"
create_doc $DOCS_DIR/start_federation "${COHCTL} start federation --help"
create_doc $DOCS_DIR/stop_federation "${COHCTL} stop federation --help"
create_doc $DOCS_DIR/pause_federation "${COHCTL} pause federation --help"
create_doc $DOCS_DIR/replicate_all "${COHCTL} replicate all --help"

# Services
create_doc $DOCS_DIR/get_services "${COHCTL} get services --help"
create_doc $DOCS_DIR/get_service_members "${COHCTL} get service-members --help"
create_doc $DOCS_DIR/get_service_storage "${COHCTL} get service-storage --help"
create_doc $DOCS_DIR/describe_service "${COHCTL} describe service --help"
create_doc $DOCS_DIR/set_service "${COHCTL} set service --help"
create_doc $DOCS_DIR/start_service "${COHCTL} start service --help"
create_doc $DOCS_DIR/stop_service "${COHCTL} stop service --help"
create_doc $DOCS_DIR/shutdown_service "${COHCTL} shutdown service --help"
create_doc $DOCS_DIR/suspend_service "${COHCTL} suspend service --help"
create_doc $DOCS_DIR/resume_service "${COHCTL} resume service --help"

# Management
create_doc $DOCS_DIR/get_management "${COHCTL} get management --help"
create_doc $DOCS_DIR/set_management "${COHCTL} set management --help"

# Caches
create_doc $DOCS_DIR/get_caches "${COHCTL} get caches --help"
create_doc $DOCS_DIR/get_cache_stores "${COHCTL} get cache-stores --help"
create_doc $DOCS_DIR/describe_cache "${COHCTL} describe cache --help"
create_doc $DOCS_DIR/set_cache "${COHCTL} set cache --help"

# Topics
create_doc $DOCS_DIR/get_topics "${COHCTL} get topics --help"
create_doc $DOCS_DIR/describe_topic "${COHCTL} describe topic --help"
create_doc $DOCS_DIR/get_topic_members "${COHCTL} get topic-members --help"
create_doc $DOCS_DIR/get_member_channels "${COHCTL} get member-channels --help"
create_doc $DOCS_DIR/get_subscribers "${COHCTL} get subscribers --help"
create_doc $DOCS_DIR/get_subscriber_channels "${COHCTL} get subscriber-channels --help"
create_doc $DOCS_DIR/get_subscriber_groups "${COHCTL} get subscriber-groups --help"
create_doc $DOCS_DIR/get_sub_grp_channels "${COHCTL} get sub-grp-channels --help"
create_doc $DOCS_DIR/get_sub_grp_channels "${COHCTL} get sub-grp-channels --help"
create_doc $DOCS_DIR/disconnect_subscriber "${COHCTL} disconnect subscriber --help"
create_doc $DOCS_DIR/connect_subscriber "${COHCTL} connect subscriber --help"
create_doc $DOCS_DIR/retrieve_heads "${COHCTL} retrieve heads --help"
create_doc $DOCS_DIR/retrieve_remaining "${COHCTL} retrieve remaining --help"
create_doc $DOCS_DIR/notify_populated "${COHCTL} notify populated --help"

# nslookup
create_doc $DOCS_DIR/nslookup "${COHCTL} nslookup --help"

# Proxies
create_doc $DOCS_DIR/get_proxies "${COHCTL} get proxies --help"
create_doc $DOCS_DIR/get_proxy_connections "${COHCTL} get proxy-connections --help"
create_doc $DOCS_DIR/describe_proxy "${COHCTL} describe proxy --help"

# Http Servers
create_doc $DOCS_DIR/get_http_servers "${COHCTL} get http-servers --help"
create_doc $DOCS_DIR/describe_http_server "${COHCTL} describe http-server --help"

# Reporters
create_doc $DOCS_DIR/get_reporters "${COHCTL} get reporters --help"
create_doc $DOCS_DIR/describe_reporter "${COHCTL} describe reporter --help"
create_doc $DOCS_DIR/start_reporter "${COHCTL} start reporter --help"
create_doc $DOCS_DIR/stop_reporter "${COHCTL} stop reporter --help"
create_doc $DOCS_DIR/set_reporter "${COHCTL} set reporter --help"

# JFRs
create_doc $DOCS_DIR/get_jfrs "${COHCTL} get jfrs --help"
create_doc $DOCS_DIR/start_jfr "${COHCTL} start jfr --help"
create_doc $DOCS_DIR/stop_jfr "${COHCTL} stop jfr --help"
create_doc $DOCS_DIR/dump_jfr "${COHCTL} dump jfr --help"
create_doc $DOCS_DIR/describe_jfr "${COHCTL} describe jfr --help"

# Dump Cluster Heap
create_doc $DOCS_DIR/dump_cluster_heap "${COHCTL} dump cluster-heap --help"

# Log Cluster State
create_doc $DOCS_DIR/log_cluster_state "${COHCTL} log cluster-state --help"

# Tracing
create_doc $DOCS_DIR/configure_tracing "${COHCTL} configure tracing --help"
create_doc $DOCS_DIR/get_tracing "${COHCTL} get tracing --help"

# Timeout
create_doc $DOCS_DIR/set_timeout "${COHCTL} set timeout --help"
create_doc $DOCS_DIR/get_timeout "${COHCTL} get timeout --help"

# Elastic Data
create_doc $DOCS_DIR/get_elastic_data "${COHCTL} get elastic-data --help"
create_doc $DOCS_DIR/describe_elastic_data "${COHCTL} describe elastic-data --help"
create_doc $DOCS_DIR/compact_elastic_data "${COHCTL} compact elastic-data --help"

# Executors
create_doc $DOCS_DIR/get_executors "${COHCTL} get executors --help"
create_doc $DOCS_DIR/set_executor "${COHCTL} set executor --help"
create_doc $DOCS_DIR/describe_executor "${COHCTL} describe executor --help"

# Http session
create_doc $DOCS_DIR/get_http_sessions "${COHCTL} get http-sessions --help"
create_doc $DOCS_DIR/describe_http_session "${COHCTL} describe http-session --help"

# Health
create_doc $DOCS_DIR/get_health "${COHCTL} get health --help"

# Environment
create_doc $DOCS_DIR/get_environment "${COHCTL} get environment --help"

# Debug
create_doc $DOCS_DIR/set_debug "${COHCTL} set debug --help"
create_doc $DOCS_DIR/get_debug "${COHCTL} get debug --help"

# Use Gradle
create_doc $DOCS_DIR/set_use_gradle "${COHCTL} set use-gradle --help"
create_doc $DOCS_DIR/get_use_gradle "${COHCTL} get use-gradle --help"

# Bytes Display
create_doc $DOCS_DIR/set_bytes_format "${COHCTL} set bytes-format --help"
create_doc $DOCS_DIR/get_bytes_format "${COHCTL} get bytes-format --help"
create_doc $DOCS_DIR/clear_bytes_format "${COHCTL} clear bytes-format --help"

# Default Heap
create_doc $DOCS_DIR/set_default_heap "${COHCTL} set default-heap --help"
create_doc $DOCS_DIR/get_default_heap "${COHCTL} get default-heap --help"
create_doc $DOCS_DIR/clear_default_heap "${COHCTL} clear default-heap --help"

# Ignore Certs
create_doc $DOCS_DIR/set_ignore_certs "${COHCTL} set ignore-certs --help"
create_doc $DOCS_DIR/get_ignore_certs "${COHCTL} get ignore-certs --help"

# Version
create_doc $DOCS_DIR/version "${COHCTL} version --help"

# Logs
create_doc $DOCS_DIR/get_logs "${COHCTL} get logs --help"

# Thread Dump
create_doc $DOCS_DIR/retrieve_thread_dumps "${COHCTL} retrieve thread-dumps --help"

# Reset Stats
create_doc $DOCS_DIR/reset_cache_stats "${COHCTL} reset cache-stats --help"
create_doc $DOCS_DIR/reset_executor_stats "${COHCTL} reset executor-stats --help"
create_doc $DOCS_DIR/reset_federation_stats "${COHCTL} reset federation-stats --help"
create_doc $DOCS_DIR/reset_flashjournal_stats "${COHCTL} reset flashjournal-stats --help"
create_doc $DOCS_DIR/reset_ramjournal_stats "${COHCTL} reset ramjournal-stats --help"
create_doc $DOCS_DIR/reset_member_stats "${COHCTL} reset member-stats --help"
create_doc $DOCS_DIR/reset_reporter_stats "${COHCTL} reset reporter-stats --help"
create_doc $DOCS_DIR/reset_service_stats "${COHCTL} reset service-stats --help"

# Persistence
create_doc $DOCS_DIR/get_persistence "${COHCTL} get persistence --help"
create_doc $DOCS_DIR/get_snapshots "${COHCTL} get snapshots --help"
create_doc $DOCS_DIR/create_snapshot "${COHCTL} create snapshot --help"
create_doc $DOCS_DIR/remove_snapshot "${COHCTL} remove snapshot --help"
create_doc $DOCS_DIR/recover_snapshot "${COHCTL} recover snapshot --help"
create_doc $DOCS_DIR/archive_snapshot "${COHCTL} archive snapshot --help"
create_doc $DOCS_DIR/retrieve_snapshot "${COHCTL} retrieve snapshot --help"

# General Help
( echo "// # tag::text[]" && \
 ${COHCTL} --help && \
 echo "// # end::text[]" ) > $DOCS_DIR/cohctl_help.adoc

# Global Flags
( echo "// # tag::text[]" && \
 ${COHCTL} --help | sed -ne '/^Flags:/,$ p' | sed '/more information about/d' && \
 echo "// # end::text[]" ) > $DOCS_DIR/global_flags.adoc

