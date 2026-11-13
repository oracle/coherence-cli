#!/bin/bash
#
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# Usage: generate-checksums.sh VERSION BIN_DIRECTORY DOCS_ZIP OUTPUT_MANIFEST
set -euo pipefail

if [ "$#" -ne 4 ] || [[ ! "$1" =~ ^[0-9A-Za-z][0-9A-Za-z._-]*$ ]]; then
    echo "Usage: $0 VERSION BIN_DIRECTORY DOCS_ZIP OUTPUT_MANIFEST" >&2
    exit 1
fi
version=$1
bin=$2
docs=$3
output=$4
command -v sha256sum >/dev/null

# Write beside the destination so only a complete manifest is published by mv.
temp=$(mktemp "${output}.tmp.XXXXXX")
trap 'rm -f -- "${temp}"' EXIT

add_checksum() {
    local path=$1 name=$2 digest
    test -f "${path}"
    digest=$(sha256sum "${path}" | awk '{print $1}')
    [[ "${digest}" =~ ^[0-9a-f]{64}$ ]]
    printf '%s  %s\n' "${digest}" "${name}" >> "${temp}"
}

add_checksum "${bin}/linux/amd64/cohctl" "cohctl-${version}-linux-amd64"
add_checksum "${bin}/linux/arm64/cohctl" "cohctl-${version}-linux-arm64"
add_checksum "${bin}/linux/386/cohctl" "cohctl-${version}-linux-386"
add_checksum "${bin}/windows/amd64/cohctl.exe" "cohctl-${version}-windows-amd64.exe"
add_checksum "${bin}/windows/arm64/cohctl.exe" "cohctl-${version}-windows-arm64.exe"
add_checksum "${docs}" docs.zip
mv -- "${temp}" "${output}"
