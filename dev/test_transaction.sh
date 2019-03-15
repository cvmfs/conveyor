#!/bin/sh

set -e

# This is an example payload script to be run during a CernVM-FS transaction

# CernVM-FS repository
repository=$1
# Leased path inside the repository
lease_path=$2
# URL of the archive which will be unpacked
archive=$3

cd /cvmfs/$repository
mkdir -p $lease_path && cd $lease_path
curl -o payload.tar.gz $archive
tar xf payload.tar.gz
rm payload.tar.gz

