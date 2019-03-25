# Conveyor

[![build status](https://travis-ci.org/cvmfs/conveyor.svg?branch=master)](https://travis-ci.org/cvmfs/conveyor)

A higher-level, job-based, interface for publishing to CernVM-FS repositories.

## Features:

* *High-level:* work in terms of jobs - no need to interact with the CernVM-FS server tools directly, no need to have CernVM-FS installed for Conveyor client operations
* *Declarative workflow:* describe what needs to be published, and Conveyor takes care of how the publication is done
* *Scalable:* schedule jobs onto multiple CernVM-FS publisher machines, for increased throughput

## Getting started

Go 1.11 or newer is required to build Conveyor:

```bash
$ go build
$ go test -v ./...
```

Please see the [user guide](https://github.com/cvmfs/conveyor/blob/master/doc/user_guide.md) for detailed configuration and usage instructions.

# License and copyright

See LICENSE and AUTHORS in the project root.
