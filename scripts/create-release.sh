#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Create a new release tag and push the tag. This script will ensure that the
# Tag trying to be created will match the values in the Makefile
#
# Must be run from root directory:
# ./scripts/create-release.sh tag

set -e

if [ $# -ne 1 ] ; then
   echo "Usage: $0 tag"
   exit 1
fi

TAG=$1

# validate that the Makefile contains the correct release values

MAKEFILE_VERSION=`grep "VERSION ?=" Makefile | grep -v MVN | grep -v COHERENCE | sed 's/^.*?= //'`
MAKEFILE_MILESTONE=`grep "MILESTONE ?=" Makefile | sed -e 's/^.*?=//' -e s'/ //g'`
MAKEFILE_FINAL_VERSION=${MAKEFILE_VERSION}${MAKEFILE_MILESTONE}

echo "Makefile version:       $MAKEFILE_VERSION"
echo "Makefile milestone:     $MAKEFILE_MILESTONE"
echo "Makefile final version: $MAKEFILE_FINAL_VERSION"
echo "Provided Tag:           $TAG"

if [ "$TAG" != "$MAKEFILE_FINAL_VERSION" ] ; then
    echo "The final version in the Makefile of ${MAKEFILE_FINAL_VERSION} is different than the tag ${TAG}"
    echo "You must ensure they are the same and the change has been pushed"
    exit 1
fi


# check to see if there are any outstanding commits/ changes
UNTRACKED=`git status| grep 'Untracked files' || true`
AHEAD=`git status| grep 'Your branch is ahead' || true`
NEW=`git status| grep 'new file' || true`
MODIFIED=`git status| grep 'modified:' || true`

if [ ! -z "$UNTRACKED" -o ! -z "$NEW" -o ! -z "$MODIFIED" ] ; then
    echo "You have uncommitted changes. Please complete these and push before running this script again"
    exit 1
fi

if [ ! -z "$AHEAD" ] ; then
    echo "Please push any committed changes before running this script again"
    exit 1
fi

echo
echo "Makefile and tag match"
echo "WARNING: You are about to create a release."
echo
echo -n "Are you sure you want to create a release with tag $TAG ? (y/n): "
read ans

if [ "$ans" != "y" ] ; then
    echo "No changes carried out"
else
    echo "Creating and pushing tag $TAG"
    git tag $TAG
    git push origin $TAG

    echo
    echo "Please check https://github.com/oracle/coherence-cli/actions"
fi
