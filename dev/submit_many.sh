#!/bin/sh

staging_server=$1
# Submit a set of independent jobs
ids=""
for i in $(seq 1 10) ; do
    id=$(./conveyor submit \
         --repo test-sw.hsf.org \
         --lease-path /ripgrep-0.$i.0 \
         --payload "script|${staging_server}/test_transaction.sh?checksum=sha1:2e657ec8cb1ae02892e6b9d388a1affd206d6d6c|${staging_server}/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz" | tail -1 | jq -r '.job_id')
    ids="$ids $id"
done
ids=$(echo $ids | tr ' ' ,)

# Submit a final job depending on all the previous ones
./conveyor submit --repo test-sw.hsf.org --deps "$ids" --wait
