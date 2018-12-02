#!/bin/sh

repo=$1
repo_path=$2

find /cvmfs/$repo -name "ripgrep-*-musl" > /cvmfs/$repo/versions.txt
