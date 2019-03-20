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

## Submitting jobs

## Checking job status