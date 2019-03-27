#!/bin/sh

#################################################
#   Run the functional/integration test suite   #
#################################################

SCRIPT_LOCATION=$(realpath "$(dirname "$0")")
cd $SCRIPT_LOCATION

# Create a workspace for the test run
CONVEYOR_TEST_WORKSPACE=`mktemp -d /tmp/conveyor-tests.XXXXXXXXX`
echo "Temp dir created: $CONVEYOR_TEST_WORKSPACE"
export CONVEYOR_TEST_WORKSPACE

# Create a shared area to be mounted into the gateway container
# The gateway container startup script writes the newly created
# repository public and gateway keys into this area, to be used
# by the publisher containers running Conveyor workers
mkdir -p $CONVEYOR_TEST_WORKSPACE/gateway
chmod g+s $CONVEYOR_TEST_WORKSPACE/gateway

# Start containers with the backing services (PostgreSQL, RabbitMQ, Minio, cvmfs-gateway)\
docker-compose up -d

# Start a container with conveyor-server

# Start multiple conveyor-worker containers

# Submit test jobs

# Cleanup
cleanup () {
    cd $SCRIPT_LOCATION
    docker-compose down
    rm -rf $CONVEYOR_TEST_WORKSPACE
}
trap cleanup EXIT HUP INT TERM || return $?