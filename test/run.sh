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

# Start containers with the backing services (PostgreSQL, RabbitMQ)
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