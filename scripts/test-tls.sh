#!/usr/bin/env bash
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.

# Read-only management checks against a secure standalone fixture.
set -euo pipefail

URL=${1:-https://127.0.0.1:30000/management/coherence/cluster}
COHCTL_BIN=${COHCTL_BIN:-"$(pwd)/bin/cohctl"}
: "${COHERENCE_TLS_CERTS_PATH:?set the fixture CA path}"
: "${COHERENCE_TLS_CLIENT_CERT:?set the fixture client certificate}"
: "${COHERENCE_TLS_CLIENT_KEY:?set the fixture client key}"
UNTRUSTED_CA="$(dirname "$COHERENCE_TLS_CERTS_PATH")/ravagers-ca.crt"
test -r "$UNTRUSTED_CA"

WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/cohctl-tls.XXXXXX")
trap 'rm -rf "$WORK_DIR"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

cohctl() {
    "$COHCTL_BIN" --config-dir "$WORK_DIR/config" "$@"
}

# Readiness must succeed with the CA, matching server identity and client key.
# A failed verified request is never accepted as successful TLS coverage.
for attempt in {1..30}; do
    if cohctl add cluster tls-verified -u "$URL" 2>"$WORK_DIR/verified.err"; then
        break
    fi
    if [ "$attempt" -eq 30 ]; then
        cat "$WORK_DIR/verified.err" >&2
        exit 1
    fi
    sleep 2
done
cat "$WORK_DIR/verified.err" >&2
if grep -q 'WARNING: TLS certificate verification' "$WORK_DIR/verified.err"; then
    echo 'Verified add unexpectedly disabled certificate verification' >&2
    exit 1
fi
cohctl get ignore-certs -c tls-verified | grep -Fx 'Current setting: false'
cohctl get members -c tls-verified

# Wrong server trust must fail even when a client identity is supplied.
if COHERENCE_TLS_CERTS_PATH="$UNTRUSTED_CA" cohctl get members -c tls-verified >"$WORK_DIR/untrusted.out" 2>&1; then
    echo 'Verified connection unexpectedly accepted an untrusted server' >&2
    exit 1
fi
grep -q 'certificate signed by unknown authority' "$WORK_DIR/untrusted.out"

# tls-files with its CA trust-manager requires a trusted client certificate.
if COHERENCE_TLS_CLIENT_CERT='' COHERENCE_TLS_CLIENT_KEY='' cohctl get members -c tls-verified >"$WORK_DIR/no-client.out" 2>&1; then
    echo 'Mutual TLS unexpectedly accepted a missing client identity' >&2
    exit 1
fi
# JVMs may close with EOF instead of a TLS alert. Require rejection here;
# successful authenticated requests before and after rule out an unavailable server.

# Use the explicit add choice against the same server with deliberately wrong
# server trust. This proves bypass rather than merely using the valid CA again.
COHERENCE_TLS_CERTS_PATH="$UNTRUSTED_CA" cohctl add cluster tls-bypass -u "$URL" --ignore-certs 2>"$WORK_DIR/bypass.err"
cat "$WORK_DIR/bypass.err" >&2
grep -q 'WARNING: TLS certificate verification is disabled for connection "tls-bypass"' "$WORK_DIR/bypass.err"
cohctl get ignore-certs -c tls-bypass | grep -Fx 'Current setting: true'
COHERENCE_TLS_CERTS_PATH="$UNTRUSTED_CA" cohctl get members -c tls-bypass
cohctl set ignore-certs false -c tls-bypass
cohctl get ignore-certs -c tls-bypass | grep -Fx 'Current setting: false'
if COHERENCE_TLS_CERTS_PATH="$UNTRUSTED_CA" cohctl get members -c tls-bypass >"$WORK_DIR/restored.out" 2>&1; then
    echo 'Removing the scoped exception did not restore verification' >&2
    exit 1
fi
grep -q 'certificate signed by unknown authority' "$WORK_DIR/restored.out"
cohctl get ignore-certs -c tls-verified | grep -Fx 'Current setting: false'
cohctl get members -c tls-verified
cohctl remove cluster tls-bypass -y
cohctl remove cluster tls-verified -y
echo 'Verified CA/mTLS and saved-connection bypass checks passed'
