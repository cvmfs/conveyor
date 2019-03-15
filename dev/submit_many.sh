#!/bin/sh

staging_server=$1
# Submit a set of independent jobs
ids=""
for i in $(seq 1 10) ; do
    id=$(./conveyor submit \
         --repo test-sw.hsf.org \
         --lease-path /ripgrep-0.$i.0 \
         --payload "script|${staging_server}/test_transaction.sh?checksum=sha1:86cd26b486a7439f2f1d0118de88021dcee0183a|${staging_server}/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz" | tail -1 | jq -r '.job_id')
    ids="$ids $id"
done
ids=$(echo $ids | tr ' ' ,)

# Submit a final job depending on all the previous ones
./conveyor submit --repo test-sw.hsf.org --deps "$ids" --wait
