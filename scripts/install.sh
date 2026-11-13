#!/bin/bash
#
# Copyright (c) 2023, 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# Description: Install script for Coherence-CLI for Linux/Mac.
# Author:      tam 2023.02.20
# Usage:
#  Run the following on Linux or Mac:
#
#   curl -fsSL --proto '=https' --proto-redir '=https' https://raw.githubusercontent.com/oracle/coherence-cli/main/scripts/install.sh | bash
#
# for Linux, export COPY=false, to just download the verified cohctl executable and not copy to the /usr/local/bin directory
#
OS=$(uname)
ARCH=$(uname -m)

function set_arch() {
    if [ "$ARCH" = "x86_64" ] ; then
        ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ] ; then
        ARCH="arm64"
    else
        echo "Unsupported architecture: $ARCH"
        exit 1
    fi
}

function installed() {
    echo
    echo "To uninstall the Coherence CLI execute the following:"
    echo "  sudo rm /usr/local/bin/cohctl"
    echo
}

function fail() {
    echo "Error: $*" >&2
    exit 1
}

function install_linux() (
    set -euo pipefail

    local copy=true
    local version
    local temp=""
    local keep_temp=false
    local asset
    local manifest
    local release_url
    local checksum_entry

    if [ "${COPY:-}" = "false" ]; then
        copy=false
    fi

    set_arch
    command -v sha256sum >/dev/null 2>&1 ||
        fail "sha256sum is required to verify the Linux download"

    version=$(curl -fsSL --proto '=https' --proto-redir '=https' https://oracle.github.io/coherence-cli/stable.txt) ||
        fail "unable to retrieve the stable Coherence CLI version"
    while [[ "${version}" == *$'\r' ]]; do
        version=${version%$'\r'}
    done
    if [[ ! "${version}" =~ ^[0-9A-Za-z][0-9A-Za-z._-]*$ ]]; then
        fail "stable.txt did not contain exactly one valid Coherence CLI version"
    fi

    echo "Installing Coherence CLI ${version} for ${OS}/${ARCH}"

    # shellcheck disable=SC2329 # Invoked indirectly by the EXIT trap.
    function cleanup_linux() {
        if [ "${keep_temp}" != "true" ] && [ -n "${temp}" ]; then
            rm -rf -- "${temp}"
        fi
    }
    trap cleanup_linux EXIT

    temp=$(mktemp -d) || fail "unable to create a temporary directory"
    asset="cohctl-${version}-linux-${ARCH}"
    manifest="cohctl-${version}-checksums.txt"
    release_url="https://github.com/oracle/coherence-cli/releases/download/${version}"
    checksum_entry="${asset}.sha256"

    curl -fsSL --proto '=https' --proto-redir '=https' -o "${temp}/${manifest}" "${release_url}/${manifest}" ||
        fail "unable to download checksum manifest ${manifest}"
    curl -fsSL --proto '=https' --proto-redir '=https' -o "${temp}/${asset}" "${release_url}/${asset}" ||
        fail "unable to download ${asset}"

    if ! awk -v target="${asset}" '
        $2 == target {
            matches++
            if (NF == 2 && length($1) == 64 && $1 !~ /[^0-9a-f]/) {
                valid++
                entry = $1 "  " $2
            }
        }
        END {
            if (matches != 1 || valid != 1) {
                exit 1
            }
            print entry
        }
    ' "${temp}/${manifest}" > "${temp}/${checksum_entry}"; then
        fail "checksum manifest must contain exactly one valid entry for ${asset}"
    fi

    if ! (cd "${temp}" && sha256sum -c "${checksum_entry}"); then
        fail "checksum verification failed for ${asset}"
    fi

    mv "${temp}/${asset}" "${temp}/cohctl"
    chmod u+x "${temp}/cohctl"

    if [ "${copy}" = "false" ]; then
        keep_temp=true
        echo "cohctl downloaded to ${temp}/cohctl, please move to your required location"
    else
        echo "Using 'sudo' to mv cohctl binary to /usr/local/bin"
        sudo mv "${temp}/cohctl" /usr/local/bin/
        installed
    fi
)

if [ "$OS" == "Darwin" ]; then
    set_arch
    TEMP=$(mktemp -d)
    # Default the version to the last know valid installer, until MacOS installer is re-instated
    VERSION=1.10.1
    PKG="Oracle-Coherence-CLI-${VERSION}-darwin-${ARCH}.pkg"
    DEST=${TEMP}/${PKG}
    echo "Installing Coherence CLI ${VERSION} for ${OS}/${ARCH}"
    echo "Downloading and opening ${DEST}"
    URL=https://github.com/oracle/coherence-cli/releases/download/${VERSION}/${PKG}
    curl -fsSL --proto '=https' --proto-redir '=https' -o "${DEST}" "${URL}" && open "${DEST}" && installed
elif [ "$OS" == "Linux" ]; then
    install_linux
else
    echo "For all other platforms, please see: https://github.com/oracle/coherence-cli/releases"
    exit 1
fi
