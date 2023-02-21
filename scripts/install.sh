#!/bin/bash
#
# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# Description: Install script for Coherence-CLI for Linux/Mac.
# Author:      tam 2023.02.20
# Usage:
#  Run the following on Linux or Mac:
#
#   curl -L https://raw.githubusercontent.com/oracle/coherence-cli/main/scripts/install.sh | bash
#
VERSION=1.4.2
OS=`uname`
ARCH=`uname -m`

function set_arch() {
    if [ "$ARCH" == "x86_64" ] ; then
        ARCH="amd64"
    elif [ "$ARCH" == "aarch64" -o "$ARCH" == "arm64" ] ; then
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

echo "Installing Coherence CLI ${VERSION} for ${OS}/${ARCH} into /usr/local/bin ..."

if [ "$OS" == "Darwin" ]; then
    set_arch
    PKG="Oracle-Coherence-CLI-${VERSION}-darwin-${ARCH}.pkg"
    echo "Downloading and opening /tmp/${PKG}"
    URL=https://github.com/oracle/coherence-cli/releases/download/${VERSION}/${PKG}
    curl -sLo /tmp/${PKG} $URL && open /tmp/${PKG} && installed
elif [ "$OS" == "Linux" ]; then
    set_arch
    echo "Using 'sudo' to mv cohctl binary to /usr/local/bin"
    URL=https://github.com/oracle/coherence-cli/releases/download/${VERSION}/cohctl-${VERSION}-linux-${ARCH}
    curl -sLo /tmp/cohctl $URL && chmod u+x /tmp/cohctl
    sudo mv /tmp/cohctl /usr/local/bin && installed
else
    echo "For all other platforms, please see: https://github.com/oracle/coherence-cli/releases"
    exit 1
fi
