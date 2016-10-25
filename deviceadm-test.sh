#!/bin/bash
docker-compose run -T --name mender-device-auth mender-device-auth > /dev/null &
docker-compose run -T --name mender-inventory mender-inventory > /dev/null &
docker-compose run -T --name mender-mongo-device-adm mender-mongo-device-adm > /dev/null &
