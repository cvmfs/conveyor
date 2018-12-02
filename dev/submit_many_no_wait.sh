#!/bin/sh

# Submit a set of independent jobs
for i in $(seq 1 10) ; do
    ./cvmfs_job submit \
        --repo test-sw.hsf.org \
        --payload http://cvmfs-publisher-test.s3.cern.ch/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz \
        --path /ripgrep-0.$i.0 | tail -1 | jq
done

