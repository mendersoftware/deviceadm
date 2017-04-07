#!/bin/bash

# tests are supposed to be located in the same directory as this file
DIR=$(readlink -f $(dirname $0))

export PYTHONDONTWRITEBYTECODE=1

HOST=${HOST="mender-device-adm:8080"}
DEVICEAUTH_HOST=${DEVICEAUTH_HOST="mender-device-auth:8080"}

# if we're running in a container, wait a little before starting tests
[ $$ -eq 1 ] && {
    echo "-- running in container, wait for other services"
    sleep 10
}

py.test-3 -s --tb=short --api=0.1.0  --host $HOST \
          --devauth-host $DEVICEAUTH_HOST \
          --devices 101 \
          --management-spec $DIR/management_api.yml \
          --internal-spec $DIR/internal_api.yml \
          --verbose --junitxml=$DIR/results.xml \
          $DIR/tests/test_*.py "$@"
