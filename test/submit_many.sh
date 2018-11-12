#!/bin/sh

for i in $(seq 1 10) ; do
    ./src/cvmfs_job submit --path /ripgrep-0.$i.0 test-sw.hsf.org http://cvmfs-publisher-test.s3.cern.ch/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz
done
