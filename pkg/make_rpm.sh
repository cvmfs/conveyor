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


### Create togo project

echo "Creating togo project"
cd $BUILD_LOCATION
mkdir -p togo
cd togo
togo project create cvmfs-publisher-tools
TOGO_PROJECT=${BUILD_LOCATION}/togo/cvmfs-publisher-tools


###  Install Python package to togo project root

cd ${SCRIPT_LOCATION}/../client
python3 setup.py install --root ${TOGO_PROJECT}/root


### Go into togo project root
cd ${TOGO_PROJECT}


### Add other files to togo project root

mkdir -p ${TOGO_PROJECT}/root/etc/systemd/system
cp -v ${SCRIPT_LOCATION}/cvmfs-job-consume.service ${TOGO_PROJECT}/root/etc/systemd/system/
togo file exclude root/etc/systemd/system


### Configure the togo build

cp -v ${SCRIPT_LOCATION}/spec/* ${TOGO_PROJECT}/spec/
sed -i -e "s/<<CVMFS_PUBLISHER_TOOLS_VERSION>>/$VERSION/g" ${TOGO_PROJECT}/spec/header
sed -i -e "s/<<CVMFS_PUBLISHER_TOOLS_RELEASE>>/$RELEASE/g" ${TOGO_PROJECT}/spec/header


### Build the package

echo "Building RPM"
togo file exclude ${TOGO_PROJECT}/root/usr/bin
togo file exclude ${TOGO_PROJECT}/root/usr/lib/python3.4/site-packages
find ${TOGO_PROJECT}/root -name "*.pyo" -exec togo file unflag {} \;
togo build package


### Copy RPM and SRPM into place                                                                                                                                                    

echo "Copying RPMs to output location"                                                                                                                                            
mkdir -p $BUILD_LOCATION/RPMS                                                                                                                                                     
cp -v ./rpms/*.rpm $BUILD_LOCATION/RPMS                                                                                                                                           
cp -v ./rpms/src/*.rpm $BUILD_LOCATION/RPMS


### Clean up

rm -rf ${BUILD_LOCATION}/togo
