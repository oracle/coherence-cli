#!/bin/bash

#
# Copyright (c) 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

# Check a docker image exists and if not, pull it

if [ $# -ne 1 ] ; then
   echo "Please supply an image"
   exit 1
fi

docker inspect $1 > /dev/null 2>&1
ret=$?

if [ $ret -ne 0 ] ; then
  docker pull $1
else
  echo "Image $1 is already present"
fi