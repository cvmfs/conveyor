#!/bin/sh

set -e

# This is an example payload script to be run during a CernVM-FS transaction

# CernVM-FS repository
repository=$1
# Leased path inside the repository
lease_path=$2
# URL of the archive which will be unpacked
archive=$3

echo "Running CernVM-FS transaction"
echo "  - Repository: $repository"
echo "  - Lease path: $lease_path"
echo "  - Archive:    $archive"

echo "- Changing directory to the root of the repository"

cd /cvmfs/$repository

echo "- Creating leased path, if needed"

mkdir -p $lease_path && cd $lease_path

echo "- Downloading archive"

curl -o payload.tar.gz $archive

echo "- Unpacking archive"

tar xfv payload.tar.gz

echo "- Cleaning up"

rm -v payload.tar.gz

