# User guide (Work in progress)

## Overview

Conveyor is a set of tools that work on top of an existing CernVM-FS publication infrastructure to provide a higher-level interface: users only need to describe what needs to be published, while Conveyor takes care of how the publication is done.
It can schedule jobs across multiple CernVM-FS publisher machines, increasing the throughput of the publication infrastructure.

A Conveyor system is made up of multiple components, implemented as subcommands of the `conveyor` program:

1. **The client tools**

     The user can submit jobs to a Conveyor system using the `conveyor submit` command.
     This is done from a machine that does not have the CernVM-FS server tools installed, such a CI builder node.

     By default, job submission is done asynchronously, but it is also possible to wait until a submitted job is processed.

     A submitted job is identified by a unique identifier (UUID) which can be used to query the status of the job with the `conveyor check` command.
     UUIDs can also be used to define job dependencies - a job will be processed before the other jobs it depends on.

1. **The job server**

     The different parts of a Conveyor system coordinate through the job server (started with `conveyor server`).
     This long-running process can be executed on a separate machine, but can also be colocated with the `cvmfs-gateway` application.

     It is responsible with receiving job submissions and queries from clients, and maintains a database with the status of all processed jobs.

     A RabbitMQ broker serves as main communication channel between the various producer and consumer parties of the system.
     Once it has validated a job submission, the server will push the job description to the message broker, making it available to interested workers.
     The message broker also manages a queue where the status of completed jobs is published.

1. **The worker daemon**

     The worker daemon (started with `conveyor worker`) runs on CernVM-FS
     publisher nodes, and is the only component that interacts with CernVM-FS.

     It connects to the message broker and consumes the messages containing new job descriptions.
     It is responsible for opening CernVM-FS transactions and dowloading payloads according to the description of the job.
     The job payload is typically a shell script that is run inside the writable area of the CernVM-FS repository during an open transaction.
     Depending on the success of the script, the Conveyor worker will either publish the transaction or abort it.
     It will then publish the status of the completed job (either successful or failed) to the job server.

## Setup and configuration

Conveyor makes use of CernVM-FS server tools, and is deployed on an existing infrastructure containing a CernVM-FS repository gateway machine with multiple publisher machines connected to it.

Conveyor is distributed as a single x86_64 binary package (either an RPM for CentOS 7 or a binary tarball), available in the [CernVM-FS download area](https://cernvm.web.cern.ch/portal/filesystem/downloads).
The tarballs has no installation scripts for configuration and systemd service files.

Once the package has been installed on all the machines, some third-party services need to be configured, before writing the configuration files.

### RabbitMQ

RabbitMQ is needed by all the Conveyor tools.
The RabbitMQ server (broker) can be run on a dedicated machine, or can be colocated on the repository gateway machine for convenience.
The version of `rabbitmq-server` from the CentOS 7 repository can be installed.
The management plugin provides a web dashboard for the administration of a RabbitMQ broker.
It is recommended to create a dedicated, non-administrator, RabbitMQ user for use by Conveyor.
An example configuration script for RabbitMQ is [included in this repository](https://github.com/cvmfs/conveyor/blob/master/setup/configure_rabbitmq.sh).

### PostgreSQL

An SQL database (either PostgreSQL > 9.5 or MySQL) is required by the Conveyor server.
PostgreSQL is recommended and recent packages can be downloaded from the [PostgreSQL website](https://postgresql.org/download).

CERN also offers PostgreSQL instances through the [Database on Demand](http://information-technology.web.cern.ch/services/database-on-demand) service.

The database schema can be create with the [provided script](https://github.com/cvmfs/conveyor/blob/master/config/create_schema_postgres.sql).

### Conveyor configuration

Conveyor uses a single configuration file located by default at `/etc/cvmfs/conveyor/config.toml`.
Another configuration file can be specified with the `--config` parameter.

Certain subsections of the configuration file are only needed for the `conveyor server` or on the `conveyor worker` commands.

#### Credentials

Credentials for RabbitMQ, the SQL database and the shared secret key can be omitted from the configuration file and be provided through environment varibles.
Including them in the configuration file is convenient for daemons (`conveyor server|worker`), while for the client tools (`conveyor submit|check`), usually run from a CI node, it is convenient to inject these secrets into the environment of a CI job.

The following environment variables are used:

* `CONVEYOR_SHARED_KEY`
* `CONVEYOR_QUEUE_USER`
* `CONVEYOR_QUEUE_PASS`
* `CONVEYOR_DB_USER`
* `CONVEYOR_DB_USER`

Values provided through environment variables override the ones from the configuration file.

#### Global parameters

Required by all commands.

* `shared_key` - (string) This secret is shared between all participants (client, server, and worker) to sign and verify HTTP requests between them
* `job_wait_timeout` - (integer) Maximum number of seconds a publication job is allowed to take. Default to 7200s

#### [server]

Required by all commands.

* `host` - (string) URL of the Conveyor server
* `port` - (int) Port on which the Conveyor server is running. Default is 8080.

#### [queue]

Required by all commands.

* `username` - (string) RabbitMQ user name
* `password` - (string) RabbitMQ user password
* `host` - (string) URL of the RabbitMQ broker
* `port` - (int) Port used by the RabbitMQ broker. Defaults to 5672
* `vhost` - (string) Virtual host configured in the broker. Defaults to "/".

#### [db]

Only required by `conveyor server`.

* `type` - (string) Type of SQL database. Can be `postgres` or `mysql`.
* `database` - (string) Database name
* `username` - (string) Database user name
* `password` - (string) Database pass word
* `host` - (string) URL of the database instance
* `port` - (int) Port used by the database. Defaults to 5432

#### [worker]

Only required by `conveyor worker`.

* `name` - (string) A name to identify the worker. It defaults to the hostname and there is not check for uniqueness among multiple workers connected to the same server.
* `job_retries` - (int) The number of times a failing job is retried. Default is 3.
* `temp_dir` - (string) Temporary directory where payload scripts are downloaded during transactions. Default is `/tmp/conveyor-worker`.

## Submitting jobs

Jobs can be submitted with the `conveyor submit` command which takes the following parameters:

* `--repo` - (string) The target CVMFS repository
* `--lease-path` (string, optional) Repository subpath to be leased for the duration of the job.
By default, a lease is requested on the entire repository (`"/"`)
* `--job-name` - (string, optional) name of the job
* `--payload` - (string, optional) URL of the job payload (see next subsection for a description of the payload specification).
When no payload is specified, the job corresponds to an empty CernVM-FS transaction.
* `--deps` - (string, optional) comma-separated list of job dependency UUIDs
* `--wait` (optional) - wait for completion of the submitted job

By default, jobs are submitted asynchronously.
An UUID is assigned to a job when it is submitted, and can be used to query the status of the job with the `conveyor check` command, or list the job as a dependency of another job.

### Job payload

The payload of a job is given as a string with the format:

```
"<PAYLOAD_TYPE>|<PAYLOAD_URL>[|<ARGUMENT>]"
```

* `<PAYLOAD_TYPE>` identifies the type of payload.
Currently, only the `script` payload has been implemented, but multiple types of payload will be added in the future.
* `<PAYLOAD_URL>` is the URL of a payload file which will be downloaded and executed by Conveyor when processing the job.
An optional checksum can be provided as a query parameter:
    ```
    http://conveyor-payloads.s3.cern.ch/task.sh?checksum=sha1:6a5f9462608383fb65e6c0f7211148974bdbdc3d
    ```
The payload script is called with the repository name and the leased path as first and second arguments, respectively.
* `<ARGUMENT>` is an optional third argument for the payload script.

The following is an example payload script which downloads and unpacks an archive into a repository subpath:

```bash
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

repository_root=/cvmfs/${repository}

echo "- Creating leased path, if needed"

mkdir -p ${repository_root}/$lease_path
cd ${repository_root}/$lease_path

echo "- Downloading archive"

curl -o payload.tar.gz $archive

echo "- Unpacking archive"

tar xfv payload.tar.gz

echo "- Cleaning up"

rm -v payload.tar.gz
```

## Checking job status

The status of one or multiple submitted jobs can be queried with the `conveyor check` command:

* `--ids` (string) A comma-separate list of job UUIDs to query
* `--full-status` (optional) Return the full status of the job.
By default, only the success status of the job is returned.
* `--wait` (optional) Wait for completion of the queried jobs

The full list of fields of job status is:

* `ID`
* `JobName`
* `Repository`
* `Payload`
* `LeasePath`
* `Dependencies`
* `WorkerName`
* `StartTime`
* `FinishTime`
* `Successful`
* `ErrorMessage`