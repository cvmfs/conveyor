#!/bin/sh

repo="test-sw.hsf.org"

for i in $(seq 1 10) ; do
    payload="http://cvmfs-publisher-test.s3.cern.ch/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz"
    path="/ripgrep-0.$i.0"
    id=$(./cvmfs_job submit \
        --repo test-sw.hsf.org \
        --payload http://cvmfs-publisher-test.s3.cern.ch/ripgrep/ripgrep-0.$i.0-x86_64-unknown-linux-musl.tar.gz \
        --path /ripgrep-0.$i.0 | tail -1 | jq -r '.ID')
    psql -h localhost -U postgres -d devcvmfs -w \
        -c "insert into jobs (ID,Repository,Payload,RepositoryPath,Script,ScriptArgs,RemoteScript,Dependencies,StartTime,FinishTime,Successful,ErrorMessage) values ('$id','$repo','$payload','$path', '', '', false, '', '2018-11-20 12:00:00', '2018-11-20 12:00:01', true, '');"
done
