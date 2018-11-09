#!/bin/sh

set -e

script_name=$(readlink -f $0)
script_location=$(dirname $script_name)
 ${script_location}/config/cvmfs-job-consume.service /etc/systemd/system/
echo systemctl daemon-reload