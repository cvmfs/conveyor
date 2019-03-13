#!/bin/sh

set -e

SCRIPT_LOCATION=$(cd "$(dirname "$0")"; pwd)

if [ $# -lt 1 ]; then
  echo "Usage: $0 <Source directory (the same as the build directory)> [NIGHTLY_NUMBER]"
  echo "This script builds packages for the current platform."
  exit 1
fi

BUILD_LOCATION="$1"
shift 1

NIGHTLY_NUMBER=
if [ $# -gt 0 ]; then
  NIGHTLY_NUMBER=$1
  shift 1
fi

CONVEYOR_VERSION=$(cat ${SCRIPT_LOCATION}/../version)

PACKAGE_VERSION=1
if [ ! -z "$NIGHTLY_NUMBER" ]; then
    PACKAGE_VERSION=0.$NIGHTLY_NUMBER
fi

# Create an RPM
${SCRIPT_LOCATION}/../pkg/make_rpm.sh ${BUILD_LOCATION} ${CONVEYOR_VERSION} ${PACKAGE_VERSION}
${SCRIPT_LOCATION}/../pkg/make_tarball.sh ${BUILD_LOCATION} ${CONVEYOR_VERSION} ${PACKAGE_VERSION}