#!/bin/sh

set -e

### Check script arguments

if [ x"$1" = x"" ]; then
  echo "Missing BUILD_LOCATION."
  exit 1
fi
if [ x"$2" = x"" ]; then
  echo "Missing VERSION."
  exit 1
fi
if [ x"$3" = x"" ]; then
  echo "Missing RELEASE."
  exit 1
fi

SCRIPT_LOCATION=$(realpath "$(dirname "$0")")
BUILD_LOCATION=$(realpath $1)
VERSION=$2
RELEASE=$3

PROJECT_NAME=conveyor

echo "Script location: ${SCRIPT_LOCATION}"
echo "Build location: ${BUILD_LOCATION}"
echo "Version: $VERSION"
echo "Release: $RELEASE"


###  Build package

echo "Building package"
cd ${BUILD_LOCATION}
export GOPATH=${GOPATH:=${BUILD_LOCATION}/../go}
go build

TARBALL_OUT=${BUILD_LOCATION}/TARBALLS
mkdir -p ${TARBALL_OUT}

PKG_WS=${BUILD_LOCATION}/pkg_ws
mkdir -p ${PKG_WS}

cp -v ${BUILD_LOCATION}/conveyor ${PKG_WS}/
cp -v ${BUILD_LOCATION}/config/config.toml ${PKG_WS}/config.toml.example
cp -v ${BUILD_LOCATION}/config/create_schema_postgres.sql ${PKG_WS}/
cp -v ${BUILD_LOCATION}/pkg/*.service ${PKG_WS}/

cd ${PKG_WS}
tar czf ${TARBALL_OUT}/conveyor-${CONVEYOR_VERSION}-${CONVEYOR_RELEASE}.Linux.x86_64.tar.gz ./*
cd ${BUILD_LOCATION}

### Clean up
rm -rf ${PKG_WS}
