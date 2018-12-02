#!/bin/sh

# Submit a set of independent jobs
ids=""
for i in $(seq 1 10) ; do
    id=$(./cvmfs_job submit \
         --repo test-sw.hsf.org \
         --payload http://cvmfs-publisher-test.s3.cern.ch/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz \
         --path /ripgrep-0.$i.0 | tail -1 | jq -r .ID)
    ids="$ids $id"
done
ids=$(echo $ids | tr ' ' ,)

# Submit a final job depending on all the previous ones
./cvmfs_job submit --repo test-sw.hsf.org --deps "$ids" --script /usr/local/bin/list_all_versions.sh --wait
