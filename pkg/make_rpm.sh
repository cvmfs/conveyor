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

echo "Script location: ${SCRIPT_LOCATION}"
echo "Build location: ${BUILD_LOCATION}"
echo "Version: $VERSION"
echo "Release: $RELEASE"


###  Build package (with GOPATH inside project)

echo "Building package"
mkdir -p ${BUILD_LOCATION}/gopath/src/github.com/cvmfs
ln -fs ${BUILD_LOCATION} ${BUILD_LOCATION}/gopath/src/github.com/cvmfs/${PROJECT_NAME}
export GOPATH="${BUILD_LOCATION}/gopath"
cd ${BUILD_LOCATION}
make clean && make


### Create togo project

echo "Creating togo project"
PROJECT_NAME=cvmfs-publisher-tools
cd $BUILD_LOCATION
mkdir -p togo
cd togo
togo project create ${PROJECT_NAME}
TOGO_PROJECT=${BUILD_LOCATION}/togo/${PROJECT_NAME}


### Go into togo project root
cd ${TOGO_PROJECT}


### Install executable(s) to togo root
mkdir -p root/usr/bin
cp -v ${BUILD_LOCATION}/cvmfs_job root/usr/bin/
togo file exclude root/usr/bin


### Add other files to togo project root

mkdir -p ${TOGO_PROJECT}/root/etc/systemd/system
cp -v ${SCRIPT_LOCATION}/cvmfs-job-consumer.service ${TOGO_PROJECT}/root/etc/systemd/system/
cp -v ${SCRIPT_LOCATION}/cvmfs-job-server.service ${TOGO_PROJECT}/root/etc/systemd/system/
togo file flag config-nr root/etc/systemd/system/cvmfs-job-consumer.service
togo file flag config-nr root/etc/systemd/system/cvmfs-job-server.service
togo file exclude root/etc/systemd/system

mkdir -p ${TOGO_PROJECT}/root/etc/cvmfs/publisher
cp -v ${BUILD_LOCATION}/config/config.toml ${TOGO_PROJECT}/root/etc/cvmfs/publisher/
togo file flag config-nr root/etc/cvmfs/publisher/config.toml
togo file exclude root/etc/cvmfs


### Configure the togo build

cp -v ${SCRIPT_LOCATION}/spec/* ${TOGO_PROJECT}/spec/
sed -i -e "s/<<CVMFS_PUBLISHER_TOOLS_VERSION>>/$VERSION/g" ${TOGO_PROJECT}/spec/header
sed -i -e "s/<<CVMFS_PUBLISHER_TOOLS_RELEASE>>/$RELEASE/g" ${TOGO_PROJECT}/spec/header


### Build the package

echo "Building RPM"
togo build package


### Copy RPM and SRPM into place
echo "Copying RPMs to output location"
mkdir -p $BUILD_LOCATION/RPMS
cp -v ./rpms/*.rpm $BUILD_LOCATION/RPMS
cp -v ./rpms/src/*.rpm $BUILD_LOCATION/RPMS


### Clean up

rm -rf ${BUILD_LOCATION}/togo
rm -rf ${BUILD_LOCATION}/gopath
