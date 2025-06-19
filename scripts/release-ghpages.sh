#!/bin/bash
#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Release the Coherence CLI gh-pages
set -e

if [ $# -ne 2 ] ; then
  echo "Usage: $0 version build-output"
  exit 1
fi

DIR=`pwd`

export VERSION=$1
export BUILD_OUTPUT=$2

echo "Version:      ${VERSION}"
echo "Build Output: ${BUILD_OUTPUT}"

[ ! -d ${BUILD_OUTPUT} ] && echo "Cannot find $BUILD_OUTPUT" && exit 1

export CLI_TMP=/tmp/coherence-cli

rm -rf ${CLI_TMP} || true
mkdir -p ${CLI_TMP} || true
cp -R ${BUILD_OUTPUT} ${CLI_TMP}

ls -l ${CLI_TMP}

git stash save --keep-index --include-untracked || true
git stash drop || true
git checkout --track origin/gh-pages
git config pull.rebase true
git pull

pwd

if [ -z "`echo $VERSION | grep RC`" ] ; then
    # Proper release so update stable.txt
    echo $VERSION > stable.txt

    # Update latest docs
    git add -A stable.txt
else
    # Must be RC
    git add -A docs/${VERSION}/*
fi

git status
git clean -d -f
git status
git commit -m "Release Coherence CLI version: ${VERSION}"
git log -1
git push origin gh-pages
